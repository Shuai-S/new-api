package openaicompat

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func createdAtToInt(v any) int {
	switch vv := v.(type) {
	case int:
		return vv
	case int64:
		return int(vv)
	case float64:
		return int(vv)
	case json.Number:
		if i, err := vv.Int64(); err == nil {
			return int(i)
		}
	case string:
		return common.String2Int(vv)
	}
	return 0
}

func chatUsageToResponsesUsage(usage dto.Usage) *dto.Usage {
	out := usage
	if out.InputTokens == 0 {
		out.InputTokens = out.PromptTokens
	}
	if out.OutputTokens == 0 {
		out.OutputTokens = out.CompletionTokens
	}
	if out.TotalTokens == 0 {
		out.TotalTokens = out.InputTokens + out.OutputTokens
	}
	if out.InputTokensDetails == nil {
		out.InputTokensDetails = &dto.InputTokenDetails{
			CachedTokens: out.PromptTokensDetails.CachedTokens,
			ImageTokens:  out.PromptTokensDetails.ImageTokens,
			AudioTokens:  out.PromptTokensDetails.AudioTokens,
			TextTokens:   out.PromptTokens - out.PromptTokensDetails.CachedTokens - out.PromptTokensDetails.ImageTokens - out.PromptTokensDetails.AudioTokens,
		}
		if out.InputTokensDetails.TextTokens < 0 {
			out.InputTokensDetails.TextTokens = 0
		}
	}
	if out.CompletionTokenDetails.TextTokens == 0 {
		out.CompletionTokenDetails.TextTokens = out.CompletionTokens - out.CompletionTokenDetails.AudioTokens - out.CompletionTokenDetails.ReasoningTokens
		if out.CompletionTokenDetails.TextTokens < 0 {
			out.CompletionTokenDetails.TextTokens = 0
		}
	}
	return &out
}

func ChatCompletionsResponseToResponsesResponse(resp *dto.OpenAITextResponse, id string) (*dto.OpenAIResponsesResponse, *dto.Usage, error) {
	if resp == nil {
		return nil, nil, errors.New("response is nil")
	}

	responseID := id
	if responseID == "" {
		responseID = resp.Id
	}
	if responseID == "" {
		responseID = "response"
	}

	if len(resp.Choices) == 0 {
		return nil, nil, errors.New("choices is empty")
	}

	choice := resp.Choices[0]
	output := make([]dto.ResponsesOutput, 0, 2)
	contentText := choice.Message.StringContent()
	if contentText != "" {
		output = append(output, dto.ResponsesOutput{
			Type:   "message",
			ID:     fmt.Sprintf("%s-msg-0", responseID),
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{
				{
					Type:        "output_text",
					Text:        contentText,
					Annotations: make([]interface{}, 0),
				},
			},
		})
	}

	toolCalls := choice.Message.ParseToolCalls()
	for idx, toolCall := range toolCalls {
		callID := toolCall.ID
		if callID == "" {
			callID = fmt.Sprintf("%s-call-%d", responseID, idx)
		}
		output = append(output, dto.ResponsesOutput{
			Type:      "function_call",
			ID:        callID,
			Status:    "completed",
			CallId:    callID,
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
		})
	}

	statusRaw, _ := common.Marshal("completed")
	usage := chatUsageToResponsesUsage(resp.Usage)
	response := &dto.OpenAIResponsesResponse{
		ID:                responseID,
		Object:            "response",
		CreatedAt:         createdAtToInt(resp.Created),
		Status:            statusRaw,
		Model:             resp.Model,
		Output:            output,
		ParallelToolCalls: len(toolCalls) > 1,
		Usage:             usage,
	}
	return response, usage, nil
}
