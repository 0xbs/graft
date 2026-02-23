package main

import (
	"reflect"
	"strings"
)

// merge integrates theirs into mine and returns the merged slice and any conflicts.
// Persons in theirs not present in mine are appended. For matched persons,
// empty fields in mine are filled from theirs; differing non-empty fields are conflicts.
func merge(mine, theirs []Person) ([]Person, []Conflict) {
	mineIndex := make(map[string]*Person, len(mine))
	for i := range mine {
		p := &mine[i]
		if _, dup := mineIndex[p.ID]; dup {
			// warn handled by caller; last occurrence wins
		}
		mineIndex[p.ID] = p
	}

	var conflicts []Conflict
	var newPersons []Person

	for _, t := range theirs {
		m, exists := mineIndex[t.ID]
		if !exists {
			newPersons = append(newPersons, t)
			continue
		}
		conflicts = append(conflicts, mergePersonData(m, t)...)
		conflicts = append(conflicts, mergePersonRels(m, t)...)
	}

	merged := make([]Person, len(mine), len(mine)+len(newPersons))
	copy(merged, mine)
	merged = append(merged, newPersons...)
	return merged, conflicts
}

// alwaysConflict lists fields that are always treated as conflicts when theirs
// differs from mine, even if mine is empty. This prevents silent adoption of
// values that require external resources (e.g. image files) to be useful.
var alwaysConflict = map[string]bool{
	"avatar_url": true,
}

// mergePersonData merges t's data fields into m using reflection.
// Empty-fills non-conflicting fields; records conflicts for differing non-empty values.
// Fields in alwaysConflict are never silently filled, even when mine is empty.
func mergePersonData(m *Person, t Person) []Conflict {
	var conflicts []Conflict

	mVal := reflect.ValueOf(&m.Data).Elem()
	tVal := reflect.ValueOf(t.Data)
	typ := reflect.TypeOf(t.Data)

	for i := 0; i < typ.NumField(); i++ {
		mv := mVal.Field(i).String()
		tv := tVal.Field(i).String()

		if mv == tv || tv == "" {
			continue
		}
		fieldTag := jsonFieldName(typ.Field(i))
		if mv == "" && !alwaysConflict[fieldTag] {
			mVal.Field(i).SetString(tv)
			continue
		}
		// both non-empty and different, or always-conflict field → conflict
		conflicts = append(conflicts, Conflict{
			PersonID: m.ID,
			Field:    "data." + fieldTag,
			Mine:     mv,
			Theirs:   tv,
		})
	}
	return conflicts
}

// mergePersonRels merges t's relationship fields into m.
// father/mother use the same empty-fill/conflict logic as data fields.
// children/spouses use union logic (no conflicts possible).
func mergePersonRels(m *Person, t Person) []Conflict {
	var conflicts []Conflict

	// Single-ref fields
	singleRefs := []struct {
		name  string
		mine  *string
		theirs string
	}{
		{"rels.father", &m.Rels.Father, t.Rels.Father},
		{"rels.mother", &m.Rels.Mother, t.Rels.Mother},
	}
	for _, ref := range singleRefs {
		mv, tv := *ref.mine, ref.theirs
		if mv == tv || tv == "" {
			continue
		}
		if mv == "" {
			*ref.mine = tv
			continue
		}
		conflicts = append(conflicts, Conflict{
			PersonID: m.ID,
			Field:    ref.name,
			Mine:     mv,
			Theirs:   tv,
		})
	}

	// Array fields: union
	m.Rels.Children = unionStrings(m.Rels.Children, t.Rels.Children)
	m.Rels.Spouses = unionStrings(m.Rels.Spouses, t.Rels.Spouses)

	return conflicts
}

// unionStrings returns base with any elements from additions not already present.
func unionStrings(base, additions []string) []string {
	if len(additions) == 0 {
		return base
	}
	existing := make(map[string]bool, len(base))
	for _, id := range base {
		existing[id] = true
	}
	for _, id := range additions {
		if !existing[id] {
			base = append(base, id)
		}
	}
	return base
}

// jsonFieldName extracts the JSON field name from a struct field's tag.
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return strings.ToLower(f.Name)
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}
