package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/config"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func hasSource(s *spec.Spec) bool {
	return s.Source.SegmentID != "" || len(s.Source.HeadingPath) > 0 || s.Source.FilePath != ""
}

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Generate example mapping report",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger().WithComponent("map")

		cfg, err := loadSpecConfig(cmd)
		if err != nil {
			return err
		}

		specs, err := spec.LoadAll(cfg.SpecDir)
		if err != nil {
			return err
		}

		outputDir := filepath.Dir(config.DefaultSpecConfigPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Error("Failed to create output directory", "dir", outputDir, "error", err)
			return err
		}

		outputPath := filepath.Join(outputDir, "map.md")
		content := renderMapMarkdown(specs)
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			log.Error("Failed to write map", "path", outputPath, "error", err)
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", outputPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
}

func renderMapMarkdown(specs []*spec.Spec) string {
	var sb strings.Builder
	sb.WriteString("# Example Mapping\n\n")
	for _, s := range specs {
		sb.WriteString(fmt.Sprintf("## %s: %s\n\n", s.ID, s.Title))
		if strings.TrimSpace(s.Description) != "" {
			sb.WriteString(s.Description)
			sb.WriteString("\n\n")
		}

		if hasSource(s) {
			src := fmt.Sprintf("Source: segment_id=%s, heading_path=%s",
				s.Source.SegmentID, strings.Join(s.Source.HeadingPath, " > "))
			if s.Source.FilePath != "" {
				src += fmt.Sprintf(", file_path=%s", s.Source.FilePath)
			}
			sb.WriteString(src + "\n\n")
		}

		if len(s.Examples) > 0 {
			sb.WriteString("Examples:\n")
			for _, ex := range s.Examples {
				id := ex.ID
				if id == "" {
					id = "E?"
				}
				sb.WriteString(fmt.Sprintf("- %s: Given %s / When %s / Then %s\n", id, ex.Given, ex.When, ex.Then))
			}
			sb.WriteString("\n")
		}

		if len(s.Questions) > 0 {
			sb.WriteString("Questions:\n")
			for _, q := range s.Questions {
				sb.WriteString(fmt.Sprintf("- %s\n", q))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
