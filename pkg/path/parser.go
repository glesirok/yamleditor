package path

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

// Parse 解析路径字符串
// 支持语法：
//   - spec.template.spec
//   - containers[*]
//   - containers[0]
//   - containers[name=foo]
//   - env[?] (占位符，实际匹配由 where 条件决定)
func Parse(pathStr string) (*Path, error) {
	if pathStr == "" {
		return nil, fmt.Errorf("empty path")
	}

	segments := []*Segment{}
	parts := splitPath(pathStr)

	for _, part := range parts {
		seg, err := parseSegment(part)
		if err != nil {
			return nil, fmt.Errorf("invalid segment '%s': %w", part, err)
		}
		segments = append(segments, seg)
	}

	return &Path{Segments: segments}, nil
}

// splitPath 分割路径，处理 . 和 []
// 例如: "spec.containers[name=foo].env" -> ["spec", "containers[name=foo]", "env"]
func splitPath(pathStr string) []string {
	var parts []string
	var current strings.Builder
	inBracket := false

	for _, ch := range pathStr {
		switch ch {
		case '[':
			inBracket = true
			current.WriteRune(ch)
		case ']':
			inBracket = false
			current.WriteRune(ch)
		case '.':
			if inBracket {
				current.WriteRune(ch)
			} else {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseSegment 解析单个路径片段
func parseSegment(part string) (*Segment, error) {
	// 检查是否有选择器
	if strings.Contains(part, "[") {
		return parseArraySegment(part)
	}

	// 普通字段
	return &Segment{
		Type:  SegmentTypeField,
		Field: part,
	}, nil
}

// parseArraySegment 解析数组片段，如 "containers[name=foo]" 或 "env[name=@^SW_.*$@]"
func parseArraySegment(part string) (*Segment, error) {
	bracketStart := strings.Index(part, "[")
	if bracketStart == -1 {
		return nil, fmt.Errorf("no opening bracket")
	}

	field := part[:bracketStart]

	// 查找配对的 ]，跳过 @...@ 内部的 ]
	bracketEnd := findClosingBracket(part, bracketStart+1)
	if bracketEnd == -1 {
		return nil, fmt.Errorf("no closing bracket")
	}

	selectorStr := part[bracketStart+1 : bracketEnd]

	selector, err := parseSelector(selectorStr)
	if err != nil {
		return nil, err
	}

	return &Segment{
		Type:     SegmentTypeArray,
		Field:    field,
		Selector: selector,
	}, nil
}

// findClosingBracket 查找配对的 ]，忽略 @...@ 内部的 ]
func findClosingBracket(s string, start int) int {
	inRegex := false

	for i := start; i < len(s); i++ {
		ch := s[i]

		if ch == '@' {
			inRegex = !inRegex
		}

		if ch == ']' && !inRegex {
			return i
		}
	}

	return -1 // 未找到
}

// parseSelector 解析选择器
// 支持语法：
//   - * 或 ? : 通配符
//   - 数字 : 索引
//   - field=value : 精确匹配
//   - field=@pattern@ : 正则匹配
func parseSelector(selectorStr string) (*Selector, error) {
	// 通配符
	if selectorStr == "*" {
		return &Selector{Type: SelectorTypeWildcard}, nil
	}

	// 占位符（where 条件）
	if selectorStr == "?" {
		return &Selector{Type: SelectorTypeWildcard}, nil
	}

	// 索引
	if idx, err := strconv.Atoi(selectorStr); err == nil {
		return &Selector{
			Type: SelectorTypeIndex,
			Condition: &Condition{
				Field: "_index",
				Op:    OpEqual,
				Value: idx,
			},
		}, nil
	}

	// 条件：field=value 或 field=@pattern@
	if strings.Contains(selectorStr, "=") {
		parts := strings.SplitN(selectorStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid condition syntax")
		}

		field := parts[0]
		value := parts[1]

		if field == "" {
			return nil, fmt.Errorf("field name cannot be empty")
		}

		// 检测是否是正则（以 @ 包裹）
		if strings.HasPrefix(value, "@") && strings.HasSuffix(value, "@") {
			pattern := strings.Trim(value, "@")

			if pattern == "" {
				return nil, fmt.Errorf("regex pattern cannot be empty")
			}

			// 校验正则合法性
			if _, err := regexp2.Compile(pattern, 0); err != nil {
				return nil, fmt.Errorf("invalid regex pattern: %w", err)
			}

			return &Selector{
				Type: SelectorTypeCondition,
				Condition: &Condition{
					Field: field,
					Op:    OpRegex,
					Value: pattern,
				},
			}, nil
		}

		// 精确匹配
		return &Selector{
			Type: SelectorTypeCondition,
			Condition: &Condition{
				Field: field,
				Op:    OpEqual,
				Value: value,
			},
		}, nil
	}

	return nil, fmt.Errorf("unknown selector syntax: %s", selectorStr)
}
