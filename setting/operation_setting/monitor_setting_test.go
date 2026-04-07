package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateMonitorJSONObject_AllowsEmpty(t *testing.T) {
	require.NoError(t, ValidateMonitorJSONObject("", "全局测试参数覆盖"))
}

func TestValidateMonitorJSONObject_AcceptsObject(t *testing.T) {
	require.NoError(t, ValidateMonitorJSONObject(`{"User-Agent":"new-api-test/1.0"}`, "全局测试请求头覆盖"))
}

func TestValidateMonitorJSONObject_RejectsArray(t *testing.T) {
	err := ValidateMonitorJSONObject(`[]`, "全局测试参数覆盖")
	require.Error(t, err)
	require.Contains(t, err.Error(), "JSON 对象")
}

func TestValidateMonitorJSONObject_RejectsMalformedJSON(t *testing.T) {
	err := ValidateMonitorJSONObject(`{"operations":[}`, "全局测试参数覆盖")
	require.Error(t, err)
	require.Contains(t, err.Error(), "JSON 对象")
}

func TestMonitorSettingGetTestParamOverrideMap(t *testing.T) {
	setting := &MonitorSetting{
		TestParamOverride: `{"operations":[{"mode":"set","path":"messages.0.content","value":"ok"}]}`,
	}

	override, err := setting.GetTestParamOverrideMap()
	require.NoError(t, err)
	require.NotNil(t, override)
	require.Contains(t, override, "operations")
}
