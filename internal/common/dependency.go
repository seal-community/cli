package common

import "fmt"

type Dependency struct {
	Name           string      `json:"library_name"`
	Version        string      `json:"library_version"`
	PackageManager string      `json:"library_package_manager"`
	DiskPath       string      `json:"-"` // local disk path for the package
	NameAlias      string      `json:"-"` // currently npm-specific, name that can be used to reference this dependency
	Parent         *Dependency `json:"-"`
	Extraneous     bool        `json:"-"` // currently npm-specific
	Branch         string      `json:"-"` // chain of dependencies that reached this
	Dev            bool        `json:"-"` // currently useful for pnpm since npm handles it implicitly
	Link           bool        `json:"-"` // currently useful for pnpm - is a link to another place
}

func (d *Dependency) HasAlias() bool {
	// might require further testing and research
	// other deps might rely on alias and we might break them in private-sps scenario
	return d.Name != d.NameAlias
}

func (d *Dependency) IsDirect() bool {
	return d.Parent == nil
}

func (d *Dependency) IsTransitive() bool {
	return !d.IsDirect()
}

func (d *Dependency) Id() string {
	return DependencyId(d.PackageManager, d.Name, d.Version)
}

func (d *Dependency) PrintableName() string {
	switch d.PackageManager {
	default:
		return fmt.Sprintf("%s:%s@%s", d.PackageManager, d.Name, d.Version)
	}
}

func DependencyId(manager string, library string, version string) string {
	if manager == "" || library == "" || version == "" {
		panic(fmt.Errorf("failed: cant generate id for library:%s version:%s", library, version))
	}
	
	return fmt.Sprintf("%s|%s@%s", manager, library, version)
}

// paths might be the same in multiple dependencies in the tree, and overwriting them can cause conflict
type DependencyMap map[string][]*Dependency
