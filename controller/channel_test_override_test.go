package controller

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMergeChannelTestOverrideMaps_ChannelSpecificWins(t *testing.T) {
	merged := mergeChannelTestOverrideMaps(
		map[string]interface{}{
			"User-Agent": "global-agent",
			"X-Global":   "global",
		},
		map[string]interface{}{
			"User-Agent": "channel-agent",
			"X-Channel":  "channel",
		},
	)

	require.Equal(t, "channel-agent", merged["User-Agent"])
	require.Equal(t, "global", merged["X-Global"])
	require.Equal(t, "channel", merged["X-Channel"])
}

func TestApplyGlobalAndChannelTestParamOverrides_ChannelSpecificWins(t *testing.T) {
	info := &relaycommon.RelayInfo{
		IsChannelTest:  true,
		RequestURLPath: "/v1/chat/completions",
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: mergeChannelTestOverrideMaps(
				map[string]interface{}{
					"User-Agent": "global-static-agent",
					"X-Global":   "global",
				},
				map[string]interface{}{
					"User-Agent": "channel-static-agent",
					"X-Channel":  "channel",
				},
			),
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"mode":  "set",
						"path":  "messages.0.content",
						"value": "channel prompt",
					},
					map[string]interface{}{
						"mode":  "set_header",
						"path":  "User-Agent",
						"value": "channel-runtime-agent",
					},
				},
			},
		},
	}
	globalParamOverride := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"mode":  "set",
				"path":  "messages.0.content",
				"value": "global prompt",
				"conditions": []interface{}{
					map[string]interface{}{
						"path":  "is_channel_test",
						"mode":  "full",
						"value": true,
					},
				},
			},
			map[string]interface{}{
				"mode":  "set_header",
				"path":  "User-Agent",
				"value": "global-runtime-agent",
			},
		},
	}

	out, err := applyGlobalAndChannelTestParamOverrides(info, []byte(`{"messages":[{"role":"user","content":"hi"}]}`), globalParamOverride)
	require.NoError(t, err)
	require.Equal(t, "channel prompt", gjson.GetBytes(out, "messages.0.content").String())
	require.True(t, info.UseRuntimeHeadersOverride)
	require.Equal(t, "channel-runtime-agent", info.RuntimeHeadersOverride["user-agent"])
	require.Equal(t, "global", info.RuntimeHeadersOverride["x-global"])
	require.Equal(t, "channel", info.RuntimeHeadersOverride["x-channel"])
}
