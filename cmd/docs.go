package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var (
	docsOutputDir string
	docsFormat    string
)

// docsCmd generates documentation for all commands
var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation for all commands",
	Long: `Generate documentation in various formats (markdown, man, rest, yaml).

The documentation will be generated in the specified output directory.
By default, it generates markdown documentation in the ./docs directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger().WithComponent("docs")

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(docsOutputDir, 0755); err != nil {
			log.Error("Failed to create output directory",
				"dir", docsOutputDir,
				"error", err)
			return err
		}

		log.Info("Generating documentation",
			"format", docsFormat,
			"output", docsOutputDir)

		var err error
		switch docsFormat {
		case "markdown", "md":
			err = doc.GenMarkdownTree(rootCmd, docsOutputDir)
		case "man":
			header := &doc.GenManHeader{
				Title:   "SPEC-TDD",
				Section: "1",
			}
			err = doc.GenManTree(rootCmd, header, docsOutputDir)
		case "rest", "rst":
			err = doc.GenReSTTree(rootCmd, docsOutputDir)
		case "yaml", "yml":
			err = doc.GenYamlTree(rootCmd, docsOutputDir)
		default:
			log.Error("Unsupported format", "format", docsFormat)
			return fmt.Errorf("unsupported format: %s", docsFormat)
		}

		if err != nil {
			log.Error("Failed to generate documentation", "error", err)
			return err
		}

		absPath, _ := filepath.Abs(docsOutputDir)
		log.Info("Documentation generated successfully", "path", absPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)

	docsCmd.Flags().StringVarP(&docsOutputDir, "output", "o", "./docs", "Output directory for documentation")
	docsCmd.Flags().StringVarP(&docsFormat, "format", "f", "markdown", "Documentation format (markdown, man, rest, yaml)")
}
