package main

// PersonData holds all personal information fields for a person as a dynamic
// map, matching the family-chart format where only id, rels, and data.gender
// are fixed and all other data fields are optional and user-defined.
type PersonData map[string]string

// PersonRels holds family relationship references for a person.
type PersonRels struct {
	Father   string   `json:"father,omitempty"`
	Mother   string   `json:"mother,omitempty"`
	Children []string `json:"children,omitempty"`
	Spouses  []string `json:"spouses,omitempty"`
}

// Person is the top-level entity in the family tree JSON array.
type Person struct {
	ID   string     `json:"id"`
	Data PersonData `json:"data"`
	Rels PersonRels `json:"rels"`
}

// Conflict captures a disagreement between mine and theirs for a given field.
type Conflict struct {
	PersonID string
	Field    string
	Mine     string
	Theirs   string
}
