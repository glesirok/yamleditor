package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"yamleditor/pkg/processor"
)

var (
	ruleFile string
	input    string
	output   string
	dryRun   bool
	backup   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "yamleditor",
		Short: "Batch edit Kubernetes YAML files",
		Long: `yamleditor is a tool to batch edit YAML files using configurable rules.
It supports path-based operations like replace, set, delete, and regex_replace.`,
		RunE: run,
	}

	rootCmd.Flags().StringVarP(&ruleFile, "config", "c", "", "Rule configuration file (required)")
	rootCmd.Flags().StringVarP(&input, "input", "i", "", "Input file or directory (required)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output file/directory (optional, defaults to in-place)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry-run mode: preview changes without writing files")
	rootCmd.Flags().BoolVar(&backup, "backup", false, "Backup original files with .bak extension")

	rootCmd.MarkFlagRequired("config")
	rootCmd.MarkFlagRequired("input")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// 创建处理器
	proc, err := processor.NewProcessor(ruleFile)
	if err != nil {
		return fmt.Errorf("create processor: %w", err)
	}

	// 判断输入类型
	info, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("stat input: %w", err)
	}

	if info.IsDir() {
		// 目录模式
		return processDirectory(proc, input, output)
	}

	// 文件模式
	return processFile(proc, input, output)
}

func processFile(proc *processor.Processor, inputFile, outputFile string) error {
	if outputFile == "" {
		outputFile = inputFile // 默认原地覆盖
	}

	if backup && !dryRun && outputFile == inputFile {
		// 只有原地覆盖才备份
		backupPath := inputFile + ".bak"
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("read file for backup: %w", err)
		}
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
	}

	if err := proc.ProcessFile(inputFile, outputFile, dryRun); err != nil {
		return err
	}

	if !dryRun {
		if outputFile == inputFile {
			fmt.Printf("✓ Processed: %s\n", inputFile)
		} else {
			fmt.Printf("✓ Processed: %s → %s\n", inputFile, outputFile)
		}
	}
	return nil
}

func processDirectory(proc *processor.Processor, inputDir, outputDir string) error {
	if err := proc.ProcessDirectory(inputDir, outputDir, dryRun, backup); err != nil {
		return err
	}

	if !dryRun {
		fmt.Println("✓ All files processed successfully")
	}
	return nil
}
