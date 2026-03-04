# graft

A Go CLI tool that merges two family tree JSON files as used by 
[donatso/family-chart](https://github.com/donatso/family-chart).
Useful if multiple persons maintain a tree in the same format and with shared IDs.
The tool integrates the other person's changes into the user's tree: adding new persons,
filling empty fields, and reporting conflicting values — either to a text file or via
an interactive TUI.

> **Why "graft"?** In botany, grafting joins two plants into a single one by uniting
> their tissues. That's exactly what this tool does: it grafts two family trees together
> into one.

## Usage

```
graft [flags] <mine.json> <theirs.json>   merge two files
graft -validate <file.json>               validate a file

  -output,         -o   string   Output merged file (default "merged.json")
  -conflicts,      -c   string   Conflicts report file (default "conflicts.txt")
  -interactive,    -i            Resolve conflicts interactively via TUI
  -always-conflict,-ac  string   Comma-separated data fields to always treat as
                                 conflicts, even when mine is empty
                                 (default "avatar_url,avatar")
  -validate,       -v            Validate a file for errors and warnings
```

```bash
# Non-interactive
graft mine.json theirs.json
graft -o result.json -c report.txt mine.json theirs.json

# Interactive
graft -i mine.json theirs.json

# Validate
graft -validate merged.json
```

## Validation

`graft -validate <file.json>` checks a family tree JSON file for data quality issues and exits with code 1 if any errors are found.

**Errors** (structural problems that should be fixed):

| Check | Description |
|-------|-------------|
| Missing/invalid gender | Every person must have `gender` set to `M` or `F` |
| Isolated person | Every person must have at least one relation |
| Missing reciprocal relations | If A lists B as father/mother, B must list A as a child; spouses must reference each other |
| Non-existent relation targets | Referenced IDs must exist in the file |
| Invalid calendar dates | `*_date`/`birthday` fields matching `yyyy-mm-dd` must be real calendar dates (e.g. `2024-02-30` is rejected) |
| Duplicate JSON keys | The same field name must not appear more than once in a `data` object |
| Duplicate IDs | Each person ID must be unique |

**Warnings** (data quality hints):

| Check | Description |
|-------|-------------|
| Non-standard date format | `*_date`/`birthday` fields with values that are neither `yyyy` nor `yyyy-mm-dd` |
| Rare fields | Fields used by exactly one person in a dataset of ≥5 (possible typo) |
| Identical persons | Two persons sharing at least one relation and having exactly matching data fields but different IDs |

## Merge behaviour

| Case | Action |
|------|--------|
| ID in theirs not in mine | Append new person to merged output |
| Field: mine empty, theirs non-empty | Fill from theirs (no conflict) |
| Field: mine non-empty, theirs empty | Keep mine (no conflict) |
| Field: both equal | No conflict |
| Field: both non-empty and different | Conflict — keep mine, log |
| `avatar_url`/`avatar`: mine empty, theirs non-empty | Always conflict — never silently filled (configurable via `-ac`) |
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
