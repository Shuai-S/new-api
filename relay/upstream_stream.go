package relay

import (
	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func shouldBufferUpstreamStream(info *relaycommon.RelayInfo) bool {
	if info == nil || info.ClientStream {
		return false
	}
	if !info.ChannelSetting.UpstreamStreamForNonStream {
		return false
	}
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		return false
	}
	return info.RelayMode == relayconstant.RelayModeChatCompletions || info.RelayFormat == types.RelayFormatClaude
}

func forceOpenAIUpstreamStream(info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) {
	if info == nil || request == nil {
		return
	}
	request.Stream = common.GetPointer(true)
	if info.SupportStreamOptions {
		request.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}
	info.IsStream = true
	info.UpstreamStream = true
	info.BufferNonStreamStream = true
}

func forceClaudeUpstreamStream(info *relaycommon.RelayInfo, request *dto.ClaudeRequest) {
	if info == nil || request == nil {
		return
	}
	request.Stream = common.GetPointer(true)
	info.IsStream = true
	info.UpstreamStream = true
	info.BufferNonStreamStream = true
}
