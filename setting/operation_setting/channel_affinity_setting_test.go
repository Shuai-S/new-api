package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaudeCLITracePassesRequestIdentityHeaders(t *testing.T) {
	var claudeRule *ChannelAffinityRule
	for i := range channelAffinitySetting.Rules {
		if channelAffinitySetting.Rules[i].Name == "claude cli trace" {
			claudeRule = &channelAffinitySetting.Rules[i]
			break
		}
	}
	require.NotNil(t, claudeRule)

	operations, ok := claudeRule.ParamOverrideTemplate["operations"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, operations, 1)
	headers, ok := operations[0]["value"].([]string)
	require.True(t, ok)
	require.Contains(t, headers, "X-Stainless-Helper-Method")
	require.Contains(t, headers, "X-Claude-Code-Session-Id")
	require.Contains(t, headers, "X-Claude-Code-Agent-Id")
	require.Contains(t, headers, "X-Client-Request-Id")
}
