package processor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"yamleditor/pkg/engine"
	"yamleditor/pkg/rule"
)

// Processor 批量处理 YAML 文件
type Processor struct {
	rules  []*engine.Rule
	engine *engine.Engine
}

// NewProcessor 创建处理器
func NewProcessor(ruleFile string) (*Processor, error) {
	rules, err := rule.LoadFromFile(ruleFile)
	if err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}

	return &Processor{
		rules:  rules,
		engine: engine.NewEngine(),
	}, nil
}

// ProcessFile 处理单个 YAML 文件
func (p *Processor) ProcessFile(inputPath, outputPath string, dryRun bool) error {
	// 读取文件
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// 检测并移除 UTF-8 BOM
	hasBOM := false
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		hasBOM = true
		data = data[3:] // 移除BOM，传递给yaml解析器
	}

	// 解析 YAML
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	// 应用所有规则
	for i, r := range p.rules {
		if err := p.engine.Apply(&root, r); err != nil {
			// 某些规则可能找不到节点（比如文件中没有那个字段），这是正常的
			// 只在非预期错误时报错
			if !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("apply rule %d: %w", i, err)
			}
		}
	}

	// 序列化 YAML（保持2空格缩进）
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	encoder.Close()
	output := []byte(buf.String())

	// 如果原文件有 BOM，添加回去
	if hasBOM {
		output = append([]byte{0xEF, 0xBB, 0xBF}, output...)
	}

	if dryRun {
		fmt.Printf("=== Dry-run: %s ===\n", inputPath)
		fmt.Println(string(output))
		fmt.Println()
		return nil
	}

	// 写入文件
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// ProcessDirectory 批量处理目录下的所有 YAML 文件
func (p *Processor) ProcessDirectory(inputDir, outputDir string, dryRun, backup bool) error {
	// 确保输出目录存在
	if !dryRun && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	// 遍历目录
	return filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理 .yaml 和 .yml 文件
		if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}

		// 计算输出路径
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		var outputPath string
		if outputDir != "" {
			outputPath = filepath.Join(outputDir, relPath)
		} else {
			outputPath = path // 原地修改
		}

		// 备份
		if backup && !dryRun && outputDir == "" {
			backupPath := path + ".bak"
			if err := copyFile(path, backupPath); err != nil {
				return fmt.Errorf("backup file: %w", err)
			}
		}

		// 处理文件
		fmt.Printf("Processing: %s\n", path)
		if err := p.ProcessFile(path, outputPath, dryRun); err != nil {
			return fmt.Errorf("process %s: %w", path, err)
		}

		return nil
	})
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
