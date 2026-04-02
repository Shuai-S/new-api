package openai

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func sendResponsesCompatEvent(c *gin.Context, event *dto.ResponsesStreamResponse) error {
	if event == nil {
		return nil
	}
	data, err := common.Marshal(event)
	if err != nil {
		return err
	}
	helper.ResponseChunkData(c, *event, string(data))
	return nil
}

func OaiChatToResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	var chatResp dto.OpenAITextResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	if err := common.Unmarshal(responseBody, &chatResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := chatResp.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	if chatResp.Usage.PromptTokens == 0 {
		completionTokens := chatResp.Usage.CompletionTokens
		if completionTokens == 0 {
			for _, choice := range chatResp.Choices {
				completionTokens += service.CountTextToken(choice.Message.StringContent()+choice.Message.ReasoningContent+choice.Message.Reasoning, info.UpstreamModelName)
			}
		}
		chatResp.Usage = dto.Usage{
			PromptTokens:     info.GetEstimatePromptTokens(),
			CompletionTokens: completionTokens,
			TotalTokens:      info.GetEstimatePromptTokens() + completionTokens,
		}
	}

	applyUsagePostProcessing(info, &chatResp.Usage, responseBody)

	responseID := chatResp.Id
	if responseID == "" {
		responseID = helper.GetResponseID(c)
	}

	responsesResp, usage, err := service.ChatCompletionsResponseToResponsesResponse(&chatResp, responseID)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if usage == nil {
		usage = &chatResp.Usage
	}

	convertedBody, err := common.Marshal(responsesResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}

	service.IOCopyBytesGracefully(c, resp, convertedBody)
	return usage, nil
}

type chatStreamToolCallState struct {
	OutputIndex int
	ID          string
	Name        string
	Arguments   strings.Builder
	Added       bool
}

func OaiChatToResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	defer service.CloseResponseBodyGracefully(resp)

	responseID := helper.GetResponseID(c)
	createAt := int64(0)
	model := info.UpstreamModelName
	messageItemID := fmt.Sprintf("%s-msg-0", responseID)

	var (
		usage            = &dto.Usage{}
		outputText       strings.Builder
		usageText        strings.Builder
		reasoningText    strings.Builder
		sentCreated      bool
		sentMessageAdded bool
		streamErr        *types.NewAPIError
	)

	toolStates := make(map[int]*chatStreamToolCallState)

	sendCreatedIfNeeded := func() bool {
		if sentCreated {
			return true
		}
		statusRaw, _ := common.Marshal("in_progress")
		event := &dto.ResponsesStreamResponse{
			Type: "response.created",
			Response: &dto.OpenAIResponsesResponse{
				ID:        responseID,
				Object:    "response",
				CreatedAt: int(createAt),
				Model:     model,
				Status:    statusRaw,
				Output:    make([]dto.ResponsesOutput, 0),
			},
		}
		if err := sendResponsesCompatEvent(c, event); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		sentCreated = true
		return true
	}

	sendMessageAddedIfNeeded := func() bool {
		if sentMessageAdded {
			return true
		}
		if !sendCreatedIfNeeded() {
			return false
		}
		idx := 0
		event := &dto.ResponsesStreamResponse{
			Type:        dto.ResponsesOutputTypeItemAdded,
			OutputIndex: &idx,
			Item: &dto.ResponsesOutput{
				Type:   "message",
				ID:     messageItemID,
				Status: "in_progress",
				Role:   "assistant",
				Content: []dto.ResponsesOutputContent{
					{
						Type:        "output_text",
						Text:        "",
						Annotations: make([]interface{}, 0),
					},
				},
			},
		}
		if err := sendResponsesCompatEvent(c, event); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		sentMessageAdded = true
		return true
	}

	sendToolCallAddedIfNeeded := func(state *chatStreamToolCallState) bool {
		if state == nil || state.Added {
			return true
		}
		if !sendCreatedIfNeeded() {
			return false
		}
		event := &dto.ResponsesStreamResponse{
			Type:        dto.ResponsesOutputTypeItemAdded,
			OutputIndex: &state.OutputIndex,
			Item: &dto.ResponsesOutput{
				Type:      "function_call",
				ID:        state.ID,
				Status:    "in_progress",
				CallId:    state.ID,
				Name:      state.Name,
				Arguments: state.Arguments.String(),
			},
		}
		if err := sendResponsesCompatEvent(c, event); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		state.Added = true
		return true
	}

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		if streamErr != nil {
			sr.Stop(streamErr)
			return
		}

		var streamResp dto.ChatCompletionsStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResp); err != nil {
			sr.Error(err)
			return
		}

		if streamResp.Id != "" {
			responseID = streamResp.Id
			if !sentMessageAdded {
				messageItemID = fmt.Sprintf("%s-msg-0", responseID)
			}
		}
		if streamResp.Created != 0 {
			createAt = streamResp.Created
		}
		if streamResp.Model != "" {
			model = streamResp.Model
		}
		if streamResp.Usage != nil {
			usage = streamResp.Usage
		}

		if len(streamResp.Choices) == 0 {
			return
		}

		choice := streamResp.Choices[0]
		if content := choice.Delta.GetContentString(); content != "" {
			if !sendMessageAddedIfNeeded() {
				sr.Stop(streamErr)
				return
			}
			outputText.WriteString(content)
			usageText.WriteString(content)
			outputIndex := 0
			contentIndex := 0
			event := &dto.ResponsesStreamResponse{
				Type:         "response.output_text.delta",
				Delta:        content,
				ItemID:       messageItemID,
				OutputIndex:  &outputIndex,
				ContentIndex: &contentIndex,
			}
			if err := sendResponsesCompatEvent(c, event); err != nil {
				streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
				sr.Stop(streamErr)
				return
			}
		}

		if reasoning := choice.Delta.GetReasoningContent(); reasoning != "" {
			if !sendCreatedIfNeeded() {
				sr.Stop(streamErr)
				return
			}
			reasoningText.WriteString(reasoning)
			event := &dto.ResponsesStreamResponse{
				Type:  "response.reasoning_summary_text.delta",
				Delta: reasoning,
			}
			if err := sendResponsesCompatEvent(c, event); err != nil {
				streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
				sr.Stop(streamErr)
				return
			}
		}

		for _, toolCall := range choice.Delta.ToolCalls {
			index := 0
			if toolCall.Index != nil {
				index = *toolCall.Index
			}

			state, ok := toolStates[index]
			if !ok {
				outputIndex := len(toolStates)
				if sentMessageAdded {
					outputIndex++
				}
				state = &chatStreamToolCallState{
					OutputIndex: outputIndex,
					ID:          fmt.Sprintf("%s-call-%d", responseID, index),
				}
				toolStates[index] = state
			}

			if toolCall.ID != "" {
				state.ID = toolCall.ID
			}
			if toolCall.Function.Name != "" {
				state.Name = toolCall.Function.Name
			}
			if toolCall.Function.Arguments != "" {
				state.Arguments.WriteString(toolCall.Function.Arguments)
				usageText.WriteString(toolCall.Function.Name)
				usageText.WriteString(toolCall.Function.Arguments)
			}

			if !sendToolCallAddedIfNeeded(state) {
				sr.Stop(streamErr)
				return
			}
			if toolCall.Function.Arguments != "" {
				event := &dto.ResponsesStreamResponse{
					Type:        "response.function_call_arguments.delta",
					Delta:       toolCall.Function.Arguments,
					ItemID:      state.ID,
					OutputIndex: &state.OutputIndex,
				}
				if err := sendResponsesCompatEvent(c, event); err != nil {
					streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
					sr.Stop(streamErr)
					return
				}
			}
		}
	})

	if streamErr != nil {
		return nil, streamErr
	}

	if usage.TotalTokens == 0 {
		sourceText := usageText.String()
		if sourceText == "" {
			sourceText = outputText.String()
		}
		if sourceText != "" {
			usage = service.ResponseText2Usage(c, sourceText, info.UpstreamModelName, info.GetEstimatePromptTokens())
		}
	}

	if !sentCreated {
		if !sendCreatedIfNeeded() {
			return nil, streamErr
		}
	}

	if sentMessageAdded {
		outputIndex := 0
		doneEvent := &dto.ResponsesStreamResponse{
			Type:        dto.ResponsesOutputTypeItemDone,
			OutputIndex: &outputIndex,
			Item: &dto.ResponsesOutput{
				Type:   "message",
				ID:     messageItemID,
				Status: "completed",
				Role:   "assistant",
				Content: []dto.ResponsesOutputContent{
					{
						Type:        "output_text",
						Text:        outputText.String(),
						Annotations: make([]interface{}, 0),
					},
				},
			},
		}
		if err := sendResponsesCompatEvent(c, doneEvent); err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}
	}

	indices := make([]int, 0, len(toolStates))
	for idx := range toolStates {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	toolCalls := make([]dto.ToolCallRequest, 0, len(indices))
	for _, idx := range indices {
		state := toolStates[idx]
		toolCalls = append(toolCalls, dto.ToolCallRequest{
			ID:   state.ID,
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      state.Name,
				Arguments: state.Arguments.String(),
			},
		})

		doneEvent := &dto.ResponsesStreamResponse{
			Type:        dto.ResponsesOutputTypeItemDone,
			OutputIndex: &state.OutputIndex,
			Item: &dto.ResponsesOutput{
				Type:      "function_call",
				ID:        state.ID,
				Status:    "completed",
				CallId:    state.ID,
				Name:      state.Name,
				Arguments: state.Arguments.String(),
			},
		}
		if err := sendResponsesCompatEvent(c, doneEvent); err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}
	}

	message := dto.Message{
		Role:    "assistant",
		Content: outputText.String(),
	}
	if len(toolCalls) > 0 {
		message.SetToolCalls(toolCalls)
	}

	chatResp := &dto.OpenAITextResponse{
		Id:      responseID,
		Object:  "chat.completion",
		Created: createAt,
		Model:   model,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index:   0,
				Message: message,
			},
		},
		Usage: *usage,
	}

	if reasoningText.Len() > 0 && len(chatResp.Choices) > 0 {
		chatResp.Choices[0].Message.ReasoningContent = reasoningText.String()
	}

	responsesResp, usage, err := service.ChatCompletionsResponseToResponsesResponse(chatResp, responseID)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	completedEvent := &dto.ResponsesStreamResponse{
		Type:     "response.completed",
		Response: responsesResp,
	}
	if err := sendResponsesCompatEvent(c, completedEvent); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	return usage, nil
}
