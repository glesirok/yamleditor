package path

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Navigator 负责在 YAML 树中导航和查找节点
type Navigator struct{}

// Find 根据路径查找所有匹配的节点
// 返回匹配的节点列表（因为可能有通配符）
func (n *Navigator) Find(root *yaml.Node, path *Path) ([]*yaml.Node, error) {
	return n.findRecursive(root, path.Segments, 0)
}
func (n *Navigator) findRecursive(node *yaml.Node, segments []*Segment, segmentIdx int) ([]*yaml.Node, error) {
	// 到达路径末尾
	if segmentIdx >= len(segments) {
		return []*yaml.Node{node}, nil
	}

	segment := segments[segmentIdx]

	// 处理文档节点和别名节点
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil, fmt.Errorf("empty document")
		}
		return n.findRecursive(node.Content[0], segments, segmentIdx)
	}

	if node.Kind == yaml.AliasNode {
		return n.findRecursive(node.Alias, segments, segmentIdx)
	}

	switch segment.Type {
	case SegmentTypeField:
		return n.findField(node, segment, segments, segmentIdx)
	case SegmentTypeArray:
		return n.findArray(node, segment, segments, segmentIdx)
	default:
		return nil, fmt.Errorf("unknown segment type")
	}
}

// findField 查找字段
func (n *Navigator) findField(node *yaml.Node, segment *Segment, segments []*Segment, segmentIdx int) ([]*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node, got %v", node.Kind)
	}

	// YAML MappingNode 的 Content 是 [key1, value1, key2, value2, ...]
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == segment.Field {
			return n.findRecursive(valueNode, segments, segmentIdx+1)
		}
	}

	return nil, fmt.Errorf("field '%s' not found", segment.Field)
}

// findArray 查找数组元素
func (n *Navigator) findArray(node *yaml.Node, segment *Segment, segments []*Segment, segmentIdx int) ([]*yaml.Node, error) {
	// 先找到数组字段
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node for array field")
	}

	var arrayNode *yaml.Node
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == segment.Field {
			arrayNode = valueNode
			break
		}
	}

	if arrayNode == nil {
		return nil, fmt.Errorf("array field '%s' not found", segment.Field)
	}

	if arrayNode.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("field '%s' is not an array", segment.Field)
	}

	// 根据选择器类型匹配元素
	switch segment.Selector.Type {
	case SelectorTypeWildcard:
		// 通配符：匹配所有元素
		var results []*yaml.Node
		for _, elem := range arrayNode.Content {
			matched, err := n.findRecursive(elem, segments, segmentIdx+1)
			if err != nil {
				continue // 某个元素不匹配，继续下一个
			}
			results = append(results, matched...)
		}
		return results, nil

	case SelectorTypeIndex:
		// 索引：匹配指定位置
		idx := segment.Selector.Condition.Value.(int)
		if idx < 0 || idx >= len(arrayNode.Content) {
			return nil, fmt.Errorf("index %d out of range", idx)
		}
		return n.findRecursive(arrayNode.Content[idx], segments, segmentIdx+1)

	case SelectorTypeCondition:
		// 条件：匹配字段值
		var results []*yaml.Node
		for _, elem := range arrayNode.Content {
			if n.matchCondition(elem, segment.Selector.Condition) {
				matched, err := n.findRecursive(elem, segments, segmentIdx+1)
				if err != nil {
					continue
				}
				results = append(results, matched...)
			}
		}
		if len(results) == 0 {
			return nil, fmt.Errorf("no elements match condition")
		}
		return results, nil

	default:
		return nil, fmt.Errorf("unknown selector type")
	}
}

// matchCondition 检查节点是否匹配条件
func (n *Navigator) matchCondition(node *yaml.Node, cond *Condition) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}

	// 查找字段
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == cond.Field {
			switch cond.Op {
			case OpEqual:
				return valueNode.Value == fmt.Sprint(cond.Value)
			case OpRegex:
				pattern := cond.Value.(string)
				matched, err := regexp.MatchString(pattern, valueNode.Value)
				return err == nil && matched
			}
		}
	}

	return false
}
