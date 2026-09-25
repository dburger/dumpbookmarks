/*
Program to dump chrome bookmarks to stdout. Without arguments:

$ go run .

and all bookmarks will be dumped. To specify only a certain subtree
of bookmarks, provide the folder names down to that path, for example:

$ go run . recipes italian

This will dump the bookmarks starting at the recipes -> italian folder.
To only dump bookmarks at the specified level, without descending into
subfolders, pass the descend flag.

$ go run . -descend=false
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// params holds the parsed command line parameters.
type params struct {
	descend      bool
	filename     string
	root         string
	bookmarkPath []string
}

// bookmark is a chrome bookmark or folder with an array of child bookmark.
type bookmark struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	URL      string     `json:"url"`
	Children []bookmark `json:"children"`
}

// bookmarks holds the entire bookmarks data structure.
type bookmarks struct {
	Roots map[string]bookmark `json:"roots"`
}

// bail is used to print an error to stderr and exit the program.
func bail(msg string, err error, exitCode int) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(exitCode)
}

// find attempts to find the bookmark starting point indicated in bookmarkPath.
// Each element of bookmarkPath walks the bookmarks tree down to the intended node.
func find(bookmark *bookmark, bookmarkPath []string) *bookmark {
	if len(bookmarkPath) == 0 {
		return bookmark
	}
	for i := range bookmark.Children {
		if bookmark.Children[i].Name == bookmarkPath[0] {
			return find(&bookmark.Children[i], bookmarkPath[1:])
		}
	}
	return nil
}

// dump dumps bookmarks to stdout. descend determines whether the code
// recurses into child nodes.
func dump(bookmark *bookmark, descend bool) {
	if bookmark.Type == "url" {
		fmt.Println(bookmark.URL)
		return
	}
	for _, child := range bookmark.Children {
		if child.Type == "url" {
			fmt.Println(child.URL)
		} else if descend {
			dump(&child, descend)
		}
	}
}

// defaultBookmarksPath returns the first existing Chrome bookmarks file
// from candidate filenames in the user's default Chrome profile across supported OSes.
func defaultBookmarksPath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		bail("Unable to determine user's home directory", err, 1)
	}

	var baseDir string
	switch runtime.GOOS {
	case "darwin":
		baseDir = filepath.Join(homedir, "Library", "Application Support", "Google", "Chrome", "Default")
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(homedir, "AppData", "Local")
		}
		baseDir = filepath.Join(localAppData, "Google", "Chrome", "User Data", "Default")
	default: // "linux" and other unix-like systems
		baseDir = filepath.Join(homedir, ".config", "google-chrome", "Default")
	}

	// Prefer AccountBookmarks if present, otherwise fall back to standard Bookmarks.
	candidates := []string{"AccountBookmarks", "Bookmarks"}
	for _, candidate := range candidates {
		path := filepath.Join(baseDir, candidate)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return filepath.Join(baseDir, "AccountBookmarks")
}

// parseFlags parses command line arguments and returns params.
func parseFlags() params {
	defaultPath := defaultBookmarksPath()

	descend := flag.Bool("descend", true, "descend to subfolders")
	filename := flag.String("filename", defaultPath, "name of chrome bookmarks file to process")
	root := flag.String("root", "bookmark_bar", "root bookmark folder (e.g. bookmark_bar, other, synced)")

	flag.Parse()

	return params{
		descend:      *descend,
		filename:     *filename,
		root:         *root,
		bookmarkPath: flag.Args(),
	}
}

func main() {
	params := parseFlags()

	bytes, err := os.ReadFile(params.filename)
	if err != nil {
		bail("Error reading bookmarks file", err, 1)
	}

	var bookmarksFile bookmarks
	err = json.Unmarshal(bytes, &bookmarksFile)

	if err != nil {
		bail("Error unmarshalling bookmarks file, has the schema changed?", err, 1)
	}

	rootBookmark, ok := bookmarksFile.Roots[params.root]
	if !ok {
		bail(fmt.Sprintf("No %s found in bookmarks file", params.root), nil, 1)
	}
	bm := &rootBookmark

	if len(params.bookmarkPath) > 0 {
		// If they specified a subtree, start there.
		bm = find(bm, params.bookmarkPath)
	}

	if bm == nil {
		bail("Requested bookmarks not found.", nil, 1)
	} else {
		dump(bm, params.descend)
	}
}
