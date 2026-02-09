package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

var (
	exampleReqID string
	exampleGiven string
	exampleWhen  string
	exampleThen  string
)

var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Manage requirement examples",
}

var exampleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an example to a requirement",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger().WithComponent("example.add")

		cfg, err := loadSpecConfig(cmd)
		if err != nil {
			return err
		}

		reqID := strings.TrimSpace(exampleReqID)
		if reqID == "" {
			return fmt.Errorf("--req is required")
		}

		path := filepath.Join(cfg.SpecDir, fmt.Sprintf("%s.yml", reqID))
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("spec not found: %s", path)
		}

		s, err := spec.Load(path)
		if err != nil {
			return err
		}

		newExample := spec.Example{
			ID:    spec.NextExampleID(s),
			Given: exampleGiven,
			When:  exampleWhen,
			Then:  exampleThen,
		}
		s.Examples = append(s.Examples, newExample)

		if err := spec.Save(path, s); err != nil {
			log.Error("Failed to save spec", "path", path, "error", err)
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "added %s to %s\n", newExample.ID, path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)
	exampleCmd.AddCommand(exampleAddCmd)

	exampleAddCmd.Flags().StringVar(&exampleReqID, "req", "", "Requirement ID (e.g., REQ-001)")
	exampleAddCmd.Flags().StringVar(&exampleGiven, "given", "", "Given clause")
	exampleAddCmd.Flags().StringVar(&exampleWhen, "when", "", "When clause")
	exampleAddCmd.Flags().StringVar(&exampleThen, "then", "", "Then clause")
	_ = exampleAddCmd.MarkFlagRequired("req")
	_ = exampleAddCmd.MarkFlagRequired("given")
	_ = exampleAddCmd.MarkFlagRequired("when")
	_ = exampleAddCmd.MarkFlagRequired("then")
}
