package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestPath2EndpointType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want constant.EndpointType
	}{
		{name: "playground chat", path: "/pg/chat/completions", want: constant.EndpointTypeOpenAI},
		{name: "responses compact", path: "/v1/responses/compact", want: constant.EndpointTypeOpenAIResponseCompact},
		{name: "responses", path: "/v1/responses", want: constant.EndpointTypeOpenAIResponse},
		{name: "anthropic", path: "/v1/messages", want: constant.EndpointTypeAnthropic},
		{name: "gemini", path: "/v1beta/models/gemini-2.5-pro:generateContent", want: constant.EndpointTypeGemini},
		{name: "rerank", path: "/v1/rerank", want: constant.EndpointTypeJinaRerank},
		{name: "image edits", path: "/v1/images/edits", want: constant.EndpointTypeImageGeneration},
		{name: "embeddings", path: "/v1/embeddings", want: constant.EndpointTypeEmbeddings},
		{name: "video", path: "/v1/videos", want: constant.EndpointTypeOpenAIVideo},
		{name: "unknown", path: "/v1/audio/speech", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Path2EndpointType(tt.path); got != tt.want {
				t.Fatalf("Path2EndpointType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
