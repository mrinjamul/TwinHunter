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

	if keep.Path != "/only.jpg" {
		t.Errorf("single: want keep=/only.jpg, got keep=%s", keep.Path)
	}
	if len(toRemove) != 0 {
		t.Errorf("single: want 0 toRemove, got %d", len(toRemove))
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
			{Path: "/long/path/first.jpg", ModTime: same},
			{Path: "/short.jpg", ModTime: same},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "oldest")

	if keep.Path != "/short.jpg" {
		t.Errorf("tie oldest: want shortest path /short.jpg, got %s", keep.Path)
	}
}

func TestApplyKeepStrategy_TieBreakNewest(t *testing.T) {
	same := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/long/path/first.jpg", ModTime: same},
			{Path: "/short.jpg", ModTime: same},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "newest")

	if keep.Path != "/short.jpg" {
		t.Errorf("tie newest: want shortest path /short.jpg, got %s", keep.Path)
	}
}

func TestApplyKeepStrategy_TieBreakShortest(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/equal.jpg", ModTime: base.Add(1 * time.Hour)},
			{Path: "/eq2.jpg", ModTime: base},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "shortest")

	if keep.Path != "/eq2.jpg" {
		t.Errorf("tie shortest: want oldest /eq2.jpg, got %s", keep.Path)
	}
}

func TestSortGroupsByPath(t *testing.T) {
	groups := []models.DuplicateGroup{
		{Files: []models.FileInfo{{Path: "/zebra/file.txt"}}},
		{Files: []models.FileInfo{{Path: "/alpha/file.txt"}}},
		{Files: []models.FileInfo{{Path: "/mango/file.txt"}}},
	}

	SortGroupsByPath(groups)

	if groups[0].Files[0].Path != "/alpha/file.txt" {
		t.Errorf("expected first group to be alpha, got %s", groups[0].Files[0].Path)
	}
	if groups[1].Files[0].Path != "/mango/file.txt" {
		t.Errorf("expected second group to be mango, got %s", groups[1].Files[0].Path)
	}
	if groups[2].Files[0].Path != "/zebra/file.txt" {
		t.Errorf("expected third group to be zebra, got %s", groups[2].Files[0].Path)
	}
}

func TestSortGroupsByPath_EmptyFiles(t *testing.T) {
	groups := []models.DuplicateGroup{
		{Files: []models.FileInfo{}},
		{Files: []models.FileInfo{{Path: "/alpha/file.txt"}}},
	}

	SortGroupsByPath(groups)

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestSortGroupsByPath_SingleGroup(t *testing.T) {
	groups := []models.DuplicateGroup{
		{Files: []models.FileInfo{{Path: "/only/file.txt"}}},
	}

	SortGroupsByPath(groups)

	if groups[0].Files[0].Path != "/only/file.txt" {
		t.Errorf("expected path to be preserved, got %s", groups[0].Files[0].Path)
	}
}

func TestGroupByHash_EmptyHashFiltering(t *testing.T) {
	annotated := []AnnotatedFile{
		{File: models.FileInfo{Path: "/a.txt"}, Hash: "abc123"},
		{File: models.FileInfo{Path: "/b.txt"}, Hash: ""},
		{File: models.FileInfo{Path: "/c.txt"}, Hash: "abc123"},
		{File: models.FileInfo{Path: "/d.txt"}, Hash: ""},
	}

	groups := GroupByHash(annotated)

	if len(groups) != 1 {
		t.Errorf("expected 1 group (empty hashes filtered), got %d", len(groups))
	}

	if len(groups["abc123"]) != 2 {
		t.Errorf("expected 2 files in abc123 group, got %d", len(groups["abc123"]))
	}
}

func TestFindDuplicates_AllUniqueSizes(t *testing.T) {
	files := []models.FileInfo{
		{Path: "/a.txt", Size: 100},
		{Path: "/b.txt", Size: 200},
		{Path: "/c.txt", Size: 300},
	}

	groups := FindDuplicates(files, 2, nil)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for all unique sizes, got %d", len(groups))
	}
}

func TestFindDuplicates_NoCandidates(t *testing.T) {
	files := []models.FileInfo{
		{Path: "/a.txt", Size: 100},
	}

	groups := FindDuplicates(files, 2, nil)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for single file, got %d", len(groups))
	}
}

func TestFindDuplicates_EmptyFiles(t *testing.T) {
	groups := FindDuplicates([]models.FileInfo{}, 2, nil)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty files, got %d", len(groups))
	}
}

func TestFindDuplicates_NilFiles(t *testing.T) {
	groups := FindDuplicates(nil, 2, nil)

	if groups != nil {
		t.Errorf("expected nil for nil input, got %v", groups)
	}
}
