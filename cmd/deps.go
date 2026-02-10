package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/deps"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

// testDepsDetector はテスト用に Detector を差し替えるための変数。
var testDepsDetector deps.Detector

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Detect and update dependencies between specs",
	RunE:  runDeps,
}

func init() {
	rootCmd.AddCommand(depsCmd)

	depsCmd.Flags().Bool("enrich", false, "Use LLM for dependency detection (requires GEMINI_API_KEY)")
	depsCmd.Flags().String("model", "gemini-2.5-flash", "Gemini model name for dependency detection")
	depsCmd.Flags().Duration("timeout", 60*time.Second, "Timeout for Gemini API call")
	depsCmd.Flags().Bool("dry-run", false, "Preview dependencies without writing files")
}

func runDeps(cmd *cobra.Command, args []string) error {
	log := GetLogger().WithComponent("deps")

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

	enrichEnabled, _ := cmd.Flags().GetBool("enrich")
	model, _ := cmd.Flags().GetString("model")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Select detector
	var detector deps.Detector
	if testDepsDetector != nil {
		detector = testDepsDetector
	} else if enrichEnabled {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY is required when --enrich is enabled")
		}
		d, err := deps.NewGeminiDetector(deps.GeminiDetectorConfig{
			APIKey:  apiKey,
			Model:   model,
			Timeout: timeout,
		})
		if err != nil {
			return err
		}
		detector = d
	} else {
		detector = &deps.HeuristicDetector{}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "detecting dependencies for %d specs ...\n", len(specs))

	results, err := detector.Detect(context.Background(), specs)
	if err != nil {
		return err
	}

	// Build result map
	depsMap := make(map[string]deps.DepsResult, len(results))
	for _, r := range results {
		depsMap[r.ID] = r
	}

	// Update specs
	updatedCount := 0
	for _, s := range specs {
		r, ok := depsMap[s.ID]
		if !ok {
			continue
		}

		if dryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] %s depends on %v (%s)\n", s.ID, r.Depends, r.Reason)
			continue
		}

		s.Depends = r.Depends
		specPath := filepath.Join(cfg.SpecDir, s.ID+".yml")
		if err := spec.Save(specPath, s); err != nil {
			log.Error("Failed to save spec", "id", s.ID, "error", err)
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "updated: %s -> %v\n", s.ID, r.Depends)
		updatedCount++
	}

	// Validate graph (warning only)
	if !dryRun && updatedCount > 0 {
		// Reload to get updated specs
		updatedSpecs, err := spec.LoadAll(cfg.SpecDir)
		if err != nil {
			log.Warn("Failed to reload specs for graph validation", "error", err)
		} else if err := spec.ValidateDependsGraph(updatedSpecs); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "warning: %v\n", err)
		}
	}

	if dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%d dependencies detected (dry-run)\n", len(results))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%d specs updated\n", updatedCount)
	}

	return nil
}
