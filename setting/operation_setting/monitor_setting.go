package operation_setting

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type MonitorSetting struct {
	AutoTestChannelEnabled bool    `json:"auto_test_channel_enabled"`
	AutoTestChannelMinutes float64 `json:"auto_test_channel_minutes"`
	TestParamOverride      string  `json:"test_param_override"`
	TestHeaderOverride     string  `json:"test_header_override"`
}

// 默认配置
var monitorSetting = MonitorSetting{
	AutoTestChannelEnabled: false,
	AutoTestChannelMinutes: 10,
	TestParamOverride:      "",
	TestHeaderOverride:     "",
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("monitor_setting", &monitorSetting)
}

func GetMonitorSetting() *MonitorSetting {
	if os.Getenv("CHANNEL_TEST_FREQUENCY") != "" {
		frequency, err := strconv.Atoi(os.Getenv("CHANNEL_TEST_FREQUENCY"))
		if err == nil && frequency > 0 {
			monitorSetting.AutoTestChannelEnabled = true
			monitorSetting.AutoTestChannelMinutes = float64(frequency)
		}
	}
	return &monitorSetting
}

func parseMonitorJSONObject(raw string, fieldName string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	var parsed map[string]interface{}
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return nil, fmt.Errorf("%s必须是合法的 JSON 对象: %w", fieldName, err)
	}
	if parsed == nil {
		return nil, fmt.Errorf("%s必须是合法的 JSON 对象", fieldName)
	}
	return parsed, nil
}

func ValidateMonitorJSONObject(raw string, fieldName string) error {
	_, err := parseMonitorJSONObject(raw, fieldName)
	return err
}

func (s *MonitorSetting) GetTestParamOverrideMap() (map[string]interface{}, error) {
	return parseMonitorJSONObject(s.TestParamOverride, "全局测试参数覆盖")
}

func (s *MonitorSetting) GetTestHeaderOverrideMap() (map[string]interface{}, error) {
	return parseMonitorJSONObject(s.TestHeaderOverride, "全局测试请求头覆盖")
}
