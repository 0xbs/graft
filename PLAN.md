# Plan: family-tree-merger CLI Tool

## Context
Build a Go CLI tool that merges two family tree JSON files. The user and their brother-in-law each maintain a family tree in the same JSON format. The tool integrates the brother-in-law's changes into the user's tree: adding new persons, filling empty fields, and reporting any conflicting values for manual resolution.

## Data Format (example.json)
JSON array of persons, each with:
- `id`: UUID (used for matching)
- `data`: object with 17 string fields (gender, names, dates, places, note, avatar_url)
- `rels`: object with `father`/`mother` (string IDs) and `children`/`spouses` (string ID arrays)

## File Structure
3 files in `package main`:
- `types.go` — all structs
- `merge.go` — merge algorithm
- `main.go` — CLI entry point, file I/O, output

## Data Structures (`types.go`)

```go
type PersonData struct {
    Gender, NickName, FirstName, SecondNames, FamilyName, BirthName string
    BirthDate, BirthPlace, ResidencePlace string
    DeathDate, DeathPlace, BurialPlace string
    MarriageDate, MarriagePlace, DivorceDate string
    AvatarURL, Note string
    // all with json:"snake_case" tags
}

type PersonRels struct {
    Father   string   `json:"father,omitempty"`
    Mother   string   `json:"mother,omitempty"`
    Children []string `json:"children,omitempty"`
    Spouses  []string `json:"spouses,omitempty"`
}

type Person struct {
    ID   string     `json:"id"`
    Data PersonData `json:"data"`
    Rels PersonRels `json:"rels"`
}

type Conflict struct {
    PersonID, Field, Mine, Theirs string
}
```

## Merge Rules

| Case | Action |
|------|--------|
| ID in theirs not in mine | Add new person to merged output |
| data/rels field: mine empty, theirs non-empty | Use theirs (no conflict) |
| data/rels field: mine non-empty, theirs empty | Keep mine (no conflict) |
| data/rels field: both equal | No conflict |
| data/rels field: both non-empty, different | Conflict — keep mine, log conflict |
| children/spouses arrays | Union (add missing IDs from theirs) — no conflicts |

## Algorithm (`merge.go`)

1. Build `mineIndex: map[string]*Person` from mine slice
2. Track new persons (in theirs but not in mine) in a separate slice
3. For each person in theirs:
   - If new: append to new-persons slice
   - If matched: call `mergePersonData(m, t)` and `mergePersonRels(m, t)`, collect conflicts
4. Output = original mine order (mutated) + new persons in theirs order

**mergePersonData**: Use `reflect` to iterate `PersonData` fields generically (avoids 17 if-statements). Field names for conflict reports derived from `json` struct tag, prefixed with `"data."`.

**mergePersonRels**: Handle `father`/`mother` like data fields. For `children`/`spouses`: build set from mine's existing IDs, append any missing IDs from theirs.

## CLI (`main.go`)

```
family-tree-merger [flags] <mine.json> <theirs.json>
  -output    string   Output merged file (default "merged.json")
  -conflicts string   Conflicts report file (default "conflicts.txt")
```

Orchestration:
1. Parse flags + 2 positional args (exit 1 with usage if wrong)
2. `loadJSON(path) -> []Person`
3. `merge(mine, theirs) -> ([]Person, []Conflict)`
4. `writeJSON(outputPath, merged)` using `json.MarshalIndent` (2-space indent)
5. `writeConflicts(conflictsPath, ...)` — always written, even if 0 conflicts
6. Print summary to stdout

## Conflicts File Format (`conflicts.txt`)

```
Family Tree Merge Conflicts
Generated: 2026-02-23 20:14:00
Mine:   mine.json
Theirs: theirs.json
Total conflicts: 2
================================================================================

Person: 6f8d3645-... (Maximilian Mustermann)
  Field:  data.birth_place
  Mine:   "Musterstadt"
  Theirs: "Berlin"
  Action: kept mine

Person: 6f8d3645-... (Maximilian Mustermann)
  Field:  rels.father
  Mine:   "6eb30373-..."
  Theirs: "aaaabbbb-..."
  Action: kept mine

================================================================================
```
If 0 conflicts: `No conflicts found — merge completed cleanly.`

Display name = `data.first_name + " " + data.family_name` (informational).

## Edge Cases
- Empty array input is valid (result = other file's content)
- Duplicate IDs within a file: warn to stderr, last occurrence wins
- Missing/null arrays in JSON: Go nil slice handles `range` safely
- Dangling ID references in rels: pass through unchanged

## Verification
```bash
# Build
go build -o family-tree-merger .

# Smoke test: copy example.json, edit one field to create a conflict
cp example.json mine.json
cp example.json theirs.json
# edit theirs.json: change birth_place to something different
./family-tree-merger mine.json theirs.json
# Check merged.json has mine's birth_place
# Check conflicts.txt lists the birth_place conflict

# Test new person addition: add a new person object to theirs.json
# Check merged.json contains the new person
```
