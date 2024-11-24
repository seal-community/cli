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
	dep       shared.DependencyDescriptor
	locations []string
}

type summarySilence struct {
	descriptor string
	locations  []string
}

type Summary struct {
	Root     string           `json:"root"`
	Fixes    []summaryFix     `json:"fixes"`
	Silenced []summarySilence `json:"silences"`
}

func getRelativePaths(root string, paths []string) ([]string, error) {
	relativePaths := make([]string, 0, len(paths))
	for _, origPath := range paths {
		path := origPath
		if filepath.IsAbs(path) {
			var err error
			path, err = filepath.Rel(root, origPath)
			if err != nil {
				// should not really happen
				slog.Error("failed converting to relative path", "err", err, "path", origPath)
				return nil, err
			}

			common.Trace("converted path to relative", "rel", path, "original", origPath, "root", root)
		}

		relativePaths = append(relativePaths, path)
	}
	return relativePaths, nil
}

func NewSummary(projectDir string, fixes []shared.DependencyDescriptor, silenced map[string][]string) *Summary {
	s := &Summary{Root: projectDir,
		Fixes: make([]summaryFix, 0, 10), // allocate, so if empty in json will be [] instead of null
	}

	for _, entry := range fixes {
		if len(entry.FixedLocations) == 0 || entry.VulnerablePackage == nil {
			slog.Error("bad entry in fix map", "len", len(entry.Locations), "package", entry.VulnerablePackage)
			continue
		}

		paths := entry.FixedLocations

		slices.Sort(paths)

		relativePaths, err := getRelativePaths(s.Root, paths)
		if err != nil {
			return nil
		}

		s.Fixes = append(s.Fixes, summaryFix{
			dep:       entry,
			locations: relativePaths,
		})

	}

	for silencedPackageId, paths := range silenced {
		relativePaths, err := getRelativePaths(s.Root, paths)
		if err != nil {
			return nil
		}

		s.Silenced = append(s.Silenced, summarySilence{
			descriptor: silencedPackageId,
			locations:  relativePaths,
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

func getRemediatedVulnerabilityIds(f summaryFix) []string {
	var remediatedCVEs []string
	if f.dep.AvailableFix == nil || f.dep.VulnerablePackage == nil {
		slog.Warn("Dependency has no available fix or vulnerable package even though it is marked as a fix.")
		return remediatedCVEs
	}

	sealedCVEs := make(map[string]bool)
	for _, item := range f.dep.AvailableFix.SealedVulnerabilities {
		sealedCVEs[item.PreferredId()] = true
	}

	for _, item := range f.dep.VulnerablePackage.OpenVulnerabilities {
		if sealedCVEs[item.PreferredId()] {
			remediatedCVEs = append(remediatedCVEs, item.PreferredId())
		}
	}

	return remediatedCVEs
}

func (s *Summary) Print(finalMsg string) {
	// if we change the response model / add additional request for each package we could also print the sealed vulnerabilities of the fixed version
	slog.Info("fixed packages", "count", len(s.Fixes))
	slog.Info("silenced packages", "count", len(s.Silenced))

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

		for _, cve := range getRemediatedVulnerabilityIds(f) {
			fmt.Printf("%s remediated\n", cve)
		}

		fmt.Println()
	}

	for _, s := range s.Silenced {
		fmt.Printf("%s was silenced\n", common.Colorize(s.descriptor, common.AnsiColdPurple))
		for _, path := range s.locations {
			fmt.Printf("%s%s\n", prefix, common.Colorize(path, common.AnsiDarkGrey))
		}

		fmt.Println()
	}

	if finalMsg != "" {
		// allow overriding summary message
		fmt.Println(finalMsg)
		return
	}

	var msg string
	fixed := len(s.Fixes)
	switch fixed {
	case 0:
		msg = "Nothing to fix"
	case 1:
		msg = "Fixed 1 package"
	default:
		msg = fmt.Sprintf("Fixed %d packages", fixed)
	}

	fmt.Println(msg)
}
