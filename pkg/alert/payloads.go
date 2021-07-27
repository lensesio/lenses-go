package alert

//SettingConditionPayloads is the payload for creating alert setttings
type SettingConditionPayloads struct {
	AlertID    int      `json:"alert" yaml:"alert"`
	Conditions []string `json:"conditions" yaml:"conditions"`
}

//SettingConditionPayload is the payload for creating alert setttings
type SettingConditionPayload struct {
	AlertID     int      `json:"alert" yaml:"alert"`
	ConditionID string   `json:"conditionID,omitempty" yaml:"conditionID"`
	Channels    []string `json:"channels" yaml:"channels"`
	Topic       string   `yaml:"topic,omitempty"`
	Group       string   `yaml:"group,omitempty"`
	Threshold   int      `yaml:"threshold,omitempty"`
	Mode        string   `yaml:"mode,omitempty"`
	MoreThan    int      `yaml:"more-than,omitempty"`
	LessThan    int      `yaml:"less-than,omitempty"`
	Duration    string   `yaml:"duration,omitempty"`
}
