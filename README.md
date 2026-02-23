# family-tree-merger

A Go CLI tool that merges two family tree JSON files if multiple persons maintain a tree
in the same format and with shared IDs.
The tool integrates the other person's changes into the user's tree: adding new persons,
filling empty fields, and reporting conflicting values — either to a text file or via
an interactive TUI.

## Usage

```
family-tree-merger [flags] <mine.json> <theirs.json>

  -output,    -o  string   Output merged file (default "merged.json")
  -conflicts, -c  string   Conflicts report file (default "conflicts.txt")
  -interactive,-i          Resolve conflicts interactively via TUI
```

```bash
# Non-interactive
./family-tree-merger mine.json theirs.json
./family-tree-merger -o result.json -c report.txt mine.json theirs.json

# Interactive
./family-tree-merger -i mine.json theirs.json
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

## Build & test

```bash
go build -o family-tree-merger .
go test ./...
```

## License

This software is distributed under a GPL-3.0-or-later license.
Also see https://www.gnu.org/licenses/gpl-3.0.html
