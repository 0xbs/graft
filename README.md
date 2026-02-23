# family-tree-merger — Project Documentation

## Context
A Go CLI tool that merges two family tree JSON files. The user and their
brother-in-law each maintain a tree in the same format. The tool integrates
the brother-in-law's changes into the user's tree: adding new persons, filling
empty fields, and reporting conflicting values — either to a text file or via
an interactive TUI.

## Data Format (`example.json`)
JSON array of persons, each with:
- `id`: UUID (used for matching)
- `data`: 17 string fields (gender, names, dates, places, note, avatar_url)
- `rels`: `father`/`mother` (string IDs), `children`/`spouses` (string ID arrays)

## File Structure

| File | Purpose |
|------|---------|
| `types.go` | `Person`, `PersonData`, `PersonRels`, `Conflict` structs |
| `merge.go` | Merge algorithm, `alwaysConflict` map |
| `main.go` | CLI entry point, file I/O, conflict report writer, `fullName()` |
| `tui.go` | Bubble Tea interactive conflict resolution TUI |
| `merge_test.go` | Tests for all merge rules |
| `testdata/mine.json` | Test fixture — base tree |
| `testdata/theirs.json` | Test fixture — incoming tree |

## Merge Rules

| Case | Action |
|------|--------|
| ID in theirs not in mine | Append new person to merged output |
| `data`/`rels` field: mine empty, theirs non-empty | Fill from theirs (no conflict) |
| `data`/`rels` field: mine non-empty, theirs empty | Keep mine (no conflict) |
| `data`/`rels` field: both equal | No conflict |
| `data`/`rels` field: both non-empty and different | Conflict — keep mine, log |
| `avatar_url`: mine empty, theirs non-empty | **Always conflict** — never silently filled (file may not exist locally) |
| `children`/`spouses` arrays | Union — add missing IDs from theirs, no conflicts |

The `alwaysConflict` map in `merge.go` controls which fields are never silently
filled even when mine is empty. Currently: `{"avatar_url": true}`.

## Algorithm (`merge.go`)

1. Build `mineIndex: map[string]*Person` from mine slice
2. For each person in theirs:
   - If new ID: append to new-persons slice
   - If matched: call `mergePersonData` + `mergePersonRels`, collect conflicts
3. Output = mutated mine (original order) + new persons (theirs order)

**`mergePersonData`**: iterates `PersonData` fields via `reflect`, checks
`alwaysConflict` before applying the empty-fill path. Field names for conflict
reports come from the `json` struct tag, prefixed with `"data."`.

**`mergePersonRels`**: `father`/`mother` use the same empty-fill/conflict logic.
`children`/`spouses` use union (deduplication via `map[string]bool`).

## CLI (`main.go`)

```
family-tree-merger [flags] <mine.json> <theirs.json>

  -output,    -o  string   Output merged file (default "merged.json")
  -conflicts, -c  string   Conflicts report file (default "conflicts.txt")
  -interactive,-i          Resolve conflicts interactively via TUI
```

Both long and short forms share the same variable via `flag.StringVar`/`flag.BoolVar`.

## Conflicts File Format (`conflicts.txt`)

Conflicts are grouped by person — each person appears once with all their
conflicting fields listed below:

```
Family Tree Merge Conflicts
Generated: 2026-02-23 20:14:00
Mine:   mine.json
Theirs: theirs.json
Total conflicts: 2
================================================================================

Person: 6f8d3645-... ("Max" Maximilian Franz Mustermann)
  Field:  data.birth_place
  Mine:   "Musterstadt"
  Theirs: "Berlin"
  Action: kept mine
  Field:  rels.father
  Mine:   "6eb30373-..."
  Theirs: "aaaabbbb-..."
  Action: kept mine

================================================================================
```

Display name format: `"NickName" FirstName SecondNames FamilyName geb. BirthName`
(each part omitted if empty; `geb.` only shown when `birth_name` is set).

If 0 conflicts: `No conflicts found — merge completed cleanly.`

In interactive mode, conflicts resolved to "theirs" are omitted from the report.

## Interactive TUI (`tui.go`)

Activated with `-i`. Uses `charmbracelet/bubbletea` + `charmbracelet/lipgloss`.

Three screens:
1. **Overview** — lists all persons with their conflict fields and values;
   scrollable with ↑/↓ or k/j; scroll indicator `[start–end / total]` in footer
2. **Resolve** — one screen per conflict; shows per-person conflict progress list
   (✓ done / → current / · pending) and side-by-side Mine/Theirs boxes;
   ← / m = mine, → / t = theirs, Enter to confirm
3. **Done** — summary of mine/theirs counts; any key saves and exits

After the TUI, "theirs" choices are applied to the merged data via `applyChoices`
→ `applyTheirsChoice` → `setPersonDataField` (reflection-based field setter by
JSON tag name).

## Edge Cases
- Empty array input: valid (result = other file's content)
- Duplicate IDs within a file: warn to stderr, last occurrence wins
- Missing/null arrays in JSON: nil slice is safe to `range` over
- Dangling ID references in rels: passed through unchanged

## Verification
```bash
go build -o family-tree-merger .
go test ./...

# Non-interactive
./family-tree-merger mine.json theirs.json
./family-tree-merger -o result.json -c report.txt mine.json theirs.json

# Interactive
./family-tree-merger -i mine.json theirs.json
```
