package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

var configurableEndpointTypes = []constant.EndpointType{
	constant.EndpointTypeOpenAI,
	constant.EndpointTypeOpenAIResponse,
	constant.EndpointTypeOpenAIResponseCompact,
	constant.EndpointTypeAnthropic,
	constant.EndpointTypeGemini,
	constant.EndpointTypeJinaRerank,
	constant.EndpointTypeImageGeneration,
	constant.EndpointTypeEmbeddings,
	constant.EndpointTypeOpenAIVideo,
}

func GetConfigurableEndpointTypes() []constant.EndpointType {
	endpointTypes := make([]constant.EndpointType, len(configurableEndpointTypes))
	copy(endpointTypes, configurableEndpointTypes)
	return endpointTypes
}

func IsKnownEndpointType(endpointType constant.EndpointType) bool {
	for _, candidate := range configurableEndpointTypes {
		if candidate == endpointType {
			return true
		}
	}
	return false
}

func NormalizeEndpointTypes(endpointTypes []constant.EndpointType) []constant.EndpointType {
	if len(endpointTypes) == 0 {
		return nil
	}
	normalized := make([]constant.EndpointType, 0, len(endpointTypes))
	seen := make(map[constant.EndpointType]struct{}, len(endpointTypes))
	for _, endpointType := range endpointTypes {
		trimmed := constant.EndpointType(strings.TrimSpace(string(endpointType)))
		if trimmed == "" || !IsKnownEndpointType(trimmed) {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func Path2EndpointType(path string) constant.EndpointType {
	switch {
	case strings.HasPrefix(path, "/pg/chat/completions"):
		fallthrough
	case strings.HasPrefix(path, "/v1/chat/completions"):
		fallthrough
	case strings.HasPrefix(path, "/v1/completions"):
		return constant.EndpointTypeOpenAI
	case strings.HasPrefix(path, "/v1/responses/compact"):
		return constant.EndpointTypeOpenAIResponseCompact
	case strings.HasPrefix(path, "/v1/responses"):
		return constant.EndpointTypeOpenAIResponse
	case strings.HasPrefix(path, "/v1/messages"):
		return constant.EndpointTypeAnthropic
	case strings.HasPrefix(path, "/v1beta/models/"):
		fallthrough
	case strings.HasPrefix(path, "/v1/models/"):
		return constant.EndpointTypeGemini
	case strings.HasPrefix(path, "/v1/rerank"):
		fallthrough
	case strings.HasPrefix(path, "/rerank"):
		return constant.EndpointTypeJinaRerank
	case strings.HasPrefix(path, "/v1/images/generations"):
		fallthrough
	case strings.HasPrefix(path, "/v1/images/edits"):
		return constant.EndpointTypeImageGeneration
	case strings.HasPrefix(path, "/v1/embeddings"):
		fallthrough
	case strings.HasSuffix(path, "embeddings"):
		return constant.EndpointTypeEmbeddings
	case strings.HasPrefix(path, "/v1/videos"):
		fallthrough
	case strings.HasPrefix(path, "/v1/video/generations"):
		return constant.EndpointTypeOpenAIVideo
	default:
		return ""
	}
}

// GetEndpointTypesByChannelType 获取渠道最优先端点类型（所有的渠道都支持 OpenAI 端点）
func GetEndpointTypesByChannelType(channelType int, modelName string) []constant.EndpointType {
	var endpointTypes []constant.EndpointType
	switch channelType {
	case constant.ChannelTypeJina:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeJinaRerank}
	//case constant.ChannelTypeMidjourney, constant.ChannelTypeMidjourneyPlus:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeMidjourney}
	//case constant.ChannelTypeSunoAPI:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeSuno}
	//case constant.ChannelTypeKling:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeKling}
	//case constant.ChannelTypeJimeng:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeJimeng}
	case constant.ChannelTypeAws:
		fallthrough
	case constant.ChannelTypeAnthropic:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAnthropic, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeVertexAi:
		fallthrough
	case constant.ChannelTypeGemini:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeGemini, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeOpenRouter: // OpenRouter 只支持 OpenAI 端点
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
	case constant.ChannelTypeXai:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI, constant.EndpointTypeOpenAIResponse}
	case constant.ChannelTypeSora:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIVideo}
	default:
		if IsOpenAIResponseOnlyModel(modelName) {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIResponse}
		} else {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
		}
	}
	if IsImageGenerationModel(modelName) {
		// add to first
		endpointTypes = append([]constant.EndpointType{constant.EndpointTypeImageGeneration}, endpointTypes...)
	}
	return endpointTypes
}
