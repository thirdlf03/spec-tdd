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
	reqAddTitle string
	reqAddID    string
)

var reqCmd = &cobra.Command{
	Use:   "req",
	Short: "Manage requirement specs",
}

var reqAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new requirement",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger().WithComponent("req.add")

		cfg, err := loadSpecConfig(cmd)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(cfg.SpecDir, 0755); err != nil {
			log.Error("Failed to create spec directory", "dir", cfg.SpecDir, "error", err)
			return err
		}

		id := strings.TrimSpace(reqAddID)
		if id == "" {
			id, err = spec.NextReqID(cfg.SpecDir)
			if err != nil {
				return err
			}
		}

		filePath := filepath.Join(cfg.SpecDir, fmt.Sprintf("%s.yml", id))
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("spec already exists: %s", filePath)
		}

		newSpec := &spec.Spec{
			ID:    id,
			Title: reqAddTitle,
		}

		if err := spec.Save(filePath, newSpec); err != nil {
			log.Error("Failed to save spec", "path", filePath, "error", err)
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", filePath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reqCmd)
	reqCmd.AddCommand(reqAddCmd)

	reqAddCmd.Flags().StringVar(&reqAddTitle, "title", "", "Requirement title")
	reqAddCmd.Flags().StringVar(&reqAddID, "id", "", "Requirement ID (e.g., REQ-001)")
	_ = reqAddCmd.MarkFlagRequired("title")
}
