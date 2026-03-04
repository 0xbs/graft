# graft — AI Coding Reference

## About this project

graft is a standard go mod project for command line usage for merging two family tree JSON files.
It has an interactive terminal mode using charmbracelet's bubbletea and lipgloss libs in file `tui.go`.

## Mandatory duties

* Always run a build (`go build`).
* Always run tests (`go test`).
* Keep the tests up-to-date and add new tests for new features.
* Keep the README.md file up-to-date.

## Data format

Data is given in JSON files in an array containing persons, see `example.json`.
Each person has the following data format:

```json5
{
   "id": "79a1361d-4311-4686-8d9d-c34e410d81d2", // required UUID
   "data": {
      "gender": "F" // required with either "M" for male or "F" for female
      // all other fields are optional
   },
   "rels": {
      // all relations are optional
      "father": "bf97ed0a-2d5f-43df-ab23-6b94809476fa",
      "mother": "b4f9b84f-ecb4-4c66-aaf6-00439e4c613e",
      "children": [
         "a082674d-d96e-4daa-b428-b970e007857a"
      ],
      "spouses": [
         "67197b4b-1e86-4465-8b41-dbb1e20d40e0"
      ]
   }
}
```
