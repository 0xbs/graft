package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Issue represents a single validation error or warning.
type Issue struct {
	PersonID string
	Message  string
}

// ValidationResult holds errors and warnings from a validate run.
type ValidationResult struct {
	Errors   []Issue
	Warnings []Issue
}

var (
	reDateFull = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reDateYear = regexp.MustCompile(`^\d{4}$`)

	// dateFieldSuffixes lists field name suffixes that indicate a date field.
	dateFieldSuffixes = []string{"_date", " date"}
)

func isDateField(name string) bool {
	if name == "birthday" {
		return true
	}
	for _, suffix := range dateFieldSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// validate checks persons for structural errors and data quality warnings.
// rawJSON is the original file bytes used to detect duplicate JSON keys; it may be nil.
func validate(persons []Person, rawJSON []byte) ValidationResult {
	var result ValidationResult

	addErr := func(id, msg string) {
		result.Errors = append(result.Errors, Issue{PersonID: id, Message: msg})
	}
	addWarn := func(id, msg string) {
		result.Warnings = append(result.Warnings, Issue{PersonID: id, Message: msg})
	}

	// Build ID index.
	index := make(map[string]*Person, len(persons))
	for i := range persons {
		index[persons[i].ID] = &persons[i]
	}

	// Count per-field usage across all persons (for rare-field warnings).
	fieldCounts := make(map[string]int)
	for i := range persons {
		for k := range persons[i].Data {
			fieldCounts[k]++
		}
	}

	// Check duplicate IDs.
	seen := make(map[string]bool, len(persons))
	for _, p := range persons {
		if seen[p.ID] {
			addErr(p.ID, "duplicate ID")
		}
		seen[p.ID] = true
	}

	// Per-person checks.
	for i := range persons {
		p := &persons[i]
		id := p.ID

		// ── Errors ──────────────────────────────────────────────────────────

		// Missing or invalid gender.
		gender, hasGender := p.Data["gender"]
		if !hasGender || gender == "" {
			addErr(id, "missing gender field")
		} else if gender != "M" && gender != "F" {
			addErr(id, fmt.Sprintf("invalid gender %q (must be M or F)", gender))
		}

		// No relations at all.
		if p.Rels.Father == "" && p.Rels.Mother == "" &&
			len(p.Rels.Children) == 0 && len(p.Rels.Spouses) == 0 {
			addErr(id, "person has no relations to anyone")
		}

		// Reciprocal: father must list this person as a child.
		if p.Rels.Father != "" {
			father, ok := index[p.Rels.Father]
			if !ok {
				addErr(id, fmt.Sprintf("father %q does not exist", p.Rels.Father))
			} else if !containsStr(father.Rels.Children, id) {
				addErr(id, fmt.Sprintf("father %q does not list this person as a child", p.Rels.Father))
			}
		}

		// Reciprocal: mother must list this person as a child.
		if p.Rels.Mother != "" {
			mother, ok := index[p.Rels.Mother]
			if !ok {
				addErr(id, fmt.Sprintf("mother %q does not exist", p.Rels.Mother))
			} else if !containsStr(mother.Rels.Children, id) {
				addErr(id, fmt.Sprintf("mother %q does not list this person as a child", p.Rels.Mother))
			}
		}

		// Reciprocal: each listed child must reference this person as a parent.
		for _, childID := range p.Rels.Children {
			child, ok := index[childID]
			if !ok {
				addErr(id, fmt.Sprintf("child %q does not exist", childID))
			} else if child.Rels.Father != id && child.Rels.Mother != id {
				addErr(id, fmt.Sprintf("child %q does not list this person as a parent", childID))
			}
		}

		// Reciprocal: each listed spouse must list this person back.
		for _, spouseID := range p.Rels.Spouses {
			spouse, ok := index[spouseID]
			if !ok {
				addErr(id, fmt.Sprintf("spouse %q does not exist", spouseID))
			} else if !containsStr(spouse.Rels.Spouses, id) {
				addErr(id, fmt.Sprintf("spouse %q does not list this person as a spouse", spouseID))
			}
		}

		// Invalid calendar dates: *_date / birthday fields matching yyyy-mm-dd format.
		for k, v := range p.Data {
			if isDateField(k) && reDateFull.MatchString(v) {
				if _, err := time.Parse("2006-01-02", v); err != nil {
					addErr(id, fmt.Sprintf("field %q contains invalid date %q", k, v))
				}
			}
		}

		// ── Warnings ────────────────────────────────────────────────────────

		// Date fields with non-standard values (not yyyy or yyyy-mm-dd).
		for k, v := range p.Data {
			if isDateField(k) && v != "" && !reDateYear.MatchString(v) && !reDateFull.MatchString(v) {
				addWarn(id, fmt.Sprintf("date field %q has non-standard value %q (expected yyyy or yyyy-mm-dd)", k, v))
			}
		}

		// Rare fields: used by exactly 1 person in a dataset of ≥5.
		if len(persons) >= 5 {
			for k := range p.Data {
				if fieldCounts[k] == 1 {
					addWarn(id, fmt.Sprintf("field %q is only used by this person — possible typo", k))
				}
			}
		}
	}

	// Persons with identical data AND at least one relation in common but different IDs.
	dataMap := make(map[string][]*Person) // canonical → persons
	for i := range persons {
		key := canonicalData(persons[i].Data)
		if key != "" {
			dataMap[key] = append(dataMap[key], &persons[i])
		}
	}
	for _, group := range dataMap {
		if len(group) < 2 {
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]
				if shareRelation(a, b) {
					ids := []string{a.ID, b.ID}
					sort.Strings(ids)
					result.Warnings = append(result.Warnings, Issue{
						PersonID: strings.Join(ids, ", "),
						Message:  "persons have exactly matching data fields but different IDs",
					})
				}
			}
		}
	}

	// Duplicate JSON keys in data objects.
	if rawJSON != nil {
		for personID, fields := range findDuplicateDataKeys(rawJSON) {
			for _, field := range fields {
				addErr(personID, fmt.Sprintf("duplicate field %q in data", field))
			}
		}
	}

	return result
}

