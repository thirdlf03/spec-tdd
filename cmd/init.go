package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize spec-tdd workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger().WithComponent("init")

		cfg := config.DefaultSpecConfig()
		if err := os.MkdirAll(cfg.SpecDir, 0755); err != nil {
			log.Error("Failed to create spec directory", "dir", cfg.SpecDir, "error", err)
			return err
		}

		configDir := filepath.Dir(config.DefaultSpecConfigPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Error("Failed to create config directory", "dir", configDir, "error", err)
			return err
		}

		if _, err := os.Stat(config.DefaultSpecConfigPath); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "config already exists: %s\n", config.DefaultSpecConfigPath)
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if err := config.SaveSpecConfig(config.DefaultSpecConfigPath, cfg); err != nil {
			log.Error("Failed to write config", "path", config.DefaultSpecConfigPath, "error", err)
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", config.DefaultSpecConfigPath)
		fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", cfg.SpecDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
