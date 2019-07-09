package alert

//SettingConditionPayloads is the payload for creating alert setttings
type SettingConditionPayloads struct {
	AlertID    int      `json:"alert" yaml:"alert"`
	Conditions []string `json:"conditions" yaml:"conditions"`
}

//SettingConditionPayload is the payload for creating alert setttings
type SettingConditionPayload struct {
	AlertID   int    `json:"alert" yaml:"alert"`
	Condition string `json:"condition" yaml:"condition"`
}
