package rule

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"yamleditor/pkg/engine"
)

// Config 表示规则配置文件
type Config struct {
	Rules []*engine.Rule `yaml:"rules"`
}

// LoadFromFile 从文件加载规则
func LoadFromFile(filePath string) ([]*engine.Rule, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	// 校验规则
	for i, rule := range config.Rules {
		if err := Validate(rule); err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}
	}

	return config.Rules, nil
}

// Validate 校验规则的合法性
func Validate(rule *engine.Rule) error {
	if rule.Path == "" {
		return fmt.Errorf("path is required")
	}

	switch rule.Action {
	case engine.ActionReplace:
		if rule.Value == nil {
			return fmt.Errorf("value is required for action %s", rule.Action)
		}

	case engine.ActionRegexReplace:
		if rule.Pattern == "" {
			return fmt.Errorf("pattern is required for regex_replace")
		}
		if rule.Value == nil {
			return fmt.Errorf("value (replacement) is required for regex_replace")
		}
		if _, ok := rule.Value.(string); !ok {
			return fmt.Errorf("value must be string for regex_replace")
		}

	case engine.ActionDelete:
		// delete 不需要 value

	default:
		return fmt.Errorf("unknown action: %s", rule.Action)
	}

	return nil
}
