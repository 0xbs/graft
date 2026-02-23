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
	var outputPath, conflictsPath string
	var interactive bool
	flag.StringVar(&outputPath, "output", "merged.json", "Output merged file path")
	flag.StringVar(&outputPath, "o", "merged.json", "")
	flag.StringVar(&conflictsPath, "conflicts", "conflicts.txt", "Output conflicts report path")
	flag.StringVar(&conflictsPath, "c", "conflicts.txt", "")
	flag.BoolVar(&interactive, "interactive", false, "Resolve conflicts interactively")
	flag.BoolVar(&interactive, "i", false, "")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: family-tree-merger [flags] <mine.json> <theirs.json>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

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

	merged, conflicts := merge(mine, theirs)
	newCount := len(merged) - len(mine)

	if interactive && len(conflicts) > 0 {
		merged, conflicts, err = runInteractive(conflicts, merged)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in interactive mode: %v\n", err)
			os.Exit(1)
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
				fmt.Fprintf(&sb, "  Field:  %s\n", c.Field)
				fmt.Fprintf(&sb, "  Mine:   %q\n", c.Mine)
				fmt.Fprintf(&sb, "  Theirs: %q\n", c.Theirs)
				fmt.Fprintf(&sb, "  Action: kept mine\n")
			}
		}
	}

	sb.WriteString("\n" + strings.Repeat("=", 80) + "\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// fullName assembles a display name from a person's name fields.
// Format: "NickName" FirstName SecondNames FamilyName geb. BirthName
func fullName(p Person) string {
	var parts []string
	if p.Data.NickName != "" {
		parts = append(parts, `"`+p.Data.NickName+`"`)
	}
	if p.Data.FirstName != "" {
		parts = append(parts, p.Data.FirstName)
	}
	if p.Data.SecondNames != "" {
		parts = append(parts, p.Data.SecondNames)
	}
	if p.Data.FamilyName != "" {
		parts = append(parts, p.Data.FamilyName)
	}
	name := strings.Join(parts, " ")
	if p.Data.BirthName != "" {
		name += " geb. " + p.Data.BirthName
	}
	return name
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
