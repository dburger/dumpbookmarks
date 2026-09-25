# Audit & Improvement Plan for dumpbookmarks

This document contains a comprehensive audit of [dumpbookmarks.go](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go) and [Makefile](file:///home/dburger/src/dumpbookmarks/Makefile), detailing functional bugs, portability issues, code quality concerns, and build/testing improvements, along with proposed fixes and completed tasks.

---

## Table of Contents
1. [Functional Bugs & Logic Flaws](#1-functional-bugs--logic-flaws)
   - [Bug 1: Leaf Bookmark URL Nodes Not Dumped](#bug-1-leaf-bookmark-url-nodes-not-dumped)
   - [Bug 2: Chrome Bookmarks in `other` and `synced` Roots Are Omitted](#bug-2-chrome-bookmarks-in-other-and-synced-roots-are-omitted)
   - [Bug 3: Silent Failure When `bookmark_bar` Is Absent](#bug-3-silent-failure-when-bookmark_bar-is-absent)
2. [Portability & File Resolution](#2-portability--file-resolution)
   - [Bug 4: Hardcoded Linux Path Breaks Cross-Platform Builds](#bug-4-hardcoded-linux-path-breaks-cross-platform-builds)
   - [Bug 5: Brittle File Selection (`AccountBookmarks` vs `Bookmarks`)](#bug-5-brittle-file-selection-accountbookmarks-vs-bookmarks)
3. [Idiomatic Go & Code Quality](#3-idiomatic-go--code-quality)
   - [Issue 8: Built-in `println` and Error Formatting in `bail`](#issue-8-built-in-println-and-error-formatting-in-bail)
   - [Issue 9: Missing Struct JSON Tags and Naming Convention](#issue-9-missing-struct-json-tags-and-naming-convention)
4. [Build System & Testing](#4-build-system--testing)
   - [Issue 10: Missing `.PHONY` and Invalid Globbing in Makefile](#issue-10-missing-phony-and-invalid-globbing-in-makefile)
   - [Issue 11: Missing Automated Tests](#issue-11-missing-automated-tests)
5. [Completed Tasks](#5-completed-tasks)
   - [Issue 6: Address of Loop Variable Copy in `find`](#issue-6-address-of-loop-variable-copy-in-find)
   - [Issue 7: Variable Identifier Shadowing (`filepath` and `bookmark`)](#issue-7-variable-identifier-shadowing-filepath-and-bookmark)

---

## 1. Functional Bugs & Logic Flaws

### Bug 1: Leaf Bookmark URL Nodes Not Dumped

* **File Location:** [`dump`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L71-L79)
* **Problem:** If a user specifies a path that resolves directly to a bookmark URL instead of a folder (e.g., `dumpbookmarks recipes lasagna` or `dumpbookmarks Apps Squoosh`), [`find`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L57-L67) returns the matching leaf node. However, [`dump`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L71-L79) only iterates over `bookmark.Children`. Because a leaf bookmark has no children, nothing is printed and the program exits with code 0.
* **Proposed Fix:** Check whether `bookmark.Type == "url"` at the beginning of [`dump`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L71-L79):
  ```go
  func dump(b *bookmark, descend bool) {
      if b.Type == "url" {
          fmt.Println(b.Url)
          return
      }
      for i := range b.Children {
          child := &b.Children[i]
          if child.Type == "url" {
              fmt.Println(child.Url)
          } else if descend {
              dump(child, descend)
          }
      }
  }
  ```

---

### Bug 2: Chrome Bookmarks in `other` and `synced` Roots Are Omitted

* **File Location:** [`main`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L117-L118)
* **Problem:** Google Chrome partitions bookmarks into three primary roots:
  - `bookmark_bar` ("Bookmarks bar")
  - `other` ("Other bookmarks")
  - `synced` ("Mobile bookmarks")
  
  The current code hardcodes `bookmarksFile.Roots["bookmark_bar"]`. Bookmarks saved under "Other bookmarks" or mobile bookmarks are invisible, and specifying folder paths located under those roots fails.
* **Proposed Fix:** Allow searching and dumping across all available roots when no specific path is given (or construct a virtual root containing all top-level roots):
  ```go
  virtualRoot := bookmark{
      Name: "Roots",
      Type: "folder",
  }
  for _, rootName := range []string{"bookmark_bar", "other", "synced"} {
      if r, ok := bookmarksFile.Roots[rootName]; ok {
          virtualRoot.Children = append(virtualRoot.Children, r)
      }
  }
  ```

---

### Bug 3: Silent Failure When `bookmark_bar` Is Absent

* **File Location:** [`main`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L117-L125)
* **Problem:** In Go, accessing a missing map key (`bookmarksFile.Roots["bookmark_bar"]`) yields the zero-value [`bookmark`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L34-L39). Taking its address (`&bookmarkBar`) produces a non-nil pointer. When no arguments are supplied, the `if bookmark == nil` check passes, [`dump`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L71-L79) iterates over empty children, and the command exits 0 without indicating that the root was missing.
* **Proposed Fix:** Check map existence:
  ```go
  bookmarkBar, ok := bookmarksFile.Roots["bookmark_bar"]
  if !ok {
      bail("No bookmark_bar found in bookmarks file", nil, 1)
  }
  ```

---

## 2. Portability & File Resolution

### Bug 4: Hardcoded Linux Path Breaks Cross-Platform Builds

* **File Location:** [`parseFlags`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L88)
* **Problem:** [Makefile](file:///home/dburger/src/dumpbookmarks/Makefile#L6-L7) provides a Windows build target (`bin/dumpbookmarks.exe`). However, the default file path is hardcoded as:
  ```go
  filepath.Join(homedir, ".config/google-chrome/Default/AccountBookmarks")
  ```
  On Windows and macOS, this Linux path does not exist.
* **Proposed Fix:** Determine default paths based on `runtime.GOOS`:
  - **Linux:** `~/.config/google-chrome/Default/`
  - **Darwin (macOS):** `~/Library/Application Support/Google/Chrome/Default/`
  - **Windows:** `%LOCALAPPDATA%\Google\Chrome\User Data\Default\` (via `os.Getenv("LOCALAPPDATA")`)

---

### Bug 5: Brittle File Selection (`AccountBookmarks` vs `Bookmarks`)

* **File Location:** [`parseFlags`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L88)
* **Problem:** Chrome only writes to `AccountBookmarks` when account-based bookmark storage is active. Standard installations and local profiles use `Bookmarks`. If `AccountBookmarks` is missing, the tool immediately exits with an error:
  `Error reading bookmarks file: open ...: no such file or directory`.
* **Proposed Fix:** Check candidate file names in order of preference (e.g. `AccountBookmarks`, then `Bookmarks`) and pick the first that exists.

---

## 3. Idiomatic Go & Code Quality

### Issue 8: Built-in `println` and Error Formatting in `bail`

* **File Location:** [`bail`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L47-L53)
* **Problem:** Built-in `println` is a Go bootstrapping primitive not intended for production application code. In addition, printing errors across two separate lines with trailing colons results in unformatted stderr output.
* **Proposed Fix:** Use standard library `fmt.Fprintf`:
  ```go
  func bail(msg string, err error, exitCode int) {
      if err != nil {
          fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
      } else {
          fmt.Fprintln(os.Stderr, msg)
      }
      os.Exit(exitCode)
  }
  ```

---

### Issue 9: Missing Struct JSON Tags and Naming Convention

* **File Location:** [`bookmark`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L34-L39), [`bookmarks`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L42-L44)
* **Problem:** Fields currently rely on case-insensitive unmarshaling. In Go, initialisms should be capitalized (`URL` instead of `Url`), and fields should be explicitly tagged with JSON keys.
* **Proposed Fix:**
  ```go
  type bookmark struct {
      Name     string     `json:"name"`
      Type     string     `json:"type"`
      URL      string     `json:"url"`
      Children []bookmark `json:"children"`
  }

  type bookmarks struct {
      Roots map[string]bookmark `json:"roots"`
  }
  ```

---

## 4. Build System & Testing

### Issue 10: Missing `.PHONY` and Invalid Globbing in Makefile

* **File Location:** [Makefile](file:///home/dburger/src/dumpbookmarks/Makefile#L1-L18)
* **Problem:**
  - Build targets (`build`, `buildl`, `buildw`, `runl`, `runw`, `clean`) lack `.PHONY` declarations.
  - `$(wildcard **/*.go)` does not recurse in standard GNU Make.
* **Proposed Fix:** Add `.PHONY` and refine source discovery:
  ```makefile
  .PHONY: all build buildl buildw runl runw clean test

  SRC = $(shell find . -name '*.go')

  bin/dumpbookmarks: $(SRC)
  	GOOS=linux GOARCH=amd64 go build -o bin/dumpbookmarks

  bin/dumpbookmarks.exe: $(SRC)
  	GOOS=windows GOARCH=amd64 go build -o bin/dumpbookmarks.exe

  buildl: bin/dumpbookmarks
  buildw: bin/dumpbookmarks.exe
  build: buildl buildw

  test:
  	go test -v ./...

  runl: buildl
  	./bin/dumpbookmarks

  clean:
  	rm -rf ./bin
  ```

---

### Issue 11: Missing Automated Tests

* **Problem:** The repository has no test files (`[no test files]`).
* **Proposed Fix:** Add `dumpbookmarks_test.go` covering:
  - Traversal with [`find`](file:///home/dburger/src/dumpbookmarks/dumpbookmarks.go#L57-L67) for single and multi-level paths
  - Dumping individual URL leaf nodes
  - Behavior when `descend` is `true` vs `false`
  - Handling of non-existent bookmark paths
  - JSON decoding of Chrome bookmark schema fixtures

---

## 5. Completed Tasks

### Issue 6: Address of Loop Variable Copy in `find`
- **Status:** Completed (commit `f2e6b6b`)
- **Resolution:** Replaced `for _, child := range bookmark.Children` with index-based iteration `for i := range bookmark.Children` passing `&bookmark.Children[i]`, avoiding local value copies and unnecessary heap escapes.

### Issue 7: Variable Identifier Shadowing (`filepath` and `bookmark`)
- **Status:** Completed (commits `b699efe` and `468bf73`)
- **Resolution:**
  - Renamed the CLI flag variable and struct field from `filepath` to `filename` to eliminate shadowing of the standard library package `path/filepath`.
  - Renamed the local variable in `main` from `bookmark` to `bm` to eliminate shadowing of the `bookmark` struct type.
