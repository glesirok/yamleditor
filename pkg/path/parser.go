package path

import (
	"fmt"
	"strconv"
	"strings"
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

// parseArraySegment 解析数组片段，如 "containers[name=foo]"
func parseArraySegment(part string) (*Segment, error) {
	bracketStart := strings.Index(part, "[")
	bracketEnd := strings.LastIndex(part, "]")

	if bracketStart == -1 || bracketEnd == -1 || bracketEnd < bracketStart {
		return nil, fmt.Errorf("invalid bracket syntax")
	}

	field := part[:bracketStart]
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

// parseSelector 解析选择器
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

	// 条件，如 "name=foo"
	if strings.Contains(selectorStr, "=") {
		parts := strings.SplitN(selectorStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid condition syntax")
		}

		return &Selector{
			Type: SelectorTypeCondition,
			Condition: &Condition{
				Field: parts[0],
				Op:    OpEqual,
				Value: parts[1],
			},
		}, nil
	}

	return nil, fmt.Errorf("unknown selector syntax: %s", selectorStr)
}
