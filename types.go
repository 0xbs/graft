package main

// PersonData holds all personal information fields for a person.
type PersonData struct {
	Gender         string `json:"gender"`
	NickName       string `json:"nick_name"`
	FirstName      string `json:"first_name"`
	SecondNames    string `json:"second_names"`
	FamilyName     string `json:"family_name"`
	BirthName      string `json:"birth_name"`
	BirthDate      string `json:"birth_date"`
	BirthPlace     string `json:"birth_place"`
	ResidencePlace string `json:"residence_place"`
	DeathDate      string `json:"death_date"`
	DeathPlace     string `json:"death_place"`
	BurialPlace    string `json:"burial_place"`
	MarriageDate   string `json:"marriage_date"`
	MarriagePlace  string `json:"marriage_place"`
	DivorceDate    string `json:"divorce_date"`
	AvatarURL      string `json:"avatar_url"`
	Note           string `json:"note"`
}

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
