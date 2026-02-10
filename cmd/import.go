package cmd

import (
	"context"
	"errors"
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

// testBatchEnricher はテスト用に BatchEnricher を差し替えるための変数。
var testBatchEnricher enrich.BatchEnricher

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
	importKireCmd.Flags().String("enrich-example-model", "", "Example generation model (enables 2-pass batch mode)")
	importKireCmd.Flags().Duration("enrich-example-timeout", 120*time.Second, "Timeout for batch example generation")
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
	enrichExampleModel, _ := cmd.Flags().GetString("enrich-example-model")
	enrichExampleTimeout, _ := cmd.Flags().GetDuration("enrich-example-timeout")

	// 2-pass batch mode: --enrich --enrich-example-model <model>
	batchMode := enrichEnabled && enrichExampleModel != ""

	// Initialize enricher if --enrich is enabled (1-pass mode)
	var enricher enrich.Enricher
	var batchEnricher enrich.BatchEnricher
	if enrichEnabled {
		if batchMode {
			// 2-pass batch mode
			if testBatchEnricher != nil {
				batchEnricher = testBatchEnricher
			} else {
				apiKey := os.Getenv("GEMINI_API_KEY")
				if apiKey == "" {
					return fmt.Errorf("GEMINI_API_KEY is required when --enrich is enabled. Set it with: export GEMINI_API_KEY=your-key")
				}
				be, err := enrich.NewGeminiBatchEnricher(enrich.GeminiBatchEnricherConfig{
					APIKey:          apiKey,
					ClassifyModel:   enrichModel,
					ExampleModel:    enrichExampleModel,
					ClassifyTimeout: enrichTimeout,
					ExampleTimeout:  enrichExampleTimeout,
				})
				if err != nil {
					return err
				}
				batchEnricher = be
			}
		} else {
			// 1-pass mode
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

	// Filter out nil segments
	var validSegments []*kire.Segment
	for i, seg := range segments {
		if seg == nil {
			log.Warn("segment file not found, skipping", "segment_id", metas[i].SegmentID, "file", metas[i].FilePath)
			fmt.Fprintf(cmd.OutOrStdout(), "warning: segment file not found: %s (%s)\n", metas[i].SegmentID, metas[i].FilePath)
		} else {
			validSegments = append(validSegments, seg)
		}
	}

	// 2-pass batch mode
	if batchMode && batchEnricher != nil {
		entries, err := runBatchEnrich(cmd, log, batchEnricher, validSegments, maxExplicit)
		if err != nil {
			return err
		}
		if err := saveEntries(cmd, entries, cfg.SpecDir, force, dryRun); err != nil {
			return err
		}
		if !dryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "%d enriched\n", len(entries))
		}
		return nil
	}

	// Phase 2: Assign IDs deterministically (1-pass or no-enrich)
	autoNext := maxExplicit
	var entries []importEntry
	var enrichedCount, skippedEnrich, errorsEnrich int

	for _, seg := range validSegments {
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

			if !result.IsExampleTarget() {
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
					FilePath:    seg.Meta.FilePath,
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

	if err := saveEntries(cmd, entries, cfg.SpecDir, force, dryRun); err != nil {
		return err
	}

	if enrichEnabled && !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%d enriched, %d skipped, %d errors\n", enrichedCount, skippedEnrich, errorsEnrich)
	}

	return nil
}

// runBatchEnrich は 2-pass バッチ enrichment を実行する。
func runBatchEnrich(cmd *cobra.Command, log interface{ Warn(string, ...any) }, be enrich.BatchEnricher, segments []*kire.Segment, maxExplicit int) ([]importEntry, error) {
	if len(segments) == 0 {
		return nil, nil
	}

	// Call 1: BatchClassify
	fmt.Fprintf(cmd.OutOrStdout(), "classifying %d segments ... ", len(segments))
	classifyResults, err := be.BatchClassify(context.Background(), segments)
	if err != nil && !errors.Is(err, enrich.ErrBatchTruncated) {
		return nil, fmt.Errorf("batch classify failed: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "done\n")

	// Build segment lookup map
	segMap := make(map[string]*kire.Segment, len(segments))
	for _, seg := range segments {
		segMap[seg.Meta.SegmentID] = seg
	}

	// Filter example-target segments (FR + NFR) and assign IDs
	autoNext := maxExplicit
	var targetSegments []*kire.Segment
	var entries []importEntry
	titleMap := make(map[string]string)   // segment_id -> title
	idMap := make(map[string]string)      // segment_id -> REQ-ID
	skippedCount := 0

	for _, cr := range classifyResults {
		seg, ok := segMap[cr.SegmentID]
		if !ok {
			continue
		}

		category := enrich.NormalizeCategory(string(cr.Category))
		if !category.IsExampleTarget() {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: skipped (%s)\n", cr.SegmentID, category)
			skippedCount++
			continue
		}

		// Resolve REQ-ID
		id := ""
		if cr.ReqID != "" {
			id = kire.ExtractReqID(cr.ReqID)
		}
		if id == "" {
			if regexID, regexTitle := kire.ExtractReqIDWithTitle(seg.Content); regexID != "" {
				id = regexID
				if cr.Title == "" {
					cr.Title = regexTitle
				}
			}
		}
		if id == "" {
			autoNext++
			id = fmt.Sprintf("REQ-%03d", autoNext)
		} else {
			if m := batchReqIDPattern.FindStringSubmatch(id); len(m) == 2 {
				n, _ := strconv.Atoi(m[1])
				if n > maxExplicit {
					maxExplicit = n
					autoNext = maxExplicit
				}
			}
		}

		// Resolve title
		title := strings.TrimSpace(cr.Title)
		if title == "" && len(seg.Meta.HeadingPath) > 0 {
			title = seg.Meta.HeadingPath[len(seg.Meta.HeadingPath)-1]
		}
		if title == "" {
			title = extractFirstHeading(seg.Content)
		}
		if title == "" {
			return nil, fmt.Errorf("empty title for segment %s after batch classify", cr.SegmentID)
		}

		targetSegments = append(targetSegments, seg)
		idMap[cr.SegmentID] = id
		titleMap[cr.SegmentID] = title
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  %d requirements (FR+NFR), %d skipped\n", len(targetSegments), skippedCount)

	// Call 2: BatchGenerateExamples for FR segments
	exampleMap := make(map[string][]spec.Example)
	if len(targetSegments) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "generating examples for %d segments ... ", len(targetSegments))
		exResults, err := batchGenerateWithFallback(context.Background(), be, targetSegments)
		if err != nil {
			log.Warn("batch example generation failed", "error", err)
			fmt.Fprintf(cmd.OutOrStdout(), "error\n")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "done\n")
			for _, er := range exResults {
				exampleMap[er.SegmentID] = er.Examples
			}
		}
	}

	// Build entries
	for _, seg := range targetSegments {
		sid := seg.Meta.SegmentID
		id := idMap[sid]
		title := titleMap[sid]

		existingExamples := kire.ExtractExamples(seg.Content)
		enrichedExamples := exampleMap[sid]
		examples := enrich.MergeExamples(existingExamples, enrichedExamples)

		questions := kire.ExtractQuestions(seg.Content)

		headingPath := make([]string, len(seg.Meta.HeadingPath))
		copy(headingPath, seg.Meta.HeadingPath)

		s := &spec.Spec{
			ID:        id,
			Title:     title,
			Examples:  examples,
			Questions: questions,
			Source: spec.SourceInfo{
				SegmentID:   sid,
				HeadingPath: headingPath,
				FilePath:    seg.Meta.FilePath,
			},
		}
		entries = append(entries, importEntry{seg: seg, spec: s})
	}

	// Check for duplicate REQ-IDs
	var ids []string
	for _, entry := range entries {
		ids = append(ids, entry.spec.ID)
	}
	if err := kire.CheckDuplicateReqIDs(ids); err != nil {
		return nil, err
	}

	return entries, nil
}

// batchGenerateWithFallback は BatchGenerateExamples を呼び、切断時に2分割リトライする。
func batchGenerateWithFallback(ctx context.Context, be enrich.BatchEnricher, targetSegments []*kire.Segment) ([]enrich.BatchExampleResult, error) {
	results, err := be.BatchGenerateExamples(ctx, targetSegments)
	if err == nil {
		return results, nil
	}
	if !errors.Is(err, enrich.ErrBatchTruncated) {
		return nil, err
	}

	// 2分割リトライ
	mid := len(targetSegments) / 2
	if mid == 0 {
		return results, nil
	}

	r1, err1 := be.BatchGenerateExamples(ctx, targetSegments[:mid])
	r2, err2 := be.BatchGenerateExamples(ctx, targetSegments[mid:])

	var combined []enrich.BatchExampleResult
	if err1 == nil {
		combined = append(combined, r1...)
	}
	if err2 == nil {
		combined = append(combined, r2...)
	}

	if err1 != nil && err2 != nil {
		return combined, fmt.Errorf("both halves failed: %v; %v", err1, err2)
	}

	return combined, nil
}

// saveEntries はエントリをファイルに保存する共通ヘルパー。
func saveEntries(cmd *cobra.Command, entries []importEntry, specDir string, force, dryRun bool) error {
	var created, skipped, overwritten int

	for _, entry := range entries {
		s := entry.spec
		specPath := filepath.Join(specDir, s.ID+".yml")

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
			FilePath:    seg.Meta.FilePath,
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
