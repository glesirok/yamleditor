# yamleditor

Kubernetes YAML 批量编辑工具,使用配置化规则进行修改。

## 特性

- **通用路径语法**：支持 `spec.containers[name=foo].env[*]` 等灵活路径表达式
- **3种操作**：replace、delete、regex_replace
- **批量处理**：递归处理目录下所有 YAML 文件
- **安全模式**：dry-run 预览变更,backup 自动备份
- **零特殊情况**：通过配置扩展,无需修改代码

## 目录结构
  yamleditor/
  ├── cmd/
  │   └── yamleditor/
  │       └── main.go              # CLI 入口
  ├── pkg/
  │   ├── path/
  │   │   ├── parser.go            # 路径解析
  │   │   ├── matcher.go           # 条件匹配
  │   │   └── navigator.go         # YAML 树遍历
  │   ├── engine/
  │   │   ├── engine.go            # 操作引擎
  │   │   ├── replace.go           # 替换操作
  │   │   ├── delete.go            # 删除操作
  │   │   └── regex.go             # 正则操作
  │   ├── rule/
  │   │   ├── loader.go            # 规则加载
  │   │   └── types.go             # 规则定义
  │   └── processor/
  │       └── processor.go         # 批量处理逻辑
  ├── go.mod
  └── README.md


## 安装

```bash
go build -o /bin/yamleditor ./cmd/yamleditor
```

## 使用方法

### 配置文件

```bash
cp rules.example.yaml rules.yaml
```

### 单文件处理

```bash
# Dry-run 预览变更
yamleditor -c rules.yaml -i deployment.yaml --dry-run

# 输出到新文件
yamleditor -c rules.yaml -i input.yaml -o output.yaml

# 直接修改文件(带备份)
yamleditor -c rules.yaml -i deployment.yaml --backup
```

### 批量处理目录

```bash
# 批量处理(输出到新目录)
yamleditor -c rules.yaml -i ./input/ -o ./output/

# 原地批量修改(带备份)
yamleditor -c rules.yaml -i ./yamls/ --backup
```


## 路径语法

| 语法 | 说明 | 示例 |
|------|------|------|
| `field.subfield` | 字段访问 | `spec.template.metadata` |
| `field[*]` | 通配符(所有元素) | `containers[*]` |
| `field[0]` | 索引访问 | `containers[0]` |
| `field[name=value]` | 精确匹配 | `containers[name=nginx]` |
| `field[name=@pattern@]` | 正则匹配 | `env[name=@^xxx_.*$@]` |

## 操作类型


支持以下可选字段：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `continue_on_not_found` | bool | false | 当路径未找到节点时是否继续处理（不报错） |

**使用场景**: 批量处理多个文件时，某些规则可能不适用于所有文件。设置 `continue_on_not_found: true` 可忽略"未找到"错误，继续处理其他规则。

```yaml
rules:
  # 找不到时继续（适用于可选字段）
  - action: replace
    path: "metadata.labels.optional-label"
    value: "new-value"
    continue_on_not_found: true

  # 找不到时报错（默认行为，适用于必须存在的字段）
  - action: replace
    path: "metadata.name"
    value: "new-name"
```

### replace
替换节点（支持对象、字段、标量）:
```yaml
# 替换整个对象
- action: replace
  path: spec.initContainers[name=old]
  value:
    name: new-container
    image: new-image:v1

# 修改字段值
- action: replace
  path: spec.replicas
  value: 3

# 使用正则匹配
- action: replace
  path: volumes[name=@.*old.*@].name
  value: new-volume
```

**说明**: `replace` 通过路径定位节点后,用 `value` 替换该节点。
- 精确匹配: `[name=foo]`
- 正则匹配: `[name=@pattern@]` (用 `@...@` 包裹正则表达式)

### delete
删除节点:
```yaml
# 删除单个字段
- action: delete
  path: metadata.managedFields

# 使用正则删除（负向断言排除特定值）
- action: delete
  path: env[name=@^xxx_(?!(foo1|foo2)$).*@]
```

### regex_replace
正则替换字符串内容:
```yaml
- action: regex_replace
  path: containers[*].image
  pattern: 'registry.old.com'
  value: 'registry.new.com'
```

## License

MIT
