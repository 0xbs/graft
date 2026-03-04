package main

import (
	"fmt"
	"strings"
	"testing"
)

// helpers

func hasError(result ValidationResult, substr string) bool {
	for _, e := range result.Errors {
		if strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

func hasErrorFor(result ValidationResult, id, substr string) bool {
	for _, e := range result.Errors {
		if e.PersonID == id && strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

func hasWarningFor(result ValidationResult, id, substr string) bool {
	for _, w := range result.Warnings {
		if w.PersonID == id && strings.Contains(w.Message, substr) {
			return true
		}
	}
	return false
}

func noErrors(t *testing.T, result ValidationResult) {
	t.Helper()
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func noWarnings(t *testing.T, result ValidationResult) {
	t.Helper()
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

// minimal valid two-person family: parent + child, reciprocal rels, correct gender.
func validFamily() []Person {
	return []Person{
		{
			ID:   "parent",
			Data: PersonData{"gender": "M", "first_name": "Adam"},
			Rels: PersonRels{Children: []string{"child"}, Spouses: []string{"spouse"}},
		},
		{
			ID:   "spouse",
			Data: PersonData{"gender": "F", "first_name": "Eve"},
			Rels: PersonRels{Children: []string{"child"}, Spouses: []string{"parent"}},
		},
		{
			ID:   "child",
			Data: PersonData{"gender": "M", "first_name": "Cain"},
			Rels: PersonRels{Father: "parent", Mother: "spouse"},
		},
	}
}

// ── Gender ────────────────────────────────────────────────────────────────────

func TestValidate_MissingGender(t *testing.T) {
	persons := validFamily()
	persons[0].Data = PersonData{"first_name": "Adam"} // remove gender
	result := validate(persons, nil)
	if !hasErrorFor(result, "parent", "missing gender") {
		t.Error("expected missing gender error for parent")
	}
}

func TestValidate_InvalidGender(t *testing.T) {
	persons := validFamily()
	persons[0].Data["gender"] = "X"
	result := validate(persons, nil)
	if !hasErrorFor(result, "parent", "invalid gender") {
		t.Error("expected invalid gender error for parent")
	}
}

func TestValidate_ValidGenders(t *testing.T) {
	persons := validFamily()
	result := validate(persons, nil)
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "gender") {
			t.Errorf("unexpected gender error: %v", e)
		}
	}
}

// ── Isolation ─────────────────────────────────────────────────────────────────

func TestValidate_IsolatedPerson(t *testing.T) {
	persons := []Person{
		{ID: "lone", Data: PersonData{"gender": "M"}},
	}
	result := validate(persons, nil)
	if !hasErrorFor(result, "lone", "no relations") {
		t.Error("expected isolated-person error")
	}
}

// ── Reciprocal relations ───────────────────────────────────────────────────────

func TestValidate_FatherMissingChildRef(t *testing.T) {
	persons := []Person{
		{
			ID:   "dad",
			Data: PersonData{"gender": "M"},
			Rels: PersonRels{Children: []string{}}, // dad does NOT list child
		},
		{
			ID:   "kid",
			Data: PersonData{"gender": "F"},
			Rels: PersonRels{Father: "dad"},
		},
	}
	result := validate(persons, nil)
	if !hasErrorFor(result, "kid", "does not list this person as a child") {
		t.Error("expected reciprocal father-child error on kid")
	}
}

func TestValidate_ChildMissingParentRef(t *testing.T) {
	persons := []Person{
		{
			ID:   "dad",
			Data: PersonData{"gender": "M"},
			Rels: PersonRels{Children: []string{"kid"}},
		},
		{
			ID:   "kid",
			Data: PersonData{"gender": "F"},
			Rels: PersonRels{}, // kid does NOT reference dad as father or mother
		},
	}
	result := validate(persons, nil)
	if !hasErrorFor(result, "dad", "does not list this person as a parent") {
		t.Error("expected reciprocal child-parent error on dad")
	}
}

func TestValidate_SpouseMissingReciprocal(t *testing.T) {
	persons := []Person{
		{
			ID:   "alice",
			Data: PersonData{"gender": "F"},
			Rels: PersonRels{Spouses: []string{"bob"}},
		},
		{
			ID:   "bob",
			Data: PersonData{"gender": "M"},
			Rels: PersonRels{}, // bob does NOT list alice as spouse
		},
	}
	result := validate(persons, nil)
	if !hasErrorFor(result, "alice", "does not list this person as a spouse") {
		t.Error("expected reciprocal spouse error on alice")
	}
}

func TestValidate_MissingRelationTarget(t *testing.T) {
	persons := []Person{
		{
			ID:   "alice",
			Data: PersonData{"gender": "F"},
			Rels: PersonRels{Spouses: []string{"ghost"}},
		},
	}
	result := validate(persons, nil)
	if !hasErrorFor(result, "alice", `"ghost" does not exist`) {
		t.Error("expected missing spouse target error")
	}
}

func TestValidate_ReciprocalsOK(t *testing.T) {
	persons := validFamily()
	result := validate(persons, nil)
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "does not list") || strings.Contains(e.Message, "does not exist") {
			t.Errorf("unexpected relation error: %v", e)
		}
	}
}

// ── Duplicate fields (raw JSON) ───────────────────────────────────────────────

func TestValidate_DuplicateDataKeys(t *testing.T) {
	raw := []byte(`[{"id":"p1","data":{"gender":"M","first_name":"A","first_name":"B"},"rels":{}}]`)
	persons := []Person{
		{ID: "p1", Data: PersonData{"gender": "M", "first_name": "B"}, Rels: PersonRels{Spouses: []string{}}},
	}
	result := validate(persons, raw)
	if !hasErrorFor(result, "p1", `duplicate field "first_name"`) {
		t.Error("expected duplicate field error for first_name")
	}
}

// ── Identical data, different IDs ─────────────────────────────────────────────

func TestValidate_IdenticalPersonsDifferentIDs(t *testing.T) {
	persons := []Person{
		{
			ID:   "a1",
			Data: PersonData{"gender": "M", "first_name": "John"},
			Rels: PersonRels{Spouses: []string{"a2"}},
		},
		{
			ID:   "a2",
			Data: PersonData{"gender": "M", "first_name": "John"},
			Rels: PersonRels{Spouses: []string{"a1"}},
		},
	}
	result := validate(persons, nil)
	if !hasError(result, "exactly matching data fields") {
		t.Error("expected identical-persons error")
	}
}

func TestValidate_DifferentPersonsNotFlagged(t *testing.T) {
	persons := validFamily()
	result := validate(persons, nil)
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "exactly matching") {
			t.Errorf("unexpected identical-persons error: %v", e)
		}
	}
}

