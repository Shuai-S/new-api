package openaicompat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func rawMessageToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if common.GetJsonType(raw) == "string" {
		var s string
		if err := common.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	return strings.TrimSpace(string(raw))
}

func convertResponsesTextFormatToChatResponseFormat(textRaw json.RawMessage) *dto.ResponseFormat {
	if len(textRaw) == 0 || common.GetJsonType(textRaw) != "object" {
		return nil
	}

	var textCfg map[string]any
	if err := common.Unmarshal(textRaw, &textCfg); err != nil {
		return nil
	}

	formatMap, ok := textCfg["format"].(map[string]any)
	if !ok {
		return nil
	}

	formatType := strings.TrimSpace(common.Interface2String(formatMap["type"]))
	if formatType == "" {
		return nil
	}

	respFormat := &dto.ResponseFormat{Type: formatType}
	if formatType == "json_schema" {
		jsonSchema := make(map[string]any)
		for key, value := range formatMap {
			if key == "type" {
				continue
			}
			jsonSchema[key] = value
		}
		if len(jsonSchema) > 0 {
			respFormat.JsonSchema, _ = common.Marshal(jsonSchema)
		}
	}
	return respFormat
}

func convertResponsesContentPartsToChatMedia(parts []any) []dto.MediaContent {
	contents := make([]dto.MediaContent, 0, len(parts))
	for _, itemAny := range parts {
		item, ok := itemAny.(map[string]any)
		if !ok {
			continue
		}

		switch strings.TrimSpace(common.Interface2String(item["type"])) {
		case "input_text", "output_text":
			contents = append(contents, dto.MediaContent{
				Type: dto.ContentTypeText,
				Text: common.Interface2String(item["text"]),
			})
		case "input_image":
			imageURL := item["image_url"]
			image := &dto.MessageImageUrl{Detail: common.Interface2String(item["detail"])}
			switch v := imageURL.(type) {
			case string:
				image.Url = v
			case map[string]any:
				image.Url = common.Interface2String(v["url"])
				if image.Detail == "" {
					image.Detail = common.Interface2String(v["detail"])
				}
			}
			if image.Url != "" {
				contents = append(contents, dto.MediaContent{
					Type:     dto.ContentTypeImageURL,
					ImageUrl: image,
				})
			}
		case "input_audio":
			audioMap, ok := item["input_audio"].(map[string]any)
			if !ok {
				continue
			}
			contents = append(contents, dto.MediaContent{
				Type: dto.ContentTypeInputAudio,
				InputAudio: &dto.MessageInputAudio{
					Data:   common.Interface2String(audioMap["data"]),
					Format: common.Interface2String(audioMap["format"]),
				},
			})
		case "input_file":
			if fileAny, ok := item["file"].(map[string]any); ok {
				file := &dto.MessageFile{
					FileName: common.Interface2String(fileAny["filename"]),
					FileData: common.Interface2String(fileAny["file_data"]),
					FileId:   common.Interface2String(fileAny["file_id"]),
				}
				contents = append(contents, dto.MediaContent{
					Type: dto.ContentTypeFile,
					File: file,
				})
				continue
			}
			fileURL := common.Interface2String(item["file_url"])
			if fileURL != "" {
				contents = append(contents, dto.MediaContent{
					Type: dto.ContentTypeText,
					Text: fileURL,
				})
			}
		case "input_video":
			videoURL := item["video_url"]
			video := &dto.MessageVideoUrl{}
			switch v := videoURL.(type) {
			case string:
				video.Url = v
			case map[string]any:
				video.Url = common.Interface2String(v["url"])
			}
			if video.Url != "" {
				contents = append(contents, dto.MediaContent{
					Type:     dto.ContentTypeVideoUrl,
					VideoUrl: video,
				})
			}
		}
	}
	return contents
}

func convertResponsesRoleItemToChatMessage(role string, content any) dto.Message {
	msg := dto.Message{Role: role}
	switch v := content.(type) {
	case nil:
		msg.Content = ""
	case string:
		msg.Content = v
	case []any:
		contents := convertResponsesContentPartsToChatMedia(v)
		if len(contents) == 0 {
			msg.Content = ""
		} else {
			msg.SetMediaContent(contents)
		}
	default:
		if raw, err := common.Marshal(v); err == nil {
			msg.Content = string(raw)
		} else {
			msg.Content = fmt.Sprintf("%v", v)
		}
	}
	return msg
}

