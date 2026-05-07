package core

import (
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/models"
)

func TestApplyKeepStrategy_Oldest(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/a/newer.jpg", ModTime: base.Add(2 * time.Hour)},
			{Path: "/b/oldest.jpg", ModTime: base},
			{Path: "/c/middle.jpg", ModTime: base.Add(1 * time.Hour)},
		},
	}

	keep, toRemove := ApplyKeepStrategy(group, "oldest")

	if keep.Path != "/b/oldest.jpg" {
		t.Errorf("oldest: want /b/oldest.jpg, got %s", keep.Path)
	}
	if len(toRemove) != 2 {
		t.Errorf("oldest: want 2 to remove, got %d", len(toRemove))
	}
}

func TestApplyKeepStrategy_Newest(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/a/old.jpg", ModTime: base},
			{Path: "/b/newest.jpg", ModTime: base.Add(2 * time.Hour)},
			{Path: "/c/middle.jpg", ModTime: base.Add(1 * time.Hour)},
		},
	}

	keep, toRemove := ApplyKeepStrategy(group, "newest")

	if keep.Path != "/b/newest.jpg" {
		t.Errorf("newest: want /b/newest.jpg, got %s", keep.Path)
	}
	if len(toRemove) != 2 {
		t.Errorf("newest: want 2 to remove, got %d", len(toRemove))
	}
}

func TestApplyKeepStrategy_Shortest(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/very/long/path/to/file.jpg"},
			{Path: "/short.jpg"},
			{Path: "/medium/path.jpg"},
		},
	}

	keep, toRemove := ApplyKeepStrategy(group, "shortest")

	if keep.Path != "/short.jpg" {
		t.Errorf("shortest: want /short.jpg, got %s", keep.Path)
	}
	if len(toRemove) != 2 {
		t.Errorf("shortest: want 2 to remove, got %d", len(toRemove))
	}
}

func TestApplyKeepStrategy_UnknownDefaultsToFirst(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/first.jpg"},
			{Path: "/second.jpg"},
		},
	}

	keep, toRemove := ApplyKeepStrategy(group, "unknown")

	if keep.Path != "/first.jpg" {
		t.Errorf("unknown: want /first.jpg, got %s", keep.Path)
	}
	if len(toRemove) != 1 {
		t.Errorf("unknown: want 1 to remove, got %d", len(toRemove))
	}
}

func TestApplyKeepStrategy_EmptyGroup(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{},
	}

	keep, toRemove := ApplyKeepStrategy(group, "oldest")

	if keep.Path != "" {
		t.Errorf("empty: want empty keep, got %s", keep.Path)
	}
	if len(toRemove) != 0 {
		t.Errorf("empty: want 0 to remove, got %d", len(toRemove))
	}
}

func TestApplyKeepStrategy_SingleFile(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/only.jpg"},
		},
	}

	keep, toRemove := ApplyKeepStrategy(group, "oldest")

	if keep.Path != "" {
		t.Errorf("single: want empty keep, got %s", keep.Path)
	}
	if len(toRemove) != 1 || toRemove[0].Path != "/only.jpg" {
		t.Errorf("single: want /only.jpg in toRemove, got %v", toRemove)
	}
}

func TestApplyKeepStrategy_Immutability(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	original := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/a.jpg", ModTime: base.Add(1 * time.Hour)},
			{Path: "/b.jpg", ModTime: base},
		},
	}

	ApplyKeepStrategy(original, "newest")

	if original.Files[0].Path != "/a.jpg" || original.Files[1].Path != "/b.jpg" {
		t.Error("ApplyKeepStrategy mutated original group")
	}
}

func TestApplyKeepStrategy_TieBreakOldest(t *testing.T) {
	same := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/first.jpg", ModTime: same},
			{Path: "/second.jpg", ModTime: same},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "oldest")

	if keep.Path != "/first.jpg" {
		t.Errorf("tie oldest: want /first.jpg, got %s", keep.Path)
	}
}

func TestApplyKeepStrategy_TieBreakNewest(t *testing.T) {
	same := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/first.jpg", ModTime: same},
			{Path: "/second.jpg", ModTime: same},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "newest")

	if keep.Path != "/first.jpg" {
		t.Errorf("tie newest: want /first.jpg, got %s", keep.Path)
	}
}

func TestApplyKeepStrategy_TieBreakShortest(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/ab.jpg"},
			{Path: "/cd.jpg"},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "shortest")

	if keep.Path != "/ab.jpg" {
		t.Errorf("tie shortest: want /ab.jpg, got %s", keep.Path)
	}
}
