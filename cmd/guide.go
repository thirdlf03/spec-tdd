package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/config"
	"github.com/thirdlf03/spec-tdd/internal/guide"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Generate implementation guide from specs and dependencies",
	RunE:  runGuide,
}

func init() {
	rootCmd.AddCommand(guideCmd)

	guideCmd.Flags().String("output", ".tdd/GUIDE.md", "Output path for the guide")
}

func runGuide(cmd *cobra.Command, args []string) error {
	log := GetLogger().WithComponent("guide")

	cfg, err := loadSpecConfig(cmd)
	if err != nil {
		return err
	}

	specs, err := spec.LoadAll(cfg.SpecDir)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "no specs found\n")
		return nil
	}

	// Validate dependency graph (error, not warning)
	if err := spec.ValidateDependsGraph(specs); err != nil {
		return fmt.Errorf("dependency graph validation failed: %w", err)
	}

	order, err := guide.TopologicalSort(specs)
	if err != nil {
		return err
	}

	depBy := guide.BuildDependedByMap(specs)

	data := guide.GuideData{
		Specs:         specs,
		Order:         order,
		Prerequisites: make(map[string][]string, len(specs)),
		DependedBy:    depBy,
	}
	for _, s := range specs {
		data.Prerequisites[s.ID] = s.Depends
	}

	content := guide.RenderGuide(data, cfg.TestDir, cfg.FileNamePattern)

	outputPath, _ := cmd.Flags().GetString("output")
	outputDir := filepath.Dir(outputPath)
	if outputDir == "" {
		outputDir = filepath.Dir(config.DefaultSpecConfigPath)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Error("Failed to create output directory", "dir", outputDir, "error", err)
		return err
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		log.Error("Failed to write guide", "path", outputPath, "error", err)
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", outputPath)
	return nil
}
