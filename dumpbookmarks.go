package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type params struct {
	descend  bool
	filepath string
}

type Bookmarks struct {
	Name     string
	Type     string
	Url      string
	Children []Bookmarks
}

type BookmarksFile struct {
	Roots map[string]Bookmarks
}

func bail(msg string, err error, exitCode int) {
	println(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(exitCode)
}

func find(bookmarks Bookmarks, args []string) *Bookmarks {
	if len(args) == 0 {
		return &bookmarks
	}
	for _, child := range bookmarks.Children {
		if child.Name == args[0] {
			return find(child, args[1:])
		}
	}
	return nil
}

func dump(bookmarks *Bookmarks, descend bool) {
	for _, child := range bookmarks.Children {
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
	file, err := os.Open(params.filepath)

	if err != nil {
		bail("Error opening bookmarks file:", err, 1)
	}

	defer file.Close()

	bytes, err := ioutil.ReadAll(file)

	if err != nil {
		bail("Error reading bookmarks file:", err, 1)
	}

	var bookmarksFile BookmarksFile
	err = json.Unmarshal([]byte(bytes), &bookmarksFile)

	if err != nil {
		bail("Error unmarshalling bookmarks file, has the schema changed?:", err, 1)
	}

	bookmarkBar := bookmarksFile.Roots["bookmark_bar"]

	bookmarks := &bookmarkBar
	args := flag.Args()
	if len(args) > 0 {
		// If they specified a subtree, start there.
		bookmarks = find(bookmarkBar, args)
	}

	if bookmarks == nil {
		bail("Requested bookmarks not found.", nil, 1)
	} else {
		dump(bookmarks, params.descend)
	}
}