// containsStr reports whether s is in slice.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// shareRelation reports whether two persons share at least one relation:
// the same father, the same mother, a common child, or a common spouse.
func shareRelation(a, b *Person) bool {
	if a.Rels.Father != "" && a.Rels.Father == b.Rels.Father {
		return true
	}
	if a.Rels.Mother != "" && a.Rels.Mother == b.Rels.Mother {
		return true
	}
	for _, id := range a.Rels.Children {
		if containsStr(b.Rels.Children, id) {
			return true
		}
	}
	for _, id := range a.Rels.Spouses {
		if containsStr(b.Rels.Spouses, id) {
			return true
		}
	}
	return false
}

// canonicalData returns a stable string key for a PersonData map.
func canonicalData(data PersonData) string {
	if len(data) == 0 {
		return ""
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\x00')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(data[k])
	}
	return b.String()
}

// findDuplicateDataKeys scans raw JSON (a persons array) and returns a map of
// personID → list of duplicate key names found inside that person's "data" object.
func findDuplicateDataKeys(raw []byte) map[string][]string {
	result := make(map[string][]string)

	dec := json.NewDecoder(bytes.NewReader(raw))

	// Expect opening '['.
	tok, err := dec.Token()
	if err != nil {
		return result
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return result
	}

	for dec.More() {
		var rawPerson json.RawMessage
		if err := dec.Decode(&rawPerson); err != nil {
			break
		}
		personID, dupes := personDuplicateKeys(rawPerson)
		if len(dupes) > 0 {
			result[personID] = dupes
		}
	}

	return result
}

// personDuplicateKeys extracts the id and any duplicate keys in the "data" field of rawPerson.
func personDuplicateKeys(raw json.RawMessage) (string, []string) {
	dec := json.NewDecoder(bytes.NewReader(raw))

	// Expect opening '{'.
	if _, err := dec.Token(); err != nil {
		return "", nil
	}

	var personID string
	var dupes []string

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			break
		}
		key, ok := keyTok.(string)
		if !ok {
			break
		}

		switch key {
		case "id":
			var id string
			if err := dec.Decode(&id); err == nil {
				personID = id
			}
		case "data":
			dupes = scanDuplicateKeys(dec)
		default:
			var skip json.RawMessage
			dec.Decode(&skip) //nolint:errcheck
		}
	}

	return personID, dupes
}

// scanDuplicateKeys reads a JSON object from dec (starting with '{') and returns
// any key names that appear more than once.
func scanDuplicateKeys(dec *json.Decoder) []string {
	// Expect opening '{'.
	if _, err := dec.Token(); err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var dupes []string

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			break
		}
		key, ok := keyTok.(string)
		if !ok {
			break
		}

		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			break
		}

		if seen[key] {
			dupes = append(dupes, key)
		}
		seen[key] = true
	}

	// Read closing '}'.
	dec.Token() //nolint:errcheck

	return dupes
}
