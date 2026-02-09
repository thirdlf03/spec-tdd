package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/config"
)

func loadSpecConfig(cmd *cobra.Command) (config.SpecConfig, error) {
	cfg, loaded, err := config.LoadSpecConfig(config.DefaultSpecConfigPath)
	if err != nil {
		return config.SpecConfig{}, err
	}
	if !loaded {
		fmt.Fprintf(cmd.OutOrStdout(), "config not found, using defaults at %s\n", config.DefaultSpecConfigPath)
	}
	return cfg, nil
}
