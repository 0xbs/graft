package main

import (
	"encoding/json"
	"os"
	"testing"
)

func loadTestJSON(t *testing.T, path string) []Person {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var persons []Person
	if err := json.Unmarshal(data, &persons); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
	return persons
}

func TestMergeRules(t *testing.T) {
	mine := loadTestJSON(t, "testdata/mine.json")
	theirs := loadTestJSON(t, "testdata/theirs.json")
	merged, conflicts := merge(mine, theirs)

	// Rule 1: new person present only in theirs is appended.
	t.Run("new_person_appended", func(t *testing.T) {
		if len(merged) != 2 {
			t.Fatalf("expected 2 persons, got %d", len(merged))
		}
		last := merged[len(merged)-1]
		if last.ID != "aaaaaaaa-0000-0000-0000-000000000002" {
			t.Errorf("expected new person ID, got %s", last.ID)
		}
		if last.Data.FirstName != "Ben" {
			t.Errorf("expected new person first_name Ben, got %s", last.Data.FirstName)
		}
	})

	// Rule 2: empty field in mine is filled from theirs (no conflict).
	t.Run("empty_field_filled_from_theirs", func(t *testing.T) {
		if merged[0].Data.BirthDate != "1985-06-15" {
			t.Errorf("expected birth_date filled from theirs, got %q", merged[0].Data.BirthDate)
		}
		hasBirthDateConflict := false
		for _, c := range conflicts {
			if c.Field == "data.birth_date" {
				hasBirthDateConflict = true
			}
		}
		if hasBirthDateConflict {
			t.Error("expected no conflict for birth_date, but one was recorded")
		}
	})

	// Rule 3: non-empty field in mine is kept when theirs is empty (no conflict).
	// nick_name is "Anna" in mine, "" in theirs → must stay "Anna".
	t.Run("non_empty_mine_kept_when_theirs_empty", func(t *testing.T) {
		if merged[0].Data.NickName != "Anna" {
			t.Errorf("expected nick_name kept from mine, got %q", merged[0].Data.NickName)
		}
	})

	// Rule 4: equal fields in both produce no conflict.
	t.Run("equal_fields_no_conflict", func(t *testing.T) {
		for _, c := range conflicts {
			if c.Field == "data.note" {
				t.Error("unexpected conflict for equal note field")
			}
		}
		if merged[0].Data.Note != "Teacher" {
			t.Errorf("expected note Teacher, got %q", merged[0].Data.Note)
		}
	})

	// Rule 5: both non-empty and different → conflict, mine's value kept.
	// birth_place: mine="Berlin", theirs="Hamburg"
	t.Run("conflicting_field_mine_wins", func(t *testing.T) {
		if merged[0].Data.BirthPlace != "Berlin" {
			t.Errorf("expected mine's birth_place kept, got %q", merged[0].Data.BirthPlace)
		}
		var found *Conflict
		for i := range conflicts {
			if conflicts[i].Field == "data.birth_place" {
				found = &conflicts[i]
			}
		}
		if found == nil {
			t.Fatal("expected conflict for birth_place, got none")
		}
		if found.Mine != "Berlin" || found.Theirs != "Hamburg" {
			t.Errorf("conflict values wrong: mine=%q theirs=%q", found.Mine, found.Theirs)
		}
	})

	// Rule 6a: rels single-ref (father) — both non-empty and different → conflict, mine kept.
	t.Run("rels_single_ref_conflict", func(t *testing.T) {
		if merged[0].Rels.Father != "aaaaaaaa-0000-0000-0000-000000000010" {
			t.Errorf("expected mine's father kept, got %q", merged[0].Rels.Father)
		}
		var found *Conflict
		for i := range conflicts {
			if conflicts[i].Field == "rels.father" {
				found = &conflicts[i]
			}
		}
		if found == nil {
			t.Fatal("expected conflict for rels.father, got none")
		}
	})

	// Rule 6b: rels single-ref (mother) — mine empty, theirs non-empty → filled, no conflict.
	t.Run("rels_single_ref_filled_from_theirs", func(t *testing.T) {
		if merged[0].Rels.Mother != "aaaaaaaa-0000-0000-0000-000000000012" {
			t.Errorf("expected mother filled from theirs, got %q", merged[0].Rels.Mother)
		}
		for _, c := range conflicts {
			if c.Field == "rels.mother" {
				t.Error("unexpected conflict for rels.mother")
			}
		}
	})

	// Rule 7: children array is the union of mine and theirs (no duplicates).
	t.Run("rels_children_union", func(t *testing.T) {
		children := merged[0].Rels.Children
		if len(children) != 2 {
			t.Fatalf("expected 2 children, got %d: %v", len(children), children)
		}
		has := func(id string) bool {
			for _, c := range children {
				if c == id {
					return true
				}
			}
			return false
		}
		if !has("aaaaaaaa-0000-0000-0000-000000000030") {
			t.Error("missing original child")
		}
		if !has("aaaaaaaa-0000-0000-0000-000000000031") {
			t.Error("missing new child from theirs")
		}
	})

	// avatar_url: mine empty, theirs has value → conflict (not a silent fill).
	t.Run("avatar_url_empty_mine_is_conflict", func(t *testing.T) {
		if merged[0].Data.AvatarURL != "" {
			t.Errorf("expected avatar_url kept empty (mine), got %q", merged[0].Data.AvatarURL)
		}
		var found bool
		for _, c := range conflicts {
			if c.Field == "data.avatar_url" {
				found = true
				if c.Mine != "" || c.Theirs != "avatars/anna.jpg" {
					t.Errorf("conflict values wrong: mine=%q theirs=%q", c.Mine, c.Theirs)
				}
			}
		}
		if !found {
			t.Error("expected conflict for data.avatar_url, got none")
		}
	})

	// Rule 7: spouses array is the union of mine and theirs (no duplicates).
	t.Run("rels_spouses_union", func(t *testing.T) {
		spouses := merged[0].Rels.Spouses
		if len(spouses) != 2 {
			t.Fatalf("expected 2 spouses, got %d: %v", len(spouses), spouses)
		}
		has := func(id string) bool {
			for _, s := range spouses {
				if s == id {
					return true
				}
			}
			return false
		}
		if !has("aaaaaaaa-0000-0000-0000-000000000020") {
			t.Error("missing original spouse")
		}
		if !has("aaaaaaaa-0000-0000-0000-000000000021") {
			t.Error("missing new spouse from theirs")
		}
	})
}