// ── Duplicate IDs ────────────────────────────────────────────────────────────

func TestValidate_DuplicateIDs(t *testing.T) {
	persons := []Person{
		{ID: "dup", Data: PersonData{"gender": "M"}, Rels: PersonRels{Spouses: []string{}}},
		{ID: "dup", Data: PersonData{"gender": "F"}, Rels: PersonRels{Spouses: []string{}}},
	}
	result := validate(persons, nil)
	if !hasErrorFor(result, "dup", "duplicate ID") {
		t.Error("expected duplicate ID error")
	}
}

// ── Invalid dates ─────────────────────────────────────────────────────────────

func TestValidate_InvalidCalendarDate(t *testing.T) {
	persons := []Person{
		{
			ID:   "p1",
			Data: PersonData{"gender": "M", "birth_date": "2024-02-30"},
			Rels: PersonRels{Spouses: []string{}},
		},
	}
	result := validate(persons, nil)
	if !hasErrorFor(result, "p1", "invalid date") {
		t.Error("expected invalid date error for birth_date")
	}
}

func TestValidate_ValidCalendarDate(t *testing.T) {
	persons := validFamily()
	persons[0].Data["birth_date"] = "1990-06-15"
	result := validate(persons, nil)
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "invalid date") {
			t.Errorf("unexpected invalid date error: %v", e)
		}
	}
}

