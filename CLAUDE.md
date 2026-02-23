# family-tree-merger — Coding Reference

**Keep this file up-to-date.** Whenever you change behaviour, flags, file structure, or data formats, update the relevant section here before finishing the task.

## File structure

| File | Purpose |
|------|---------|
| `types.go` | `Person`, `PersonData`, `PersonRels`, `Conflict` structs |
| `merge.go` | Merge algorithm, `alwaysConflict` map |
| `main.go` | CLI entry point, file I/O, conflict report writer, `fullName()` |
| `tui.go` | Bubble Tea interactive conflict resolution TUI |
| `merge_test.go` | Tests for all merge rules |
| `testdata/mine.json` | Test fixture — base tree |
| `testdata/theirs.json` | Test fixture — incoming tree |

## Merge algorithm (`merge.go`)

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

The `alwaysConflict` map controls which fields are never silently filled even
when mine is empty. Currently: `{"avatar_url": true}`.

## CLI flags (`main.go`)

Both long and short forms share the same variable via `flag.StringVar`/`flag.BoolVar`.

```
-output/-o      string  Output merged file (default "merged.json")
-conflicts/-c   string  Conflicts report file (default "conflicts.txt")
-interactive/-i bool    Resolve conflicts via TUI
```

## Conflicts file format

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
- Field:  data.birth_place
  Mine:   "Musterstadt"
  Theirs: "Berlin"
  Action: kept mine
- Field:  rels.father
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

Uses `charmbracelet/bubbletea` + `charmbracelet/lipgloss`.

Four screens:
1. **Overview** — lists all persons with their conflict fields and values;
   scrollable with ↑/↓ or k/j; scroll indicator `[start–end / total]` in footer
2. **Resolve** — one screen per conflict; shows per-person conflict progress list
   (✓ done / → current / · pending) and side-by-side Mine/Theirs boxes;
   ← / m = mine, → / t = theirs, Enter to confirm
3. **Done** — summary of mine/theirs counts; any key saves and exits
4. **Exit confirm** — shown when quitting early (q/Ctrl+C); prompts to confirm
   before discarding unresolved choices

After the TUI, "theirs" choices are applied to the merged data via `applyChoices`
→ `applyTheirsChoice` → `setPersonDataField` (reflection-based field setter by
JSON tag name).

## Edge cases

- Empty array input: valid (result = other file's content)
- Duplicate IDs within a file: warn to stderr, last occurrence wins
- Missing/null arrays in JSON: nil slice is safe to `range` over
- Dangling ID references in rels: passed through unchanged
