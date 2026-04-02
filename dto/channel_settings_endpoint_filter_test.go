package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestChannelOtherSettingsAllowsEndpointType(t *testing.T) {
	t.Parallel()

	settings := &ChannelOtherSettings{
		EndpointFilterEnabled: true,
		AllowedEndpointTypes: []constant.EndpointType{
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeOpenAI,
			" openai-response ",
		},
	}

	if !settings.IsEndpointFilterEnabled() {
		t.Fatal("expected endpoint filter to be enabled")
	}
	if !settings.AllowsEndpointType(constant.EndpointTypeOpenAI) {
		t.Fatal("expected openai endpoint to be allowed")
	}
	if !settings.AllowsEndpointType(constant.EndpointTypeOpenAIResponse) {
		t.Fatal("expected responses endpoint to be allowed after normalization")
	}
	if settings.AllowsEndpointType(constant.EndpointTypeAnthropic) {
		t.Fatal("did not expect anthropic endpoint to be allowed")
	}
	if settings.AllowsEndpointType("") {
		t.Fatal("did not expect empty endpoint type to be allowed")
	}
}

func TestChannelOtherSettingsValidateEndpointTypes(t *testing.T) {
	t.Parallel()

	valid := &ChannelOtherSettings{
		EndpointFilterEnabled: true,
		AllowedEndpointTypes:  []constant.EndpointType{constant.EndpointTypeOpenAI},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid settings, got %v", err)
	}

	invalid := &ChannelOtherSettings{
		EndpointFilterEnabled: true,
		AllowedEndpointTypes:  []constant.EndpointType{"not-a-real-endpoint"},
	}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected invalid endpoint type to fail validation")
	}
}
