package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/config"
	"github.com/thirdlf03/spec-tdd/internal/spec"
	"github.com/thirdlf03/spec-tdd/internal/trace"
)

var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Generate traceability report",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger().WithComponent("trace")

		cfg, err := loadSpecConfig(cmd)
		if err != nil {
			return err
		}

		specs, err := spec.LoadAll(cfg.SpecDir)
		if err != nil {
			return err
		}

		counts, err := trace.CountTestsByReq(cfg.TestDir)
		if err != nil {
			return err
		}

		report := trace.BuildReport(specs, counts)

		outputDir := filepath.Dir(config.DefaultSpecConfigPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Error("Failed to create output directory", "dir", outputDir, "error", err)
			return err
		}

		jsonPath := filepath.Join(outputDir, "trace.json")
		data, err := report.ToJSON()
		if err != nil {
			return err
		}
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			log.Error("Failed to write trace JSON", "path", jsonPath, "error", err)
			return err
		}

		mdPath := filepath.Join(outputDir, "trace.md")
		if err := os.WriteFile(mdPath, []byte(report.ToMarkdown()), 0644); err != nil {
			log.Error("Failed to write trace markdown", "path", mdPath, "error", err)
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", jsonPath)
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", mdPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(traceCmd)
}
