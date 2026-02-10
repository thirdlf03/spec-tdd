package enrich

import (
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/kire"
)

func TestBatchClassifyPrompt(t *testing.T) {
	if batchClassifyPrompt == "" {
		t.Fatal("batchClassifyPrompt should not be empty")
	}
	if !strings.Contains(batchClassifyPrompt, "%s") {
		t.Error("batchClassifyPrompt should contain a format placeholder")
	}
	if !strings.Contains(batchClassifyPrompt, "functional_requirement") {
		t.Error("batchClassifyPrompt should mention functional_requirement")
	}
	if !strings.Contains(batchClassifyPrompt, "segment_id") {
		t.Error("batchClassifyPrompt should mention segment_id")
	}
}

func TestBatchExamplesPrompt(t *testing.T) {
	if batchExamplesPrompt == "" {
		t.Fatal("batchExamplesPrompt should not be empty")
	}
	if !strings.Contains(batchExamplesPrompt, "%s") {
		t.Error("batchExamplesPrompt should contain a format placeholder")
	}
	if !strings.Contains(batchExamplesPrompt, "Given") {
		t.Error("batchExamplesPrompt should mention Given/When/Then")
	}
}

func TestBatchClassifySchema(t *testing.T) {
	if batchClassifySchema == nil {
		t.Fatal("batchClassifySchema should not be nil")
	}
	if batchClassifySchema.Items == nil {
		t.Fatal("batchClassifySchema should have Items (array type)")
	}

	item := batchClassifySchema.Items
	for _, key := range []string{"segment_id", "category", "title", "req_id"} {
		if _, ok := item.Properties[key]; !ok {
			t.Errorf("batchClassifySchema item should have %q property", key)
		}
	}
}

func TestBatchExamplesSchema(t *testing.T) {
	if batchExamplesSchema == nil {
		t.Fatal("batchExamplesSchema should not be nil")
	}
	if batchExamplesSchema.Items == nil {
		t.Fatal("batchExamplesSchema should have Items (array type)")
	}

	item := batchExamplesSchema.Items
	if _, ok := item.Properties["segment_id"]; !ok {
		t.Error("batchExamplesSchema item should have 'segment_id' property")
	}
	if _, ok := item.Properties["examples"]; !ok {
		t.Error("batchExamplesSchema item should have 'examples' property")
	}
}

func TestFormatSegmentsForClassify(t *testing.T) {
	segments := []*kire.Segment{
		{Meta: kire.SegmentMeta{SegmentID: "seg-0001"}, Content: "# Overview\nContent A"},
		{Meta: kire.SegmentMeta{SegmentID: "seg-0002"}, Content: "# Login\nContent B"},
	}

	result := formatSegmentsForClassify(segments)

	if !strings.Contains(result, "--- segment_id: seg-0001 ---") {
		t.Error("should contain seg-0001 delimiter")
	}
	if !strings.Contains(result, "--- segment_id: seg-0002 ---") {
		t.Error("should contain seg-0002 delimiter")
	}
	if !strings.Contains(result, "Content A") {
		t.Error("should contain segment content A")
	}
	if !strings.Contains(result, "Content B") {
		t.Error("should contain segment content B")
	}
}

func TestFormatSegmentsForClassify_empty(t *testing.T) {
	result := formatSegmentsForClassify(nil)
	if result != "" {
		t.Errorf("expected empty string for nil segments, got %q", result)
	}
}

func TestFormatSegmentsForExamples(t *testing.T) {
	segments := []*kire.Segment{
		{Meta: kire.SegmentMeta{SegmentID: "seg-0001", HeadingPath: []string{"Doc", "Login"}}, Content: "Content A"},
		{Meta: kire.SegmentMeta{SegmentID: "seg-0002", HeadingPath: []string{"Doc", "Register"}}, Content: "Content B"},
	}
	titles := map[string]string{
		"seg-0001": "ログイン",
		"seg-0002": "登録",
	}

	result := formatSegmentsForExamples(segments, titles)

	if !strings.Contains(result, "--- segment_id: seg-0001 | title: ログイン ---") {
		t.Error("should contain seg-0001 with title")
	}
	if !strings.Contains(result, "--- segment_id: seg-0002 | title: 登録 ---") {
		t.Error("should contain seg-0002 with title")
	}
}
