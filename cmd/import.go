package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thirdlf03/spec-tdd/internal/enrich"
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

// testEnricher はテスト用に Enricher を差し替えるための変数。
// nil の場合は実際の GeminiEnricher を使用する。
var testEnricher enrich.Enricher

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(importKireCmd)

	importKireCmd.Flags().String("dir", ".kire", "Directory containing kire segment files")
	importKireCmd.Flags().String("jsonl", ".kire/metadata.jsonl", "Path to kire JSONL metadata file")
	importKireCmd.Flags().Bool("force", false, "Overwrite existing spec files")
	importKireCmd.Flags().Bool("dry-run", false, "Preview without writing files")
	importKireCmd.Flags().Bool("enrich", false, "Enable LLM enrichment (requires GEMINI_API_KEY)")
	importKireCmd.Flags().String("enrich-model", "gemini-2.5-flash-lite", "Gemini model name for enrichment")
	importKireCmd.Flags().Duration("enrich-timeout", 30*time.Second, "Timeout for each Gemini API call")
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
	enrichEnabled, _ := cmd.Flags().GetBool("enrich")
	enrichModel, _ := cmd.Flags().GetString("enrich-model")
	enrichTimeout, _ := cmd.Flags().GetDuration("enrich-timeout")

	// Initialize enricher if --enrich is enabled
	var enricher enrich.Enricher
	if enrichEnabled {
		if testEnricher != nil {
			enricher = testEnricher
		} else {
			apiKey := os.Getenv("GEMINI_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("GEMINI_API_KEY is required when --enrich is enabled. Set it with: export GEMINI_API_KEY=your-key")
			}
			e, err := enrich.NewGeminiEnricher(enrich.GeminiEnricherConfig{
				APIKey:  apiKey,
				Model:   enrichModel,
				Timeout: enrichTimeout,
			})
			if err != nil {
				return err
			}
			enricher = e
		}
	}

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

	// Phase 2: Assign IDs deterministically
	autoNext := maxExplicit
	var entries []importEntry
	var enrichedCount, skippedEnrich, errorsEnrich int

	for i, seg := range segments {
		if seg == nil {
			log.Warn("segment file not found, skipping", "segment_id", metas[i].SegmentID, "file", metas[i].FilePath)
			fmt.Fprintf(cmd.OutOrStdout(), "warning: segment file not found: %s (%s)\n", metas[i].SegmentID, metas[i].FilePath)
			continue
		}

		if enrichEnabled && enricher != nil {
			// Enrichment pipeline
			fmt.Fprintf(cmd.OutOrStdout(), "enriching: %s ... ", seg.Meta.SegmentID)

			result, err := enricher.Enrich(context.Background(), seg)
			if err != nil {
				// Fallback to existing logic
				log.Warn("enrichment failed, falling back to regex extraction", "segment_id", seg.Meta.SegmentID, "error", err)
				fmt.Fprintf(cmd.OutOrStdout(), "error (fallback)\n")
				errorsEnrich++
				entry := buildEntryFromRegex(seg, &autoNext)
				entries = append(entries, entry)
				continue
			}

			if !result.IsRequirement() {
				fmt.Fprintf(cmd.OutOrStdout(), "skipped (%s)\n", result.Category)
				skippedEnrich++
				continue
			}

			fmt.Fprintf(cmd.OutOrStdout(), "done\n")
			enrichedCount++

			// Use enrichment result
			id := ""
			if result.ReqID != "" {
				// Normalize LLM-returned req_id (may contain extra text)
				id = kire.ExtractReqID(result.ReqID)
			}
			if id == "" {
				// Fallback: extract from segment content
				if regexID, regexTitle := kire.ExtractReqIDWithTitle(seg.Content); regexID != "" {
					id = regexID
					if result.Title == "" {
						result.Title = regexTitle
					}
				}
			}
			if id == "" {
				autoNext++
				id = fmt.Sprintf("REQ-%03d", autoNext)
			} else {
				// Track explicit ID
				if m := batchReqIDPattern.FindStringSubmatch(id); len(m) == 2 {
					n, _ := strconv.Atoi(m[1])
					if n > maxExplicit {
						maxExplicit = n
						autoNext = maxExplicit
					}
				}
			}

			title := result.Title
			if title == "" && len(seg.Meta.HeadingPath) > 0 {
				title = seg.Meta.HeadingPath[len(seg.Meta.HeadingPath)-1]
			}
			if title == "" {
				title = extractFirstHeading(seg.Content)
			}
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("empty title for segment %s after enrichment", seg.Meta.SegmentID)
			}

			// Merge examples: existing GWT > enriched GWT
			existingExamples := kire.ExtractExamples(seg.Content)
			examples := enrich.MergeExamples(existingExamples, result.Examples)

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
		} else {
			// Existing logic (no enrichment)
			entry := buildEntryFromRegex(seg, &autoNext)
			entries = append(entries, entry)
		}
	}

	// Check for duplicate REQ-IDs
	if enrichEnabled {
		var ids []string
		for _, entry := range entries {
			ids = append(ids, entry.spec.ID)
		}
		if err := kire.CheckDuplicateReqIDs(ids); err != nil {
			return err
		}
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

	if enrichEnabled && !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%d enriched, %d skipped, %d errors\n", enrichedCount, skippedEnrich, errorsEnrich)
	}

	return nil
}

// buildEntryFromRegex は既存のロジック（正規表現ベース）でセグメントを変換する。
func buildEntryFromRegex(seg *kire.Segment, autoNext *int) importEntry {
	title := ""
	if len(seg.Meta.HeadingPath) > 0 {
		title = seg.Meta.HeadingPath[len(seg.Meta.HeadingPath)-1]
	}
	if title == "" {
		title = extractFirstHeading(seg.Content)
	}

	id := kire.ExtractReqID(seg.Content)
	if id == "" {
		*autoNext++
		id = fmt.Sprintf("REQ-%03d", *autoNext)
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

	return importEntry{seg: seg, spec: s}
}

// extractFirstHeading returns the text of the first Markdown heading in content.
func extractFirstHeading(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		}
	}
	return ""
}
