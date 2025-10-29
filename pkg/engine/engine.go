package engine

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
	"yamleditor/pkg/path"
)

// Engine 执行 YAML 修改操作
type Engine struct {
	navigator *path.Navigator
}

func NewEngine() *Engine {
	return &Engine{
		navigator: &path.Navigator{},
	}
}

// Apply 应用规则到 YAML 文档
func (e *Engine) Apply(root *yaml.Node, rule *Rule) error {
	switch rule.Action {
	case ActionReplace:
		return e.replace(root, rule)
	case ActionSet:
		return e.set(root, rule)
	case ActionDelete:
		return e.delete(root, rule)
	case ActionRegexReplace:
		return e.regexReplace(root, rule)
	default:
		return fmt.Errorf("unknown action: %s", rule.Action)
	}
}

// replace 替换整个节点
func (e *Engine) replace(root *yaml.Node, rule *Rule) error {
	p, err := path.Parse(rule.Path)
	if err != nil {
		return fmt.Errorf("parse path: %w", err)
	}

	nodes, err := e.navigator.Find(root, p)
	if err != nil {
		return fmt.Errorf("find nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found")
	}

	// 将 Value 转换为 yaml.Node
	newNode := &yaml.Node{}
	if err := newNode.Encode(rule.Value); err != nil {
		return fmt.Errorf("encode value: %w", err)
	}

	// 替换所有匹配的节点
	for _, node := range nodes {
		*node = *newNode
	}

	return nil
}

// set 修改节点的某个字段值
func (e *Engine) set(root *yaml.Node, rule *Rule) error {
	p, err := path.Parse(rule.Path)
	if err != nil {
		return fmt.Errorf("parse path: %w", err)
	}

	nodes, err := e.navigator.Find(root, p)
	if err != nil {
		return fmt.Errorf("find nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found")
	}

	// 修改值
	for _, node := range nodes {
		switch v := rule.Value.(type) {
		case string:
			node.Value = v
			node.Kind = yaml.ScalarNode
			node.Tag = "!!str"
		case int:
			node.Value = fmt.Sprint(v)
			node.Kind = yaml.ScalarNode
			node.Tag = "!!int"
		case bool:
			node.Value = fmt.Sprint(v)
			node.Kind = yaml.ScalarNode
			node.Tag = "!!bool"
		default:
			// 复杂类型，用 Encode
			newNode := &yaml.Node{}
			if err := newNode.Encode(v); err != nil {
				return fmt.Errorf("encode value: %w", err)
			}
			*node = *newNode
		}
	}

	return nil
}

// delete 删除节点
func (e *Engine) delete(root *yaml.Node, rule *Rule) error {
	p, err := path.Parse(rule.Path)
	if err != nil {
		return fmt.Errorf("parse path: %w", err)
	}

	// 使用 where 条件查找
	nodes, err := e.navigator.FindWithWhere(root, p, rule.Where)
	if err != nil {
		return fmt.Errorf("find nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil // 没有要删除的节点，不报错
	}

	// 删除节点需要从父节点操作
	for _, node := range nodes {
		if err := e.deleteNode(root, node); err != nil {
			return err
		}
	}

	return nil
}

// deleteNode 从树中删除节点
func (e *Engine) deleteNode(root, target *yaml.Node) error {
	return e.deleteNodeRecursive(root, target)
}

// deleteNodeRecursive 递归查找并删除节点
func (e *Engine) deleteNodeRecursive(node, target *yaml.Node) error {
	if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			if err := e.deleteNodeRecursive(child, target); err != nil {
				return err
			}
		}
		return nil
	}

	if node.Kind == yaml.MappingNode {
		// 检查值是否是目标
		newContent := []*yaml.Node{}
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if valueNode == target {
				// 跳过这个键值对（删除）
				continue
			}

			newContent = append(newContent, keyNode, valueNode)

			// 递归查找子节点
			if err := e.deleteNodeRecursive(valueNode, target); err != nil {
				return err
			}
		}
		node.Content = newContent
		return nil
	}

	if node.Kind == yaml.SequenceNode {
		// 检查元素是否是目标
		newContent := []*yaml.Node{}
		for _, elem := range node.Content {
			if elem == target {
				// 跳过这个元素（删除）
				continue
			}

			newContent = append(newContent, elem)

			// 递归查找子节点
			if err := e.deleteNodeRecursive(elem, target); err != nil {
				return err
			}
		}
		node.Content = newContent
		return nil
	}

	return nil
}

// regexReplace 正则替换字符串值
func (e *Engine) regexReplace(root *yaml.Node, rule *Rule) error {
	p, err := path.Parse(rule.Path)
	if err != nil {
		return fmt.Errorf("parse path: %w", err)
	}

	nodes, err := e.navigator.Find(root, p)
	if err != nil {
		return fmt.Errorf("find nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found")
	}

	if rule.Pattern == "" {
		return fmt.Errorf("pattern is required for regex_replace")
	}

	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("compile regex: %w", err)
	}

	replacement, ok := rule.Value.(string)
	if !ok {
		return fmt.Errorf("replacement must be string")
	}

	// 替换所有匹配节点的值
	for _, node := range nodes {
		if node.Kind != yaml.ScalarNode {
			continue
		}
		node.Value = re.ReplaceAllString(node.Value, replacement)
	}

	return nil
}
