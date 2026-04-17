package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  graft merge [flags] <mine.json> <theirs.json>\n")
	fmt.Fprintf(os.Stderr, "  graft validate <file.json>\n")
	fmt.Fprintf(os.Stderr, "  graft subtree [flags] <file.json>\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  merge     Merge two family tree files\n")
	fmt.Fprintf(os.Stderr, "  validate  Validate a file for errors and warnings\n")
	fmt.Fprintf(os.Stderr, "  subtree   Extract a connected subtree starting from a person\n\n")
	fmt.Fprintf(os.Stderr, "Run 'graft <command> -help' for command-specific flags.\n")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "merge":
		runMergeCmd(os.Args[2:])
	case "validate":
		runValidateCmd(os.Args[2:])
	case "subtree":
		runSubtreeCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runMergeCmd(args []string) {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	var outputPath, conflictsPath, alwaysConflictFlag string
	var interactive bool
	fs.StringVar(&outputPath, "output", "merged.json", "Output merged file path")
	fs.StringVar(&outputPath, "o", "merged.json", "")
	fs.StringVar(&conflictsPath, "conflicts", "conflicts.txt", "Output conflicts report path")
	fs.StringVar(&conflictsPath, "c", "conflicts.txt", "")
	fs.BoolVar(&interactive, "interactive", false, "Resolve conflicts interactively")
	fs.BoolVar(&interactive, "i", false, "")
	fs.StringVar(&alwaysConflictFlag, "always-conflict", "avatar_url,avatar", "Comma-separated data fields to always treat as conflicts, even when mine is empty")
	fs.StringVar(&alwaysConflictFlag, "ac", "avatar_url,avatar", "")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: graft merge [flags] <mine.json> <theirs.json>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) != 2 {
		fs.Usage()
		os.Exit(1)
	}
	minePath, theirsPath := remaining[0], remaining[1]

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

func runValidateCmd(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: graft validate <file.json>\n")
	}
	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) != 1 {
		fs.Usage()
		os.Exit(1)
	}
	runValidate(remaining[0])
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

	// Build ID → Person index for name lookup.
	index := make(map[string]Person, len(persons))
	for _, p := range persons {
		index[p.ID] = p
	}

	// personLabel returns "Name (id)" for a single ID, or a comma-separated
	// list when PersonID holds multiple IDs (e.g. the identical-persons warning).
	personLabel := func(personID string) string {
		ids := strings.Split(personID, ", ")
		parts := make([]string, 0, len(ids))
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if p, ok := index[id]; ok {
				if name := fullName(p); name != "" {
					parts = append(parts, fmt.Sprintf("%s (%s)", name, id))
					continue
				}
			}
			parts = append(parts, id)
		}
		return strings.Join(parts, ", ")
	}

	printIssues := func(issues []Issue) {
		for _, issue := range issues {
			fmt.Printf("  %s\n  → %s\n\n", personLabel(issue.PersonID), issue.Message)
		}
	}

	fmt.Printf("Validating %s — %d persons\n", path, len(persons))
	fmt.Println(strings.Repeat("─", 60))

	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		fmt.Println("No issues found.")
		return
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors (%d)\n\n", len(result.Errors))
		printIssues(result.Errors)
	}
	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d)\n\n", len(result.Warnings))
		printIssues(result.Warnings)
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Result: %d error(s), %d warning(s)\n",
		len(result.Errors), len(result.Warnings))

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

func runSubtreeCmd(args []string) {
	fs := flag.NewFlagSet("subtree", flag.ExitOnError)
	var fromID, stopFlag, outputPath string
	fs.StringVar(&fromID, "from", "", "Start person ID (required)")
	fs.StringVar(&stopFlag, "stop", "", "Comma-separated IDs at which traversal stops")
	fs.StringVar(&stopFlag, "s", "", "")
	fs.StringVar(&outputPath, "output", "", "Output file path (default: stdout)")
	fs.StringVar(&outputPath, "o", "", "")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: graft subtree [flags] <file.json>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) != 1 || fromID == "" {
		fs.Usage()
		os.Exit(1)
	}

	persons, err := loadJSON(remaining[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", remaining[0], err)
		os.Exit(1)
	}

	stopIDs := parseCommaSeparated(stopFlag)

	result, err := subtree(persons, fromID, stopIDs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputPath, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Extracted %d persons -> %s\n", len(result), outputPath)
	} else {
		fmt.Println(string(data))
	}
}

// parseCommaSeparated parses a comma-separated list of strings into a set,
// returning nil for an empty input.
func parseCommaSeparated(s string) map[string]bool {
	if s == "" {
		return nil
	}
	result := make(map[string]bool)
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			result[f] = true
		}
	}
	return result
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
