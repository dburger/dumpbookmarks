package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadTestBookmarks loads and unmarshals the test fixture from testdata/bookmarks.json.
func loadTestBookmarks(t *testing.T) *bookmark {
	t.Helper()
	data, err := os.ReadFile("testdata/bookmarks.json")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	var bf bookmarks
	if err := json.Unmarshal(data, &bf); err != nil {
		t.Fatalf("failed to unmarshal test fixture: %v", err)
	}

	bar, ok := bf.Roots["bookmark_bar"]
	if !ok {
		t.Fatalf("test fixture missing bookmark_bar")
	}

	return &bar
}

// captureStdout redirects os.Stdout during the execution of f and returns the captured output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = oldStdout
	return <-outChan
}

func TestFind(t *testing.T) {
	root := loadTestBookmarks(t)

	tests := []struct {
		name     string
		path     []string
		wantName string
		wantType string
		wantURL  string
		wantNil  bool
	}{
		{
			name:     "empty path returns root",
			path:     []string{},
			wantName: "Bookmarks bar",
			wantType: "folder",
		},
		{
			name:     "single level folder",
			path:     []string{"Recipes"},
			wantName: "Recipes",
			wantType: "folder",
		},
		{
			name:     "nested folder",
			path:     []string{"Recipes", "Desserts"},
			wantName: "Desserts",
			wantType: "folder",
		},
		{
			name:     "nested url leaf node",
			path:     []string{"Recipes", "Lasagna"},
			wantName: "Lasagna",
			wantType: "url",
			wantURL:  "https://example.com/lasagna",
		},
		{
			name:     "deeply nested url leaf node",
			path:     []string{"Recipes", "Desserts", "Tiramisu"},
			wantName: "Tiramisu",
			wantType: "url",
			wantURL:  "https://example.com/tiramisu",
		},
		{
			name:    "nonexistent root level",
			path:    []string{"Nonexistent"},
			wantNil: true,
		},
		{
			name:    "nonexistent child level",
			path:    []string{"Recipes", "Nonexistent"},
			wantNil: true,
		},
		{
			name:    "traversing past a leaf url node",
			path:    []string{"Recipes", "Lasagna", "InvalidSubnode"},
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := find(root, tc.path)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("find(%v) = %+v; want nil", tc.path, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("find(%v) = nil; want non-nil", tc.path)
			}
			if got.Name != tc.wantName {
				t.Errorf("Name = %q; want %q", got.Name, tc.wantName)
			}
			if got.Type != tc.wantType {
				t.Errorf("Type = %q; want %q", got.Type, tc.wantType)
			}
			if tc.wantURL != "" && got.URL != tc.wantURL {
				t.Errorf("URL = %q; want %q", got.URL, tc.wantURL)
			}
		})
	}
}

func TestDump(t *testing.T) {
	root := loadTestBookmarks(t)

	t.Run("descend true dumps all subfolder URLs", func(t *testing.T) {
		output := captureStdout(t, func() {
			dump(root, true)
		})
		lines := strings.Split(strings.TrimSpace(output), "\n")
		wantLines := []string{
			"https://example.com/top",
			"https://example.com/lasagna",
			"https://example.com/tiramisu",
		}
		if len(lines) != len(wantLines) {
			t.Fatalf("dump output line count = %d; want %d. Output:\n%s", len(lines), len(wantLines), output)
		}
		for i, want := range wantLines {
			if lines[i] != want {
				t.Errorf("line[%d] = %q; want %q", i, lines[i], want)
			}
		}
	})

	t.Run("descend false dumps only immediate child URLs", func(t *testing.T) {
		output := captureStdout(t, func() {
			dump(root, false)
		})
		lines := strings.Split(strings.TrimSpace(output), "\n")
		wantLines := []string{
			"https://example.com/top",
		}
		if len(lines) != len(wantLines) {
			t.Fatalf("dump output line count = %d; want %d. Output:\n%s", len(lines), len(wantLines), output)
		}
		if lines[0] != wantLines[0] {
			t.Errorf("line[0] = %q; want %q", lines[0], wantLines[0])
		}
	})

	t.Run("subtree with descend true", func(t *testing.T) {
		subtree := find(root, []string{"Recipes"})
		output := captureStdout(t, func() {
			dump(subtree, true)
		})
		lines := strings.Split(strings.TrimSpace(output), "\n")
		wantLines := []string{
			"https://example.com/lasagna",
			"https://example.com/tiramisu",
		}
		if len(lines) != len(wantLines) {
			t.Fatalf("dump output line count = %d; want %d. Output:\n%s", len(lines), len(wantLines), output)
		}
		for i, want := range wantLines {
			if lines[i] != want {
				t.Errorf("line[%d] = %q; want %q", i, lines[i], want)
			}
		}
	})
}

func TestDumpLeafURL(t *testing.T) {
	root := loadTestBookmarks(t)
	leaf := find(root, []string{"Recipes", "Lasagna"})
	if leaf == nil {
		t.Fatal("failed to find leaf node 'Recipes' -> 'Lasagna'")
	}

	output := captureStdout(t, func() {
		dump(leaf, true)
	})

	want := "https://example.com/lasagna\n"
	if output != want {
		t.Errorf("dump(leaf) output = %q; want %q", output, want)
	}
}

func TestBookmarksJSONUnmarshal(t *testing.T) {
	data, err := os.ReadFile("testdata/bookmarks.json")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	var bf bookmarks
	if err := json.Unmarshal(data, &bf); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	bar, ok := bf.Roots["bookmark_bar"]
	if !ok {
		t.Fatalf("missing 'bookmark_bar' in Roots")
	}

	if bar.Name != "Bookmarks bar" || bar.Type != "folder" {
		t.Errorf("root = {Name: %q, Type: %q}; want {'Bookmarks bar', 'folder'}", bar.Name, bar.Type)
	}

	if len(bar.Children) != 3 {
		t.Fatalf("bar.Children len = %d; want 3", len(bar.Children))
	}

	topLink := bar.Children[0]
	if topLink.Name != "Top Link" || topLink.Type != "url" || topLink.URL != "https://example.com/top" {
		t.Errorf("topLink = %+v; want Top Link url node", topLink)
	}
}

func TestDefaultBookmarksPath(t *testing.T) {
	path := defaultBookmarksPath()
	if path == "" {
		t.Fatal("defaultBookmarksPath() returned empty string")
	}

	base := filepath.Base(path)
	if base != "AccountBookmarks" && base != "Bookmarks" {
		t.Errorf("defaultBookmarksPath() filename = %q; want 'AccountBookmarks' or 'Bookmarks'", base)
	}
}
