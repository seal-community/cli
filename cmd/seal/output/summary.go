package output

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/phase"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"

	"golang.org/x/exp/maps"
)

type summaryFix struct {
	pkg       *api.PackageVersion
	locations []string
}

type Summary struct {
	Root  string        `json:"root"`
	Fixes []*summaryFix `json:"fixes"`
}

func NewSummary(projectDir string, fixes phase.FixMap) *Summary {
	s := &Summary{Root: projectDir,
		Fixes: make([]*summaryFix, 0, 10), // allocate, so if empty in json will be [] instead of null
	}

	keys := maps.Keys(fixes)
	slices.Sort(keys)

	for _, k := range keys {
		entry := fixes[k]
		if len(entry.Paths) == 0 || entry.Package == nil {
			slog.Error("bad entry in fix map", "len", len(entry.Paths), "package", entry.Package)
			continue
		}

		paths := maps.Keys(entry.Paths)
		relativePaths := make([]string, 0, len(paths))

		slices.Sort(paths)
		for _, origPath := range paths {
			path := origPath
			if filepath.IsAbs(path) {
				var err error
				path, err = filepath.Rel(s.Root, origPath)
				if err != nil {
					// should not really happen
					slog.Error("failed converting to relative path", "err", err, "path", origPath)
					return nil
				}
				common.Trace("converted path to relaive", "rel", path, "original", origPath)
			}
			relativePaths = append(relativePaths, path)
		}

		sf := &summaryFix{
			pkg:       entry.Package,
			locations: relativePaths,
		}

		s.Fixes = append(s.Fixes, sf)
	}

	return s
}

func (f *summaryFix) MarshalJSON() ([]byte, error) {
	// can't marshal self since will cause infinite recursion
	return json.Marshal(&struct {
		From  string   `json:"from"`
		To    string   `json:"to"`
		Paths []string `json:"locations"`
	}{
		From:  f.pkg.Descriptor(),
		To:    f.pkg.RecommendedDescriptor(),
		Paths: f.locations,
	})
}

func (s *Summary) Save(w io.Writer) error {
	data, err := json.MarshalIndent(s, "", "  ") // pretty-format json using 2-space indents

	if err != nil {
		slog.Error("failed to marshal summary", "err", err)
		return err
	}

	_, err = w.Write(data)
	if err != nil {
		slog.Error("failed to write summary to file", "err", err)
		return err
	}

	return nil
}

func (s *Summary) Print() {
	// if we change the response model / add additional request for each package we could also print the sealed vulnerabilities of the fixed version
	const prefix = "   "
	for _, f := range s.Fixes {
		fmt.Printf("%s replaced with %s\n",
			common.Colorize(f.pkg.Descriptor(), common.AnsiColdPurple),
			common.Colorize(f.pkg.RecommendedDescriptor(), common.AnsiBlue))

		for _, path := range f.locations {
			fmt.Printf("%s%s\n", prefix, common.Colorize(path, common.AnsiDarkGrey))
		}

		fmt.Println()
	}
}
