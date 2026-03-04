package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	var outputPath, conflictsPath, alwaysConflictFlag string
	var interactive, validateMode bool
	flag.StringVar(&outputPath, "output", "merged.json", "Output merged file path")
	flag.StringVar(&outputPath, "o", "merged.json", "")
	flag.StringVar(&conflictsPath, "conflicts", "conflicts.txt", "Output conflicts report path")
	flag.StringVar(&conflictsPath, "c", "conflicts.txt", "")
	flag.BoolVar(&interactive, "interactive", false, "Resolve conflicts interactively")
	flag.BoolVar(&interactive, "i", false, "")
	flag.StringVar(&alwaysConflictFlag, "always-conflict", "avatar_url,avatar", "Comma-separated data fields to always treat as conflicts, even when mine is empty")
	flag.StringVar(&alwaysConflictFlag, "ac", "avatar_url,avatar", "")
	flag.BoolVar(&validateMode, "validate", false, "Validate a file for errors and warnings instead of merging")
	flag.BoolVar(&validateMode, "v", false, "")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  graft [flags] <mine.json> <theirs.json>   merge two files\n")
		fmt.Fprintf(os.Stderr, "  graft -validate <file.json>               validate a file\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if validateMode {
		args := flag.Args()
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "Usage: graft -validate <file.json>\n")
			os.Exit(1)
		}
		runValidate(args[0])
		return
	}

	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}
	minePath, theirsPath := args[0], args[1]

	mine, err := loadJSON(minePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", minePath, err)
		os.Exit(1)
	}
	theirs, err := loadJSON(theirsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", theirsPath, err)
		os.Exit(1)
	}

	// Warn about duplicate IDs
	warnDuplicates(mine, minePath)
	warnDuplicates(theirs, theirsPath)

	alwaysConflict := parseAlwaysConflict(alwaysConflictFlag)

	merged, conflicts := merge(mine, theirs, alwaysConflict)
	newCount := len(merged) - len(mine)

	if interactive && len(conflicts) > 0 {
		var aborted bool
		merged, conflicts, aborted, err = runInteractive(conflicts, merged)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in interactive mode: %v\n", err)
			os.Exit(1)
		}
		if aborted {
			fmt.Fprintln(os.Stderr, "Aborted — no output files written.")
			os.Exit(0)
		}
	}

	if err := writeJSON(outputPath, merged); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputPath, err)
		os.Exit(1)
	}
	if err := writeConflicts(conflictsPath, conflicts, merged, minePath, theirsPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", conflictsPath, err)
		os.Exit(1)
	}

	fmt.Printf("Merged %d persons (%d new, %d updated) -> %s\n",
		len(merged), newCount, len(merged)-newCount, outputPath)
	fmt.Printf("Conflicts: %d -> %s\n", len(conflicts), conflictsPath)
}

// runValidate loads path, runs validation, prints all issues, and exits 1 if there are errors.
func runValidate(path string) {
	rawJSON, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
		os.Exit(1)
	}
	var persons []Person
	if err := json.Unmarshal(rawJSON, &persons); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", path, err)
		os.Exit(1)
	}

	result := validate(persons, rawJSON)

	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		fmt.Printf("Validated %d persons — no issues found.\n", len(persons))
		return
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Errors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  [%s] %s\n", e.PersonID, e.Message)
		}
	}
	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings (%d):\n", len(result.Warnings))
		for _, w := range result.Warnings {
			fmt.Printf("  [%s] %s\n", w.PersonID, w.Message)
		}
	}

	fmt.Printf("\nValidated %d persons: %d error(s), %d warning(s)\n",
		len(persons), len(result.Errors), len(result.Warnings))

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

// parseAlwaysConflict parses a comma-separated list of field names into a set.
func parseAlwaysConflict(s string) map[string]bool {
	result := make(map[string]bool)
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			result[f] = true
		}
	}
	return result
}

func loadJSON(path string) ([]Person, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var persons []Person
	if err := json.Unmarshal(data, &persons); err != nil {
		return nil, err
	}
	return persons, nil
}

func writeJSON(path string, persons []Person) error {
	data, err := json.MarshalIndent(persons, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeConflicts(path string, conflicts []Conflict, persons []Person, minePath, theirsPath string) error {
	// Build ID → Person index for name lookup.
	personIndex := make(map[string]Person, len(persons))
	for _, p := range persons {
		personIndex[p.ID] = p
	}

	// Group conflicts by PersonID, preserving first-occurrence order.
	var order []string
	grouped := make(map[string][]Conflict)
	for _, c := range conflicts {
		if _, seen := grouped[c.PersonID]; !seen {
			order = append(order, c.PersonID)
		}
		grouped[c.PersonID] = append(grouped[c.PersonID], c)
	}

	var sb strings.Builder

	sb.WriteString("Family Tree Merge Conflicts\n")
	fmt.Fprintf(&sb, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "Mine:   %s\n", minePath)
	fmt.Fprintf(&sb, "Theirs: %s\n", theirsPath)
	fmt.Fprintf(&sb, "Total conflicts: %d\n", len(conflicts))
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	if len(conflicts) == 0 {
		sb.WriteString("\nNo conflicts found — merge completed cleanly.\n")
	} else {
		for _, id := range order {
			p := personIndex[id]
			sb.WriteString("\n")
			fmt.Fprintf(&sb, "Person: %s (%s)\n", id, fullName(p))
			for _, c := range grouped[id] {
				fmt.Fprintf(&sb, "- Field:  %s\n", c.Field)
				fmt.Fprintf(&sb, "  Mine:   %q\n", c.Mine)
				fmt.Fprintf(&sb, "  Theirs: %q\n", c.Theirs)
				fmt.Fprintf(&sb, "  Action: kept mine\n")
			}
		}
	}

	sb.WriteString("\n" + strings.Repeat("=", 80) + "\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// nameFields lists data keys whose non-empty values are joined to form a display name.
// Alternatives for the same concept sit next to each other; real data will only have one.
var nameFields = []string{
	"nick_name", "nick name",
	"first_name", "first name",
	"family_name", "last_name", "last name",
	"birth_name", "birth name",
	"birth_date", "birthday",
}

// fullName assembles a display name by joining all nameFields values that are non-empty.
func fullName(p Person) string {
	var parts []string
	for _, key := range nameFields {
		if v := p.Data[key]; v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

func warnDuplicates(persons []Person, path string) {
	seen := make(map[string]bool, len(persons))
	for _, p := range persons {
		if seen[p.ID] {
			fmt.Fprintf(os.Stderr, "Warning: duplicate ID %s in %s, using last occurrence\n", p.ID, path)
		}
		seen[p.ID] = true
	}
}
