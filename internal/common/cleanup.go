package common

// use 'enum' (combine const and typedef)
type RemoveType string

const (
	RemoveTypeWd RemoveType = "wd"
)

var PathsToClean = map[RemoveType][]string{}

func AddPathToClean(t RemoveType, path string) {
	PathsToClean[t] = append(PathsToClean[t], path)
}
