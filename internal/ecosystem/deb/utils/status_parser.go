package utils

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
)

type StatusFilePackage struct {
	Package       string
	Protected     string
	Essential     string
	Status        string
	Priority      string
	Section       string
	InstalledSize int64
	Maintainer    string
	Architecture  string
	MultiArch     string
	Source        string
	Version       string
	Provides      string
	Replaces      string
	Depends       string
	PreDepends    string
	Recommends    string
	Suggests      string
	Breaks        string
	Enhances      string
	Conflicts     string
	Conffiles     string
	Description   string
	Homepage      string
	Important     string
}

type Parser struct {
	r *bufio.Reader
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		r: bufio.NewReader(r),
	}
}

func (p *Parser) parseLine(line string) (string, string) {
	line = strings.TrimRight(line, "\n")

	if len(line) == 0 {
		return "", ""
	}

	if line[0] == ' ' {
		return "", line
	}

	separatorIndex := strings.Index(line, ":")
	key := line[0:separatorIndex]
	value := line[separatorIndex+1:]

	return key, value
}

func (p *Parser) mapToPackage(m map[string]string) StatusFilePackage {
	pkg := StatusFilePackage{}

	for key, value := range m {
		value = strings.TrimRight(value, " \n")
		value = strings.TrimLeft(value, " ")

		switch key {
		case "Package":
			pkg.Package = value
		case "Protected":
			pkg.Protected = value
		case "Essential":
			pkg.Essential = value
		case "Status":
			pkg.Status = value
		case "Priority":
			pkg.Priority = value
		case "Section":
			pkg.Section = value
		case "Installed-Size":
			i, err := strconv.Atoi(value)
			if err == nil {
				pkg.InstalledSize = int64(i)
			}
		case "Maintainer":
			pkg.Maintainer = value
		case "Architecture":
			pkg.Architecture = value
		case "Multi-Arch":
			pkg.MultiArch = value
		case "Source":
			pkg.Source = value
		case "Version":
			pkg.Version = value
		case "Provides":
			pkg.Provides = value
		case "Replaces":
			pkg.Replaces = value
		case "Depends":
			pkg.Depends = value
		case "Pre-Depends":
			pkg.PreDepends = value
		case "Recommends":
			pkg.Recommends = value
		case "Suggests":
			pkg.Suggests = value
		case "Breaks":
			pkg.Breaks = value
		case "Enhances":
			pkg.Enhances = value
		case "Conflicts":
			pkg.Conflicts = value
		case "Conffiles":
			pkg.Conffiles = value
		case "Description":
			pkg.Description = value
		case "Homepage":
			pkg.Homepage = value
		case "Important":
			pkg.Important = value
		}
	}

	return pkg
}

func (p *Parser) Parse() ([]StatusFilePackage, error) {
	prevKey := ""
	packages := []StatusFilePackage{}
	m := make(map[string]string)

	for {
		line, readError := p.r.ReadString('\n')
		if readError != nil {
			if readError == io.EOF {
				if line == "" {
					// EOF and actual content can exist on the same line so we wont auto return on EOF error
					packages = append(packages, p.mapToPackage(m))
					return packages, nil
				}
			} else {
				slog.Error("Failed reading contents of status file", "err", readError)
				return []StatusFilePackage{}, readError
			}
		}
		key, value := p.parseLine(line)

		if key == "" && value != "" {
			m[prevKey] = m[prevKey] + "\n" + strings.TrimLeft(value, " ")
		} else if key == "" && value == "" {
			if len(m) > 0 {
				pkg := p.mapToPackage(m)
				packages = append(packages, pkg)
				m = make(map[string]string)
			}
		} else if key != "" {
			prevKey = key
			m[key] = value
		}

		if readError == io.EOF {
			packages = append(packages, p.mapToPackage(m))
			return packages, nil
		}
	}
}

func dumpMultilineField(value string) string {
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		return "\n"
	}

	return lines[0] + "\n " + strings.Join(lines[1:], "\n ")
}

func DumpPackages(pkgs []StatusFilePackage) string {
	var sb strings.Builder
	for _, pkg := range pkgs {
		// Only write fields if they are non-empty, to mirror actual dpkg behavior.
		if pkg.Package != "" {
			fmt.Fprintf(&sb, "Package: %s\n", pkg.Package)
		}
		if pkg.Protected != "" {
			fmt.Fprintf(&sb, "Protected: %s\n", pkg.Protected)
		}
		if pkg.Essential != "" {
			fmt.Fprintf(&sb, "Essential: %s\n", pkg.Essential)
		}
		if pkg.Status != "" {
			fmt.Fprintf(&sb, "Status: %s\n", pkg.Status)
		}
		if pkg.Priority != "" {
			fmt.Fprintf(&sb, "Priority: %s\n", pkg.Priority)
		}
		if pkg.Section != "" {
			fmt.Fprintf(&sb, "Section: %s\n", pkg.Section)
		}
		if pkg.InstalledSize > 0 {
			fmt.Fprintf(&sb, "Installed-Size: %d\n", pkg.InstalledSize)
		}
		if pkg.Maintainer != "" {
			fmt.Fprintf(&sb, "Maintainer: %s\n", pkg.Maintainer)
		}
		if pkg.Architecture != "" {
			fmt.Fprintf(&sb, "Architecture: %s\n", pkg.Architecture)
		}
		if pkg.MultiArch != "" {
			fmt.Fprintf(&sb, "Multi-Arch: %s\n", pkg.MultiArch)
		}
		if pkg.Source != "" {
			fmt.Fprintf(&sb, "Source: %s\n", pkg.Source)
		}
		if pkg.Version != "" {
			fmt.Fprintf(&sb, "Version: %s\n", pkg.Version)
		}
		if pkg.Provides != "" {
			fmt.Fprintf(&sb, "Provides: %s\n", pkg.Provides)
		}
		if pkg.Replaces != "" {
			fmt.Fprintf(&sb, "Replaces: %s\n", pkg.Replaces)
		}
		if pkg.Depends != "" {
			fmt.Fprintf(&sb, "Depends: %s\n", pkg.Depends)
		}
		if pkg.PreDepends != "" {
			fmt.Fprintf(&sb, "Pre-Depends: %s\n", pkg.PreDepends)
		}
		if pkg.Recommends != "" {
			fmt.Fprintf(&sb, "Recommends: %s\n", pkg.Recommends)
		}
		if pkg.Suggests != "" {
			fmt.Fprintf(&sb, "Suggests: %s\n", pkg.Suggests)
		}
		if pkg.Breaks != "" {
			fmt.Fprintf(&sb, "Breaks: %s\n", pkg.Breaks)
		}
		if pkg.Enhances != "" {
			fmt.Fprintf(&sb, "Enhances: %s\n", pkg.Enhances)
		}
		if pkg.Conflicts != "" {
			fmt.Fprintf(&sb, "Conflicts: %s\n", pkg.Conflicts)
		}
		if pkg.Conffiles != "" {
			fmt.Fprintf(&sb, "Conffiles:%s\n", dumpMultilineField(pkg.Conffiles))
		}
		if pkg.Description != "" {
			fmt.Fprintf(&sb, "Description: %s\n", dumpMultilineField(pkg.Description))
		}
		if pkg.Homepage != "" {
			fmt.Fprintf(&sb, "Homepage: %s\n", pkg.Homepage)
		}
		if pkg.Important != "" {
			fmt.Fprintf(&sb, "Important: %s\n", pkg.Important)
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
