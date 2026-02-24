# graft

A Go CLI tool that merges two family tree JSON files if multiple persons maintain a tree
in the same format and with shared IDs.
The tool integrates the other person's changes into the user's tree: adding new persons,
filling empty fields, and reporting conflicting values — either to a text file or via
an interactive TUI.

> **Why "graft"?** In botany, grafting joins two plants into a single one by uniting
> their tissues. That's exactly what this tool does: it grafts two family trees together
> into one.

## Usage

```
graft [flags] <mine.json> <theirs.json>

  -output,    -o  string   Output merged file (default "merged.json")
  -conflicts, -c  string   Conflicts report file (default "conflicts.txt")
  -interactive,-i          Resolve conflicts interactively via TUI
```

```bash
# Non-interactive
graft mine.json theirs.json
graft -o result.json -c report.txt mine.json theirs.json

# Interactive
graft -i mine.json theirs.json
```

## Merge behaviour

| Case | Action |
|------|--------|
| ID in theirs not in mine | Append new person to merged output |
| Field: mine empty, theirs non-empty | Fill from theirs (no conflict) |
| Field: mine non-empty, theirs empty | Keep mine (no conflict) |
| Field: both equal | No conflict |
| Field: both non-empty and different | Conflict — keep mine, log |
| `avatar_url`: mine empty, theirs non-empty | Always conflict — never silently filled |
| `children`/`spouses` arrays | Union — add missing IDs, no conflicts |

## Install

### Using Homebrew
```shell
brew tap 0xbs/tap
brew install graft
```

### From Source
```shell
go install github.com/0xbs/graft@latest
```

### Download Binary
Check out the [release page](https://github.com/0xbs/graft/releases) and download the latest release.

## Build

To build, use standard Golang commands like `go build`.

## License

This software is distributed under a GPL-3.0-or-later license.
Also see https://www.gnu.org/licenses/gpl-3.0.html
