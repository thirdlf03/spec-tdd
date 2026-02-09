package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/scaffold"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

var (
	scaffoldRunner string
	scaffoldForce  bool
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Generate test scaffolds from specs",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger().WithComponent("scaffold")

		cfg, err := loadSpecConfig(cmd)
		if err != nil {
			return err
		}

		runner := cfg.Runner
		if strings.TrimSpace(scaffoldRunner) != "" {
			runner = scaffoldRunner
		}
		if runner != "vitest" && runner != "jest" {
			return fmt.Errorf("unsupported runner: %s", runner)
		}

		specs, err := spec.LoadAll(cfg.SpecDir)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(cfg.TestDir, 0755); err != nil {
			log.Error("Failed to create test directory", "dir", cfg.TestDir, "error", err)
			return err
		}

		for _, s := range specs {
			slug := scaffold.Slugify(s.Title)
			fileName := scaffold.ApplyPattern(cfg.FileNamePattern, s.ID, slug)
			path := filepath.Join(cfg.TestDir, fileName)

			if _, err := os.Stat(path); err == nil && !scaffoldForce {
				return fmt.Errorf("test file exists: %s (use --force to overwrite)", path)
			}

			content := scaffold.RenderTest(s, runner)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				log.Error("Failed to write test file", "path", path, "error", err)
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scaffoldCmd)

	scaffoldCmd.Flags().StringVar(&scaffoldRunner, "runner", "", "Override test runner (vitest or jest)")
	scaffoldCmd.Flags().BoolVar(&scaffoldForce, "force", false, "Overwrite existing test files")
}