func TestValidate_DateOnlyYear_NoError(t *testing.T) {
	persons := validFamily()
	persons[0].Data["birth_date"] = "1990" // yyyy only — valid, no date parse applied
	result := validate(persons, nil)
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "invalid date") {
			t.Errorf("unexpected invalid date error for year-only value: %v", e)
		}
	}
}

// ── Date format warnings ───────────────────────────────────────────────────────

func TestValidate_NonStandardDateFormat_Warning(t *testing.T) {
	persons := validFamily()
	persons[0].Data["birth_date"] = "15.06.1990"
	result := validate(persons, nil)
	if !hasWarningFor(result, "parent", "non-standard value") {
		t.Error("expected non-standard date format warning")
	}
}

func TestValidate_StandardDateFormats_NoWarning(t *testing.T) {
	persons := validFamily()
	persons[0].Data["birth_date"] = "1990-06-15"
	persons[1].Data["birth_date"] = "1992"
	result := validate(persons, nil)
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "non-standard") {
			t.Errorf("unexpected non-standard date warning: %v", w)
		}
	}
}

// ── Rare fields ───────────────────────────────────────────────────────────────

func TestValidate_RareField_Warning(t *testing.T) {
	// 5 persons, only one uses "occupation_typo" — should warn.
	persons := make([]Person, 5)
	for i := range persons {
		persons[i] = Person{
			ID:   fmt.Sprintf("p%d", i),
			Data: PersonData{"gender": "M", "first_name": "X"},
			Rels: PersonRels{Spouses: []string{}},
		}
	}
	// Give persons some relations so they're not "isolated"
	persons[0].Rels.Spouses = []string{"p1"}
	persons[1].Rels.Spouses = []string{"p0"}
	persons[2].Rels.Spouses = []string{"p3"}
	persons[3].Rels.Spouses = []string{"p2"}
	persons[4].Rels.Father = "p0"
	persons[0].Rels.Children = []string{"p4"}
	persons[1].Rels.Children = []string{"p4"}
	persons[4].Rels.Mother = "p1"

	persons[2].Data["occupaton"] = "farmer" // typo: occupaton vs occupation
	result := validate(persons, nil)
	if !hasWarningFor(result, "p2", "possible typo") {
		t.Error("expected rare-field typo warning for p2")
	}
}

func TestValidate_CommonField_NoRareWarning(t *testing.T) {
	persons := make([]Person, 5)
	for i := range persons {
		persons[i] = Person{
			ID:   fmt.Sprintf("p%d", i),
			Data: PersonData{"gender": "M", "first_name": "X", "occupation": "farmer"},
			Rels: PersonRels{Spouses: []string{}},
		}
	}
	persons[0].Rels.Spouses = []string{"p1"}
	persons[1].Rels.Spouses = []string{"p0"}
	persons[2].Rels.Spouses = []string{"p3"}
	persons[3].Rels.Spouses = []string{"p2"}
	persons[4].Rels.Father = "p0"
	persons[0].Rels.Children = []string{"p4"}
	persons[1].Rels.Children = []string{"p4"}
	persons[4].Rels.Mother = "p1"

	result := validate(persons, nil)
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "possible typo") && strings.Contains(w.Message, "occupation") {
			t.Errorf("unexpected typo warning for common field: %v", w)
		}
	}
}

// ── Clean file ────────────────────────────────────────────────────────────────

func TestValidate_CleanFile(t *testing.T) {
	persons := validFamily()
	result := validate(persons, nil)
	noErrors(t, result)
	noWarnings(t, result)
}
