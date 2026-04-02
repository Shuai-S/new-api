package model

import (
	"strings"
	"testing"

	common2 "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func withEndpointFilterTestState(t *testing.T, enabled bool, channels map[int]*Channel, mapping map[string]map[string][]int) {
	t.Helper()

	previousMemoryCacheEnabled := common2.MemoryCacheEnabled
	previousGroupMap := group2model2channels
	previousChannels := channelsIDM
	previousGlobalEnabled := model_setting.GetGlobalSettings().ChannelEndpointFilterEnabled

	common2.MemoryCacheEnabled = true
	group2model2channels = mapping
	channelsIDM = channels
	model_setting.GetGlobalSettings().ChannelEndpointFilterEnabled = enabled

	t.Cleanup(func() {
		common2.MemoryCacheEnabled = previousMemoryCacheEnabled
		group2model2channels = previousGroupMap
		channelsIDM = previousChannels
		model_setting.GetGlobalSettings().ChannelEndpointFilterEnabled = previousGlobalEnabled
	})
}

func buildTestChannel(id int, priority int64, allowed []constant.EndpointType) *Channel {
	channel := &Channel{
		Id:       id,
		Priority: common2.GetPointer(priority),
		Weight:   common2.GetPointer(uint(100)),
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		EndpointFilterEnabled: true,
		AllowedEndpointTypes:  allowed,
	})
	return channel
}

func TestGetRandomSatisfiedChannelEndpointFilterSkipsHigherPriorityMismatch(t *testing.T) {
	withEndpointFilterTestState(t, true, map[int]*Channel{
		1: buildTestChannel(1, 200, []constant.EndpointType{constant.EndpointTypeOpenAIResponse}),
		2: buildTestChannel(2, 100, []constant.EndpointType{constant.EndpointTypeOpenAI}),
	}, map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1, 2},
		},
	})

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5.4", 0, constant.EndpointTypeOpenAI)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if channel == nil || channel.Id != 2 {
		t.Fatalf("expected channel 2, got %#v", channel)
	}
}

func TestGetRandomSatisfiedChannelEndpointFilterDisabledByGlobalSetting(t *testing.T) {
	withEndpointFilterTestState(t, false, map[int]*Channel{
		1: buildTestChannel(1, 200, []constant.EndpointType{constant.EndpointTypeOpenAIResponse}),
		2: buildTestChannel(2, 100, []constant.EndpointType{constant.EndpointTypeOpenAI}),
	}, map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1, 2},
		},
	})

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5.4", 0, constant.EndpointTypeOpenAI)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if channel == nil || channel.Id != 1 {
		t.Fatalf("expected channel 1 when global filter is disabled, got %#v", channel)
	}
}

func TestGetRandomSatisfiedChannelEndpointFilterReturnsErrorWhenNoChannelMatches(t *testing.T) {
	withEndpointFilterTestState(t, true, map[int]*Channel{
		1: buildTestChannel(1, 200, []constant.EndpointType{constant.EndpointTypeOpenAIResponse}),
	}, map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1},
		},
	})

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5.4", 0, constant.EndpointTypeOpenAI)
	if err == nil {
		t.Fatal("expected endpoint filter to reject all channels")
	}
	if channel != nil {
		t.Fatalf("expected no channel, got %#v", channel)
	}
	if !strings.Contains(err.Error(), "openai") {
		t.Fatalf("expected error to mention endpoint type, got %v", err)
	}
}
