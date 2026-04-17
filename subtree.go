package main

import "fmt"

// subtree extracts a connected subset of persons starting from fromID.
// It traverses all relationships (father, mother, children, spouses) via BFS.
// Persons whose ID is in stopIDs are included but their connections are not
// followed further. The returned persons have their rels filtered to only
// reference persons that are part of the result set.
func subtree(persons []Person, fromID string, stopIDs map[string]bool) ([]Person, error) {
	index := make(map[string]Person, len(persons))
	for _, p := range persons {
		index[p.ID] = p
	}

	if _, ok := index[fromID]; !ok {
		return nil, fmt.Errorf("from-ID not found: %s", fromID)
	}
	for id := range stopIDs {
		if _, ok := index[id]; !ok {
			return nil, fmt.Errorf("stop-ID not found: %s", id)
		}
	}

	visited := make(map[string]bool)
	queue := []string{fromID}
	visited[fromID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current != fromID && stopIDs[current] {
			continue
		}

		p := index[current]
		neighbors := connectedIDs(p)
		for _, nID := range neighbors {
			if visited[nID] {
				continue
			}
			if _, ok := index[nID]; !ok {
				continue
			}
			visited[nID] = true
			queue = append(queue, nID)
		}
	}

	result := make([]Person, 0, len(visited))
	for _, p := range persons {
		if visited[p.ID] {
			result = append(result, filterRels(p, visited))
		}
	}
	return result, nil
}

// connectedIDs returns all person IDs referenced in p's relationships.
func connectedIDs(p Person) []string {
	var ids []string
	if p.Rels.Father != "" {
		ids = append(ids, p.Rels.Father)
	}
	if p.Rels.Mother != "" {
		ids = append(ids, p.Rels.Mother)
	}
	ids = append(ids, p.Rels.Children...)
	ids = append(ids, p.Rels.Spouses...)
	return ids
}

// filterRels returns a copy of p with rels filtered to only include IDs
// present in the included set.
func filterRels(p Person, included map[string]bool) Person {
	var rels PersonRels
	if included[p.Rels.Father] {
		rels.Father = p.Rels.Father
	}
	if included[p.Rels.Mother] {
		rels.Mother = p.Rels.Mother
	}
	for _, id := range p.Rels.Children {
		if included[id] {
			rels.Children = append(rels.Children, id)
		}
	}
	for _, id := range p.Rels.Spouses {
		if included[id] {
			rels.Spouses = append(rels.Spouses, id)
		}
	}
	p.Rels = rels
	return p
}
