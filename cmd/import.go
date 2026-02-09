package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/kire"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import specs from external tools",
}

var importKireCmd = &cobra.Command{
	Use:   "kire",
	Short: "Import specs from kire output (JSONL + Markdown segments)",
	RunE:  runImportKire,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(importKireCmd)

	importKireCmd.Flags().String("dir", ".kire", "Directory containing kire segment files")
	importKireCmd.Flags().String("jsonl", ".kire/metadata.jsonl", "Path to kire JSONL metadata file")
	importKireCmd.Flags().Bool("force", false, "Overwrite existing spec files")
	importKireCmd.Flags().Bool("dry-run", false, "Preview without writing files")
}

var batchReqIDPattern = regexp.MustCompile(`REQ-(\d{3})`)

type importEntry struct {
	seg  *kire.Segment
	spec *spec.Spec
}

func runImportKire(cmd *cobra.Command, args []string) error {
	log := GetLogger().WithComponent("import")

	cfg, err := loadSpecConfig(cmd)
	if err != nil {
		return err
	}

	dir, _ := cmd.Flags().GetString("dir")
	jsonlPath, _ := cmd.Flags().GetString("jsonl")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	metas, err := kire.ParseJSONL(jsonlPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.SpecDir, 0755); err != nil {
		return err
	}

	// Phase 1: Read all segments and collect explicit IDs
	var segments []*kire.Segment
	maxExplicit := 0

	for _, meta := range metas {
		seg, err := kire.ReadSegment(dir, meta)
		if err != nil {
			return err
		}
		segments = append(segments, seg) // nil segments are kept for index alignment

		if seg != nil {
			if id := kire.ExtractReqID(seg.Content); id != "" {
				if m := batchReqIDPattern.FindStringSubmatch(id); len(m) == 2 {
					n, _ := strconv.Atoi(m[1])
					if n > maxExplicit {
						maxExplicit = n
					}
				}
			}
		}
	}

	// Phase 2: Assign IDs deterministically from batch content only
	autoNext := maxExplicit
	var entries []importEntry

	for i, seg := range segments {
		if seg == nil {
			log.Warn("segment file not found, skipping", "segment_id", metas[i].SegmentID, "file", metas[i].FilePath)
			fmt.Fprintf(cmd.OutOrStdout(), "warning: segment file not found: %s (%s)\n", metas[i].SegmentID, metas[i].FilePath)
			continue
		}

		title := ""
		if len(seg.Meta.HeadingPath) > 0 {
			title = seg.Meta.HeadingPath[len(seg.Meta.HeadingPath)-1]
		}

		id := kire.ExtractReqID(seg.Content)
		if id == "" {
			autoNext++
			id = fmt.Sprintf("REQ-%03d", autoNext)
		}

		examples := kire.ExtractExamples(seg.Content)
		for j := range examples {
			examples[j].ID = fmt.Sprintf("E%d", j+1)
		}
		questions := kire.ExtractQuestions(seg.Content)

		headingPath := make([]string, len(seg.Meta.HeadingPath))
		copy(headingPath, seg.Meta.HeadingPath)

		s := &spec.Spec{
			ID:        id,
			Title:     title,
			Examples:  examples,
			Questions: questions,
			Source: spec.SourceInfo{
				SegmentID:   seg.Meta.SegmentID,
				HeadingPath: headingPath,
			},
		}

		entries = append(entries, importEntry{seg: seg, spec: s})
	}

	// Phase 3: Save specs
	var created, skipped, overwritten int

	for _, entry := range entries {
		s := entry.spec
		specPath := filepath.Join(cfg.SpecDir, s.ID+".yml")

		if dryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] %s: %s (%s)\n", s.ID, s.Title, specPath)
			continue
		}

		_, statErr := os.Stat(specPath)
		fileExists := statErr == nil

		if fileExists && !force {
			fmt.Fprintf(cmd.OutOrStdout(), "skip: %s already exists\n", specPath)
			skipped++
			continue
		}

		s.Normalize()
		if err := spec.Save(specPath, s); err != nil {
			return err
		}

		if fileExists && force {
			fmt.Fprintf(cmd.OutOrStdout(), "overwritten: %s\n", specPath)
			overwritten++
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", specPath)
			created++
		}
	}

	if !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%d created, %d skipped, %d overwritten\n", created, skipped, overwritten)
	}

	return nil
}
