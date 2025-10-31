package engine

// ActionType 定义操作类型
type ActionType string

const (
	ActionReplace      ActionType = "replace"
	ActionDelete       ActionType = "delete"
	ActionRegexReplace ActionType = "regex_replace"
)

// Rule 表示一条修改规则
type Rule struct {
	Action             ActionType  `yaml:"action"`
	Path               string      `yaml:"path"`
	Value              interface{} `yaml:"value,omitempty"`
	Pattern            string      `yaml:"pattern,omitempty"`            // 用于 regex_replace
	ContinueOnNotFound bool        `yaml:"continue_on_not_found,omitempty"` // 找不到节点时是否继续
}
