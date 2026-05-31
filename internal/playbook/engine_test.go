package playbook

import (
	"strings"
	"testing"
)

func TestGenerate_ReturnsExpectedSectionCount(t *testing.T) {
	eng := NewEngine()
	pb, err := eng.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	if len(pb.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(pb.Sections))
	}
}

func TestGenerate_ContentIsNonEmpty(t *testing.T) {
	eng := NewEngine()
	pb, err := eng.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	if pb.Content == "" {
		t.Fatal("expected non-empty content")
	}
}

func TestGenerate_ExpectedSectionTitles(t *testing.T) {
	eng := NewEngine()
	pb, err := eng.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	expected := []string{
		"Available Strategies",
		"Best Practices by Vendor Category",
		"Common Tactics",
		"Price Benchmarks",
		"Vendor-Specific Tips",
	}

	for i, title := range expected {
		if pb.Sections[i].Title != title {
			t.Errorf("section %d: expected title %q, got %q", i, title, pb.Sections[i].Title)
		}
	}
}

func TestGenerate_ItemsPerSection(t *testing.T) {
	eng := NewEngine()
	pb, err := eng.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	for _, sec := range pb.Sections {
		if len(sec.Items) == 0 {
			t.Errorf("section %q has no items", sec.Title)
		}
	}
}

func TestGenerate_ContentContainsSectionTitles(t *testing.T) {
	eng := NewEngine()
	pb, err := eng.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	titles := []string{
		"Available Strategies",
		"Best Practices by Vendor Category",
		"Common Tactics",
		"Price Benchmarks",
		"Vendor-Specific Tips",
	}

	for _, title := range titles {
		if !strings.Contains(pb.Content, title) {
			t.Errorf("expected content to contain section title %q", title)
		}
	}
}

func TestGenerate_FirstSectionHasFiveStrategies(t *testing.T) {
	eng := NewEngine()
	pb, err := eng.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	if len(pb.Sections[0].Items) != 5 {
		t.Errorf("section %q: expected 5 items, got %d", pb.Sections[0].Title, len(pb.Sections[0].Items))
	}
}