func convertResponsesToolsToChat(raw json.RawMessage) ([]dto.ToolCallRequest, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var toolMaps []map[string]any
	if err := common.Unmarshal(raw, &toolMaps); err != nil {
		return nil, err
	}

	tools := make([]dto.ToolCallRequest, 0, len(toolMaps))
	for _, toolMap := range toolMaps {
		toolType := strings.TrimSpace(common.Interface2String(toolMap["type"]))
		if toolType == "" {
			continue
		}

		tool := dto.ToolCallRequest{Type: toolType}
		if toolType == "function" {
			tool.Function = dto.FunctionRequest{
				Name:        common.Interface2String(toolMap["name"]),
				Description: common.Interface2String(toolMap["description"]),
				Parameters:  toolMap["parameters"],
			}
		} else {
			tool.Custom, _ = common.Marshal(toolMap)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func convertResponsesToolChoiceToChat(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	if common.GetJsonType(raw) == "string" {
		var s string
		if err := common.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return s, nil
	}

	var choiceMap map[string]any
	if err := common.Unmarshal(raw, &choiceMap); err != nil {
		return nil, err
	}

	if strings.TrimSpace(common.Interface2String(choiceMap["type"])) == "function" {
		name := strings.TrimSpace(common.Interface2String(choiceMap["name"]))
		if name != "" {
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": name,
				},
			}, nil
		}
	}
	return choiceMap, nil
}

func ResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Model == "" {
		return nil, errors.New("model is required")
	}
	if req.PreviousResponseID != "" {
		return nil, errors.New("previous_response_id is not supported in chat compatibility mode")
	}
	if len(req.Conversation) > 0 || len(req.ContextManagement) > 0 {
		return nil, errors.New("conversation state is not supported in chat compatibility mode")
	}

	out := &dto.GeneralOpenAIRequest{
		Model:                req.Model,
		Stream:               req.Stream,
		StreamOptions:        req.StreamOptions,
		MaxCompletionTokens:  req.MaxOutputTokens,
		Temperature:          req.Temperature,
		TopP:                 req.TopP,
		TopLogProbs:          req.TopLogProbs,
		User:                 req.User,
		SafetyIdentifier:     req.SafetyIdentifier,
		Store:                req.Store,
		PromptCacheRetention: req.PromptCacheRetention,
		Metadata:             req.Metadata,
		EnableThinking:       req.EnableThinking,
	}

	if req.ServiceTier != "" {
		out.ServiceTier, _ = common.Marshal(req.ServiceTier)
	}
	if promptCacheKey := rawMessageToString(req.PromptCacheKey); promptCacheKey != "" {
		out.PromptCacheKey = promptCacheKey
	}
	if req.Reasoning != nil && strings.TrimSpace(req.Reasoning.Effort) != "" {
		out.ReasoningEffort = req.Reasoning.Effort
	}
	out.ResponseFormat = convertResponsesTextFormatToChatResponseFormat(req.Text)
	if len(req.ParallelToolCalls) > 0 && common.GetJsonType(req.ParallelToolCalls) == "boolean" {
		var parallel bool
		if err := common.Unmarshal(req.ParallelToolCalls, &parallel); err == nil {
			out.ParallelTooCalls = common.GetPointer(parallel)
		}
	}
	if len(req.Tools) > 0 {
		tools, err := convertResponsesToolsToChat(req.Tools)
		if err != nil {
			return nil, err
		}
		out.Tools = tools
	}
	if len(req.ToolChoice) > 0 {
		toolChoice, err := convertResponsesToolChoiceToChat(req.ToolChoice)
		if err != nil {
			return nil, err
		}
		out.ToolChoice = toolChoice
	}

	if instructions := rawMessageToString(req.Instructions); strings.TrimSpace(instructions) != "" {
		out.Messages = append(out.Messages, dto.Message{
			Role:    "system",
			Content: instructions,
		})
	}

	switch common.GetJsonType(req.Input) {
	case "string":
		var s string
		if err := common.Unmarshal(req.Input, &s); err != nil {
			return nil, err
		}
		out.Messages = append(out.Messages, dto.Message{
			Role:    "user",
			Content: s,
		})
	case "array":
		var items []any
		if err := common.Unmarshal(req.Input, &items); err != nil {
			return nil, err
		}
		for _, itemAny := range items {
			item, ok := itemAny.(map[string]any)
			if !ok {
				continue
			}

			role := strings.TrimSpace(common.Interface2String(item["role"]))
			if role != "" {
				out.Messages = append(out.Messages, convertResponsesRoleItemToChatMessage(role, item["content"]))
				continue
			}

			switch strings.TrimSpace(common.Interface2String(item["type"])) {
			case "function_call":
				name := strings.TrimSpace(common.Interface2String(item["name"]))
				if name == "" {
					continue
				}
				callID := strings.TrimSpace(common.Interface2String(item["call_id"]))
				if callID == "" {
					callID = strings.TrimSpace(common.Interface2String(item["id"]))
				}
				msg := dto.Message{
					Role:    "assistant",
					Content: "",
				}
				msg.SetToolCalls([]dto.ToolCallRequest{
					{
						ID:   callID,
						Type: "function",
						Function: dto.FunctionRequest{
							Name:      name,
							Arguments: common.Interface2String(item["arguments"]),
						},
					},
				})
				out.Messages = append(out.Messages, msg)
			case "function_call_output":
				output := item["output"]
				content := ""
				switch v := output.(type) {
				case nil:
				case string:
					content = v
				default:
					if raw, err := common.Marshal(v); err == nil {
						content = string(raw)
					} else {
						content = fmt.Sprintf("%v", v)
					}
				}
				out.Messages = append(out.Messages, dto.Message{
					Role:       "tool",
					ToolCallId: strings.TrimSpace(common.Interface2String(item["call_id"])),
					Content:    content,
				})
			case "input_text", "input_image", "input_audio", "input_file", "input_video":
				msg := convertResponsesRoleItemToChatMessage("user", []any{item})
				out.Messages = append(out.Messages, msg)
			}
		}
	default:
		return nil, errors.New("input is required")
	}

	if len(out.Messages) == 0 {
		return nil, errors.New("input is required")
	}
	return out, nil
}
