package output

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"sort"
)

type summaryFix struct {
	dep       shared.DependnecyDescriptor
	locations []string
}

type Summary struct {
	Root  string       `json:"root"`
	Fixes []summaryFix `json:"fixes"`
}

func NewSummary(projectDir string, fixes []shared.DependnecyDescriptor) *Summary {
	s := &Summary{Root: projectDir,
		Fixes: make([]summaryFix, 0, 10), // allocate, so if empty in json will be [] instead of null
	}

	for _, entry := range fixes {
		if len(entry.FixedLocations) == 0 || entry.VulnerablePackage == nil {
			slog.Error("bad entry in fix map", "len", len(entry.Locations), "package", entry.VulnerablePackage)
			continue
		}

		paths := entry.FixedLocations
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

		s.Fixes = append(s.Fixes, summaryFix{
			dep:       entry,
			locations: relativePaths,
		})
	}

	// sort results based on library name; sorting here to keep order of input
	sort.Slice(s.Fixes, func(i, j int) bool {
		return s.Fixes[i].dep.VulnerablePackage.Library.Name < s.Fixes[j].dep.VulnerablePackage.Library.Name
	})

	return s
}

func (f *summaryFix) MarshalJSON() ([]byte, error) {
	// can't marshal self since will cause infinite recursion
	return json.Marshal(&struct {
		From  string   `json:"from"`
		To    string   `json:"to"`
		Paths []string `json:"locations"`
	}{
		From:  f.dep.VulnerablePackage.Descriptor(),
		To:    f.dep.AvailableFix.Descriptor(),
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
		overrideMsg := ""
		if f.dep.IsOverridden() {
			switch f.dep.OverrideMethod {
			case shared.OverriddenFromLocal:
				overrideMsg = common.Colorize(" (actions file)", common.AnsiDarkGrey)
			case shared.OverriddenFromRemote:
				overrideMsg = common.Colorize(" (remote config)", common.AnsiDarkGrey)
			}
		}

		fmt.Printf("%s replaced with %s%s\n",
			common.Colorize(f.dep.VulnerablePackage.Descriptor(), common.AnsiColdPurple),
			common.Colorize(f.dep.AvailableFix.Descriptor(), common.AnsiBlue),
			overrideMsg,
		)

		for _, path := range f.locations {
			fmt.Printf("%s%s\n", prefix, common.Colorize(path, common.AnsiDarkGrey))
		}

		fmt.Println()
	}
}
