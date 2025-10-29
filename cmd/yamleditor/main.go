package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"yamleditor/pkg/processor"
)

var (
	ruleFile  string
	inputFile string
	inputDir  string
	outputDir string
	dryRun    bool
	backup    bool
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
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input YAML file")
	rootCmd.Flags().StringVarP(&inputDir, "dir", "d", "", "Input directory (batch mode)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (optional, defaults to in-place)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry-run mode: preview changes without writing files")
	rootCmd.Flags().BoolVar(&backup, "backup", false, "Backup original files with .bak extension")

	rootCmd.MarkFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// 参数校验
	if inputFile == "" && inputDir == "" {
		return fmt.Errorf("either --input or --dir must be specified")
	}

	if inputFile != "" && inputDir != "" {
		return fmt.Errorf("--input and --dir cannot be used together")
	}

	// 创建处理器
	proc, err := processor.NewProcessor(ruleFile)
	if err != nil {
		return fmt.Errorf("create processor: %w", err)
	}

	// 单文件模式
	if inputFile != "" {
		output := inputFile
		if outputDir != "" {
			return fmt.Errorf("--output is not supported in single file mode, use --dir for batch mode")
		}

		if backup && !dryRun {
			backupPath := inputFile + ".bak"
			data, err := os.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("read file for backup: %w", err)
			}
			if err := os.WriteFile(backupPath, data, 0644); err != nil {
				return fmt.Errorf("create backup: %w", err)
			}
		}

		if err := proc.ProcessFile(inputFile, output, dryRun); err != nil {
			return err
		}

		if !dryRun {
			fmt.Printf("✓ Processed: %s\n", inputFile)
		}
		return nil
	}

	// 批量模式
	if inputDir != "" {
		if err := proc.ProcessDirectory(inputDir, outputDir, dryRun, backup); err != nil {
			return err
		}

		if !dryRun {
			fmt.Println("✓ All files processed successfully")
		}
		return nil
	}

	return nil
}