// TestMergeEmptyInputs verifies that merging with empty slices is handled correctly.
func TestMergeEmptyInputs(t *testing.T) {
	person := Person{ID: "aaa", Data: PersonData{FirstName: "X"}}

	t.Run("empty_mine", func(t *testing.T) {
		merged, conflicts := merge(nil, []Person{person})
		if len(merged) != 1 || merged[0].ID != "aaa" {
			t.Errorf("expected theirs added to empty mine, got %v", merged)
		}
		if len(conflicts) != 0 {
			t.Errorf("expected no conflicts, got %v", conflicts)
		}
	})

	t.Run("empty_theirs", func(t *testing.T) {
		merged, conflicts := merge([]Person{person}, nil)
		if len(merged) != 1 || merged[0].ID != "aaa" {
			t.Errorf("expected mine unchanged, got %v", merged)
		}
		if len(conflicts) != 0 {
			t.Errorf("expected no conflicts, got %v", conflicts)
		}
	})
}

// TestAvatarURLAlwaysConflict verifies that avatar_url is never silently filled,
// even when mine is empty, because the file may not be available locally.
func TestAvatarURLAlwaysConflict(t *testing.T) {
	mine := []Person{{ID: "p1", Data: PersonData{FirstName: "X"}}}
	theirs := []Person{{ID: "p1", Data: PersonData{FirstName: "X", AvatarURL: "avatars/x.jpg"}}}
	merged, conflicts := merge(mine, theirs)

	if merged[0].Data.AvatarURL != "" {
		t.Errorf("expected avatar_url not filled (mine stays empty), got %q", merged[0].Data.AvatarURL)
	}
	if len(conflicts) != 1 || conflicts[0].Field != "data.avatar_url" {
		t.Errorf("expected exactly one avatar_url conflict, got %v", conflicts)
	}
}

// TestArrayUnionNoDuplicates ensures existing IDs in mine are not duplicated.
func TestArrayUnionNoDuplicates(t *testing.T) {
	mine := []Person{{
		ID:   "p1",
		Rels: PersonRels{Children: []string{"c1", "c2"}},
	}}
	theirs := []Person{{
		ID:   "p1",
		Rels: PersonRels{Children: []string{"c2", "c3"}},
	}}
	merged, _ := merge(mine, theirs)
	if len(merged[0].Rels.Children) != 3 {
		t.Errorf("expected 3 children (no duplicate c2), got %v", merged[0].Rels.Children)
	}
}
