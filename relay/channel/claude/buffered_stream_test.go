package claude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newClaudeBufferedStreamTestContext(t *testing.T, body string, relayFormat types.RelayFormat) (*gin.Context, *httptest.ResponseRecorder, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Set(common.RequestIdKey, "claude-buffered-test")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta:           &relaycommon.ChannelMeta{UpstreamModelName: "claude-test"},
		IsStream:              true,
		ClientStream:          false,
		BufferNonStreamStream: true,
		RelayFormat:           relayFormat,
	}
	return c, recorder, resp, info
}

func TestClaudeBufferedStreamHandlerReturnsClaudeMessageJSONFromSSE(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	body := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-test","usage":{"input_tokens":2}}}`,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup","input":{}}}`,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"q\":\"x\"}"}}`,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	c, recorder, resp, info := newClaudeBufferedStreamTestContext(t, body, types.RelayFormatClaude)

	usage, err := ClaudeBufferedStreamHandler(c, resp, info)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 2, usage.PromptTokens)
	require.Equal(t, 3, usage.CompletionTokens)
	require.Equal(t, 5, usage.TotalTokens)

	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	got := recorder.Body.String()
	require.NotContains(t, got, "data:")

	var parsed dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(common.StringToByteSlice(got), &parsed))
	require.Equal(t, "msg_1", parsed.Id)
	require.Equal(t, "message", parsed.Type)
	require.Equal(t, "assistant", parsed.Role)
	require.Equal(t, "claude-test", parsed.Model)
	require.Equal(t, "end_turn", parsed.StopReason)
	require.NotNil(t, parsed.Usage)
	require.Equal(t, 2, parsed.Usage.InputTokens)
	require.Equal(t, 3, parsed.Usage.OutputTokens)
	require.Len(t, parsed.Content, 2)
	require.Equal(t, "text", parsed.Content[0].Type)
	require.Equal(t, "hello", parsed.Content[0].GetText())
	require.Equal(t, "tool_use", parsed.Content[1].Type)
	require.Equal(t, "toolu_1", parsed.Content[1].Id)
	require.Equal(t, "lookup", parsed.Content[1].Name)
	input, ok := parsed.Content[1].Input.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "x", input["q"])
}

func TestClaudeBufferedStreamHandlerCanReturnOpenAIJSONFromSSE(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	body := strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-test","usage":{"input_tokens":2}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	c, recorder, resp, info := newClaudeBufferedStreamTestContext(t, body, types.RelayFormatOpenAI)

	usage, err := ClaudeBufferedStreamHandler(c, resp, info)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 5, usage.TotalTokens)

	var parsed dto.OpenAITextResponse
	require.NoError(t, common.Unmarshal(common.StringToByteSlice(recorder.Body.String()), &parsed))
	require.Equal(t, "msg_1", parsed.Id)
	require.Equal(t, "chat.completion", parsed.Object)
	require.Equal(t, "claude-test", parsed.Model)
	require.Equal(t, 5, parsed.Usage.TotalTokens)
	require.Len(t, parsed.Choices, 1)
	require.Equal(t, "hello", parsed.Choices[0].Message.StringContent())
	require.Equal(t, "stop", parsed.Choices[0].FinishReason)
}
