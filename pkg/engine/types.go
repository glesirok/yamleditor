package engine

import "yamleditor/pkg/path"

// ActionType 定义操作类型
type ActionType string

const (
	ActionReplace      ActionType = "replace"
	ActionSet          ActionType = "set"
	ActionDelete       ActionType = "delete"
	ActionRegexReplace ActionType = "regex_replace"
)

// Rule 表示一条修改规则
type Rule struct {
	Action  ActionType             `yaml:"action"`
	Path    string                 `yaml:"path"`
	Value   interface{}            `yaml:"value,omitempty"`
	Pattern string                 `yaml:"pattern,omitempty"` // 用于 regex_replace
	Where   *path.WhereCondition   `yaml:"where,omitempty"`   // 用于 delete 的条件过滤
}
