package model

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func IsChannelAllowedForEndpoint(channel *Channel, endpointType constant.EndpointType) bool {
	if channel == nil {
		return false
	}
	if !model_setting.GetGlobalSettings().ChannelEndpointFilterEnabled {
		return true
	}
	settings := channel.GetOtherSettings()
	return settings.AllowsEndpointType(endpointType)
}
