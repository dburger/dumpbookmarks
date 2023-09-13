/*
Program to dump chrome bookmarks to stdout. Without arguments:

$ go run dumpbookmarks.go

and all bookmarks will be dumped. To specify only a certain subtree
of bookmarks, provide the folder names down to that path, for example:

$ go run dumpbookmarks.go recipes italian

This will dump the bookmarks starting at the recipes -> italian folder.
To only dump bookmarks at the specified level, without descending into
subfolders, pass the descend flag.

$ go run dumpbookmarks.go -descend=false
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type params struct {
	descend  bool
	filepath string
}

type Bookmark struct {
	Name     string
	Type     string
	Url      string
	Children []Bookmark
}

type Bookmarks struct {
	Roots map[string]Bookmark
}

func bail(msg string, err error, exitCode int) {
	println(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(exitCode)
}

func find(bookmark *Bookmark, args []string) *Bookmark {
	if len(args) == 0 {
		return bookmark
	}
	for _, child := range bookmark.Children {
		if child.Name == args[0] {
			return find(&child, args[1:])
		}
	}
	return nil
}

func dump(bookmark *Bookmark, descend bool) {
	for _, child := range bookmark.Children {
		if child.Type == "url" {
			fmt.Println(child.Url)
		} else if descend {
			dump(&child, descend)
		}
	}
}

func parseFlags() params {
	homedir, err := os.UserHomeDir()
	if err != nil {
		bail("Unable to determine user's home directory:", err, 1)
	}

	defaultpath := filepath.Join(homedir, ".config/google-chrome/Default/Bookmarks")

	descend := flag.Bool("descend", true, "descend to subfolders")
	filepath := flag.String("filename", defaultpath, "name of chrome bookmarks file to process")

	flag.Parse()

	return params{
		descend:  *descend,
		filepath: *filepath,
	}
}

func main() {
	params := parseFlags()

	bytes, err := os.ReadFile(params.filepath)
	if err != nil {
		bail("Error reading bookmarks file:", err, 1)
	}

	var bookmarksFile Bookmarks
	err = json.Unmarshal(bytes, &bookmarksFile)

	if err != nil {
		bail("Error unmarshalling bookmarks file, has the schema changed?:", err, 1)
	}

	bookmarkBar := bookmarksFile.Roots["bookmark_bar"]
	bookmark := &bookmarkBar

	args := flag.Args()
	if len(args) > 0 {
		// If they specified a subtree, start there.
		bookmark = find(bookmark, args)
	}

	if bookmark == nil {
		bail("Requested bookmarks not found.", nil, 1)
	} else {
		dump(bookmark, params.descend)
	}
}
