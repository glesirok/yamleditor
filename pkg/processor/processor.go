package processor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/glesirok/yamleditor/pkg/engine"
	"github.com/glesirok/yamleditor/pkg/rule"
)

// ProcessResult 批量处理结果
type ProcessResult struct {
	TotalFiles   int
	SuccessFiles int
	FailedFiles  []FailedFile
}

// FailedFile 失败文件信息
type FailedFile struct {
	Path  string
	Error error
}

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
			// 由规则logic决定是否忽略错误
			return fmt.Errorf("apply rule %d, path:{%s}: %w", i, r.Path, err)
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

	// 确保输出目录存在
	if outputDir := filepath.Dir(outputPath); outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	// 写入文件
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// ProcessDirectory 批量处理目录下的所有 YAML 文件
func (p *Processor) ProcessDirectory(inputDir, outputDir string, dryRun, backup bool) (*ProcessResult, error) {
	result := &ProcessResult{}

	// 遍历目录
	walkErr := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// filepath.Walk 本身的错误（如权限问题），直接返回终止遍历
			return err
		}

		// 只处理 .yaml 和 .yml 文件
		if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}

		result.TotalFiles++

		// 计算输出路径
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			result.FailedFiles = append(result.FailedFiles, FailedFile{
				Path:  path,
				Error: fmt.Errorf("compute relative path: %w", err),
			})
			return nil // 继续处理下一个文件
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
				result.FailedFiles = append(result.FailedFiles, FailedFile{
					Path:  path,
					Error: fmt.Errorf("backup file: %w", err),
				})
				return nil // 继续处理下一个文件
			}
		}

		// 处理文件
		fmt.Printf("Processing: %s\n", path)
		if err := p.ProcessFile(path, outputPath, dryRun); err != nil {
			result.FailedFiles = append(result.FailedFiles, FailedFile{
				Path:  path,
				Error: err,
			})
			return nil // 继续处理下一个文件
		}

		result.SuccessFiles++
		return nil
	})

	// 如果 Walk 本身出错（系统级错误），返回 error
	if walkErr != nil {
		return result, walkErr
	}

	return result, nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
