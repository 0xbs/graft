package main

import (
	"testing"
)

// helpers

func personIDs(persons []Person) map[string]bool {
	ids := make(map[string]bool, len(persons))
	for _, p := range persons {
		ids[p.ID] = true
	}
	return ids
}

func findPerson(persons []Person, id string) Person {
	for _, p := range persons {
		if p.ID == id {
			return p
		}
	}
	return Person{}
}

// A chain: A --(child)--> B --(child)--> C
var chainPersons = []Person{
	{ID: "A", Rels: PersonRels{Children: []string{"B"}}},
	{ID: "B", Rels: PersonRels{Father: "A", Children: []string{"C"}}},
	{ID: "C", Rels: PersonRels{Father: "B"}},
}

func TestSubtreeFullChain(t *testing.T) {
	result, err := subtree(chainPersons, "A", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 persons, got %d", len(result))
	}
}

func TestSubtreeStopAtB(t *testing.T) {
	stop := map[string]bool{"B": true}
	result, err := subtree(chainPersons, "A", stop)
	if err != nil {
		t.Fatal(err)
	}
	ids := personIDs(result)
	if !ids["A"] || !ids["B"] {
		t.Fatal("A and B must be included")
	}
	if ids["C"] {
		t.Fatal("C must not be included when stopping at B")
	}
	// B's rels should be filtered: father=A kept, children=C removed
	b := findPerson(result, "B")
	if b.Rels.Father != "A" {
		t.Errorf("B.Father should be A, got %q", b.Rels.Father)
	}
	if len(b.Rels.Children) != 0 {
		t.Errorf("B.Children should be empty, got %v", b.Rels.Children)
	}
}

func TestSubtreeWithSpouses(t *testing.T) {
	persons := []Person{
		{ID: "A", Rels: PersonRels{Spouses: []string{"B"}}},
		{ID: "B", Rels: PersonRels{Spouses: []string{"A"}, Children: []string{"C"}}},
		{ID: "C", Rels: PersonRels{Mother: "B"}},
	}
	result, err := subtree(persons, "A", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 persons, got %d", len(result))
	}
}

func TestSubtreeStopAtSpouse(t *testing.T) {
	persons := []Person{
		{ID: "A", Rels: PersonRels{Spouses: []string{"B"}}},
		{ID: "B", Rels: PersonRels{Spouses: []string{"A"}, Children: []string{"C"}}},
		{ID: "C", Rels: PersonRels{Mother: "B"}},
	}
	stop := map[string]bool{"B": true}
	result, err := subtree(persons, "A", stop)
	if err != nil {
		t.Fatal(err)
	}
	ids := personIDs(result)
	if !ids["A"] || !ids["B"] {
		t.Fatal("A and B must be included")
	}
	if ids["C"] {
		t.Fatal("C must not be included when stopping at B")
	}
}

func TestSubtreeInvalidFromID(t *testing.T) {
	_, err := subtree(chainPersons, "UNKNOWN", nil)
	if err == nil {
		t.Fatal("expected error for unknown from-ID")
	}
}

func TestSubtreeInvalidStopID(t *testing.T) {
	stop := map[string]bool{"UNKNOWN": true}
	_, err := subtree(chainPersons, "A", stop)
	if err == nil {
		t.Fatal("expected error for unknown stop-ID")
	}
}

func TestSubtreeRelsFiltered(t *testing.T) {
	// D is connected to A but not reachable from C without going through B
	persons := []Person{
		{ID: "A", Rels: PersonRels{Children: []string{"B"}, Spouses: []string{"D"}}},
		{ID: "B", Rels: PersonRels{Father: "A", Children: []string{"C"}}},
		{ID: "C", Rels: PersonRels{Father: "B"}},
		{ID: "D", Rels: PersonRels{Spouses: []string{"A"}}},
	}
	stop := map[string]bool{"A": true}
	result, err := subtree(persons, "C", stop)
	if err != nil {
		t.Fatal(err)
	}
	ids := personIDs(result)
	if !ids["C"] || !ids["B"] || !ids["A"] {
		t.Fatal("C, B, A must be included")
	}
	if ids["D"] {
		t.Fatal("D must not be included when stopping at A")
	}
	// A's rels should have D filtered out
	a := findPerson(result, "A")
	if len(a.Rels.Spouses) != 0 {
		t.Errorf("A.Spouses should be empty (D excluded), got %v", a.Rels.Spouses)
	}
	if len(a.Rels.Children) != 1 || a.Rels.Children[0] != "B" {
		t.Errorf("A.Children should be [B], got %v", a.Rels.Children)
	}
}

func TestSubtreePreservesOrder(t *testing.T) {
	result, err := subtree(chainPersons, "C", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Order should match the original slice order, not BFS order
	if result[0].ID != "A" || result[1].ID != "B" || result[2].ID != "C" {
		t.Errorf("expected order A,B,C but got %s,%s,%s", result[0].ID, result[1].ID, result[2].ID)
	}
}
