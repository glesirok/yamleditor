package path

// Segment 表示路径的一个片段
type Segment struct {
	Type      SegmentType
	Field     string            // 字段名，如 "spec"
	Selector  *Selector         // 选择器，如 [name=foo] 或 [*]
}

type SegmentType int

const (
	SegmentTypeField SegmentType = iota // 普通字段访问
	SegmentTypeArray                    // 数组访问
)

// Selector 表示数组选择器
type Selector struct {
	Type      SelectorType
	Condition *Condition // 条件匹配
}

type SelectorType int

const (
	SelectorTypeWildcard SelectorType = iota // [*] 通配符
	SelectorTypeIndex                        // [0] 索引
	SelectorTypeCondition                    // [name=foo] 条件
)

// Condition 表示匹配条件
type Condition struct {
	Field string      // 字段名
	Op    Operator    // 操作符
	Value interface{} // 值
}

type Operator int

const (
	OpEqual    Operator = iota // =
	OpNotEqual                 // !=
)

// Path 表示解析后的完整路径
type Path struct {
	Segments []*Segment
}
