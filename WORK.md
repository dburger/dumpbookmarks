# Audit & Improvement Plan for dumpbookmarks

This document contains a comprehensive audit of [main.go](file:///home/dburger/src/dumpbookmarks/main.go) and [Makefile](file:///home/dburger/src/dumpbookmarks/Makefile), detailing functional bugs, portability issues, code quality concerns, and build/testing improvements, along with completed tasks.

---

## Completed Tasks

All identified issues from the audit have been successfully resolved and verified with tests:

- [Issue 6: Address of Loop Variable Copy in `find`](#issue-6-address-of-loop-variable-copy-in-find)
- [Issue 7: Variable Identifier Shadowing (`filepath` and `bookmark`)](#issue-7-variable-identifier-shadowing-filepath-and-bookmark)
- [Issue 8: Built-in `println` and Error Formatting in `bail`](#issue-8-built-in-println-and-error-formatting-in-bail)
- [Issue 9: Missing Struct JSON Tags and Naming Convention](#issue-9-missing-struct-json-tags-and-naming-convention)
- [Issue 10: Missing `.PHONY` and Invalid Globbing in Makefile](#issue-10-missing-phony-and-invalid-globbing-in-makefile)
- [Source File Renaming (`main.go`)](#source-file-renaming-maingo)
- [Bug 4: Hardcoded Linux Path Breaks Cross-Platform Builds](#bug-4-hardcoded-linux-path-breaks-cross-platform-builds)
- [Bug 5: Brittle File Selection (`AccountBookmarks` vs `Bookmarks`)](#bug-5-brittle-file-selection-accountbookmarks-vs-bookmarks)
- [Issue 11: Missing Automated Tests](#issue-11-missing-automated-tests)
- [Bug 1: Leaf Bookmark URL Nodes Not Dumped](#bug-1-leaf-bookmark-url-nodes-not-dumped)
- [Bug 3: Silent Failure When `bookmark_bar` Is Absent](#bug-3-silent-failure-when-bookmark_bar-is-absent)
- [Bug 2: Chrome Bookmarks in `other` and `synced` Roots Are Omitted](#bug-2-chrome-bookmarks-in-other-and-synced-roots-are-omitted)

---

### Issue 6: Address of Loop Variable Copy in `find`
- **Status:** Completed (commit `f2e6b6b`)
- **Resolution:** Replaced `for _, child := range bookmark.Children` with index-based iteration `for i := range bookmark.Children` passing `&bookmark.Children[i]`, avoiding local value copies and unnecessary heap escapes.

### Issue 7: Variable Identifier Shadowing (`filepath` and `bookmark`)
- **Status:** Completed (commits `b699efe` and `468bf73`)
- **Resolution:**
  - Renamed the CLI flag variable and struct field from `filepath` to `filename` to eliminate shadowing of the standard library package `path/filepath`.
  - Renamed the local variable in `main` from `bookmark` to `bm` to eliminate shadowing of the `bookmark` struct type.

### Issue 8: Built-in `println` and Error Formatting in `bail`
- **Status:** Completed (commit `c9bd10a`)
- **Resolution:** Replaced built-in `println` with `fmt.Fprintf(os.Stderr, ...)` for error messages and cleaned up trailing colons at all call sites so errors output cleanly on a single line.

### Issue 9: Missing Struct JSON Tags and Naming Convention
- **Status:** Completed (commit `074cf7e`)
- **Resolution:** Added explicit JSON field tags (`json:"name"`, `json:"type"`, `json:"url"`, `json:"children"`, `json:"roots"`) to `bookmark` and `bookmarks` structs, and renamed `Url` to `URL` to follow Go conventions for initialisms.

### Issue 10: Missing `.PHONY` and Invalid Globbing in Makefile
- **Status:** Completed (commits `364a8f9` and `8e6e7ba`)
- **Resolution:**
  - Added `.PHONY: build buildl buildw runl runw clean` to ensure phony targets execute reliably regardless of filesystem state.
  - Simplified `SRC` from `$(wildcard *.go) $(wildcard **/*.go)` to `$(wildcard *.go)` since GNU Make's `wildcard` does not do recursive `**` expansion, and the project sources are in the root directory.

### Source File Renaming (`main.go`)
- **Status:** Completed (commit `d972c6f`)
- **Resolution:** Renamed `dumpbookmarks.go` to `main.go` per Go conventions for standalone single-package executable applications, and updated documentation examples to use `go run .`.

### Bug 4: Hardcoded Linux Path Breaks Cross-Platform Builds
- **Status:** Completed (commit `d5e5814`)
- **Resolution:** Introduced `defaultBookmarksPath()` which inspects `runtime.GOOS` to construct the correct Chrome profile directory for Linux (`~/.config/google-chrome/Default`), macOS (`~/Library/Application Support/Google/Chrome/Default`), and Windows (`%LOCALAPPDATA%\Google\Chrome\User Data\Default`).

### Bug 5: Brittle File Selection (`AccountBookmarks` vs `Bookmarks`)
- **Status:** Completed (commit `d5e5814`)
- **Resolution:** In `defaultBookmarksPath()`, sequentially check candidate filenames (`AccountBookmarks`, then `Bookmarks`), returning the first that exists on disk and falling back cleanly if neither is present.

### Issue 11: Missing Automated Tests
- **Status:** Completed (commits `3f1cbd1` and `2cc61a0`)
- **Resolution:** Added automated test suite in `main_test.go` and test fixture in `testdata/bookmarks.json` covering path resolution with `find()`, subtree vs leaf URL dumping with `dump()`, `-descend` behavior, JSON unmarshaling, and default path resolution.

### Bug 1: Leaf Bookmark URL Nodes Not Dumped
- **Status:** Completed (commit `c884d36`)
- **Resolution:** Added an initial check in `dump()` for `bookmark.Type == "url"`. If true, it prints `bookmark.URL` and returns immediately. Verified passing with `TestDumpLeafURL`.

### Bug 3: Silent Failure When `bookmark_bar` Is Absent
- **Status:** Completed (commit `21a92d1`)
- **Resolution:** Checked presence of root key in `bookmarksFile.Roots` via comma-ok syntax; call `bail(...)` if absent to avoid silent exit with empty output.

### Bug 2: Chrome Bookmarks in `other` and `synced` Roots Are Omitted
- **Status:** Completed (commit `0638da6`)
- **Resolution:** Added `-root` flag (defaulting to `"bookmark_bar"`) to allow querying and dumping from other Chrome bookmark root sections like `"other"` ("Other bookmarks") and `"synced"` ("Mobile bookmarks"), completely avoiding namespace collisions.
