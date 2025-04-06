package phase

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"reflect"
	"testing"
)

func TestDependencyDescriptorMerging(t *testing.T) {

	dep := common.Dependency{
		Name:           "Django",
		NormalizedName: "django",
		Version:        "3.2.17",
		PackageManager: mappings.PythonManager,
		DiskPath:       "abc",
	}

	vulnerable := api.PackageVersion{
		Version:                         "3.2.17",
		Library:                         api.Package{NormalizedName: dep.NormalizedName, Name: dep.Name, PackageManager: dep.PackageManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "3.2.17+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-27351"},
			{CVE: "CVE-2023-46695"},
			{CVE: "CVE-2023-43665"},
		},
	}
	vulnerableWithoutFix := api.PackageVersion{
		Version:                         "1.2.1",
		Library:                         api.Package{NormalizedName: dep.NormalizedName, Name: dep.Name, PackageManager: dep.PackageManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.1+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-27351"},
		},
	}

	fixed := api.PackageVersion{
		Version:             "3.2.17+sp1",
		Library:             vulnerable.Library,
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-27351"},
			{CVE: "CVE-2023-46695"},
			{CVE: "CVE-2023-43665"},
		},
		OriginVersionString: vulnerable.Version,
		OriginVersionId:     vulnerable.VersionId,
	}

	res := ScanResult{
		Vulnerable: []api.PackageVersion{
			vulnerable,
			vulnerableWithoutFix,
		},
		AllDependencies: common.DependencyMap{dep.Id(): []*common.Dependency{&dep}},
	}

	for _, ovride := range []shared.OverriddenMethod{
		shared.NotOverridden,
		shared.OverriddenFromLocal,
		shared.OverriddenFromRemote,
	} {

		t.Run(fmt.Sprintf("test_override_%s", ovride), func(t *testing.T) {

			descs, err := buildDescriptorsForFixes(res, []api.PackageVersion{fixed}, ovride, actions.FilesManager)

			if err != nil || descs == nil {
				t.Fatalf("got error %v - %v", err, descs)
			}

			if len(descs) != 1 {
				t.Fatalf("got %d descriptors", len(descs))
			}

			dsc := descs[0]

			if !reflect.DeepEqual(fixed, *dsc.AvailableFix) {
				t.Fatalf("got wrong fixed package %v", dsc.AvailableFix)
			}

			if dsc.OverrideMethod != ovride {
				t.Fatalf("got wrong override method %v", ovride)
			}

			if !reflect.DeepEqual(vulnerable, *dsc.VulnerablePackage) {
				t.Fatalf("got wrong vulnerable package %v", dsc.VulnerablePackage)
			}

			if !reflect.DeepEqual(map[string]common.Dependency{dep.DiskPath: dep}, dsc.Locations) {
				t.Fatalf("got wrong locations map %v", dsc.Locations)
			}
		})
	}

}

func TestRemoteOverrideQuerySanity(t *testing.T) {
	djangoVulnerable := api.PackageVersion{
		Version:                     "3.2.17",
		Library:                     api.Package{NormalizedName: "django", Name: "django", PackageManager: mappings.PythonManager},
		OriginVersionId:             "111",
		RecommendedLibraryVersionId: "222",
	}

	queries := buildRemoteOverrideQuery([]api.PackageVersion{djangoVulnerable})
	if len(queries) != 1 {
		t.Fatalf("bad number of queries %d", len(queries))
	}

	q := queries[0]
	if q.LibraryId != djangoVulnerable.Library.Id {
		t.Fatalf("got bad library id: %s", q.LibraryId)
	}

	if q.OriginVersionId != djangoVulnerable.OriginVersionId {
		t.Fatalf("got bad origin id: %s", q.OriginVersionId)
	}

	if q.RecommendedVersionId == nil {
		t.Fatalf("got nil recommended id")
	}

	if *q.RecommendedVersionId != djangoVulnerable.RecommendedLibraryVersionId {
		t.Fatalf("got bad recommended id: %s", *q.RecommendedVersionId)
	}
}

func TestRemoteOverrideQueryNoRecommended(t *testing.T) {
	djangoVulnerable := api.PackageVersion{
		Version:                     "3.2.17",
		Library:                     api.Package{NormalizedName: "django", Name: "django", PackageManager: mappings.PythonManager},
		OriginVersionId:             "111",
		RecommendedLibraryVersionId: "",
	}

	queries := buildRemoteOverrideQuery([]api.PackageVersion{djangoVulnerable})
	if len(queries) != 0 {
		t.Fatalf("shoud not have queries %v", queries)
	}
}

func TestPopulateDescriptorsWithDifferentArch(t *testing.T) {
	//perp
	vulnerableWith2Arch := api.PackageVersion{
		Version: "1.1.11",
		Library: api.Package{NormalizedName: "pcre", Name: "pcre", PackageManager: mappings.RpmEcosystem},
	}

	vulnerableWith1Arch := api.PackageVersion{
		Version: "3.2.17",
		Library: api.Package{NormalizedName: "gzip", Name: "gzip", PackageManager: mappings.RpmEcosystem},
	}

	depWith64Arch := common.Dependency{
		Name:           "pcre",
		NormalizedName: "pcre",
		Version:        "1.1.11",
		PackageManager: mappings.RpmEcosystem,
		Arch:           "x86_64",
	}

	depWith32Arch := common.Dependency{
		Name:           "pcre",
		NormalizedName: "pcre",
		Version:        "1.1.11",
		PackageManager: mappings.RpmEcosystem,
		Arch:           "i686",
	}

	scanResult := ScanResult{
		Vulnerable: []api.PackageVersion{
			vulnerableWith2Arch,
			vulnerableWith1Arch,
		},
		AllDependencies: common.DependencyMap{depWith64Arch.Id(): []*common.Dependency{&depWith64Arch, &depWith32Arch}},
	}
	descs := make(map[string][]*shared.DependencyDescriptor)

	// function call
	populateDescriptorsWithDifferentArch(scanResult, vulnerableWith2Arch, descs)

	// validations
	descsArray := descs[depWith64Arch.Id()]
	if len(descsArray) != 2 {
		t.Fatalf("shoud have 2 sperated descriptors in the populated desc when in fact we have %v", len(descsArray))
	}

	if len(descsArray[0].Locations) != 1 {
		t.Fatalf("shoud have 1 location in the first populated desc when in fact we have %v", len(descsArray[0].Locations))
	}
	if len(descsArray[1].Locations) != 1 {
		t.Fatalf("shoud have 1 location in the second populated desc when in fact we have %v", len(descsArray[1].Locations))
	}
	if descsArray[0].Locations[""].Arch != "x86_64" {
		t.Fatalf("x86_64 arch is not included in the descriptors locations")
	}
	if descsArray[1].Locations[""].Arch != "i686" {
		t.Fatalf("i686 arch is not included in the descriptors locations")
	}
}

func TestPopulateDescriptorsWithDifferentLocation(t *testing.T) {
	//perp
	vulnerableWith2Locations := api.PackageVersion{
		Version: "1.1.11",
		Library: api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
	}

	vulnerableWith1Location := api.PackageVersion{
		Version: "3.2.17",
		Library: api.Package{NormalizedName: "axios", Name: "axios", PackageManager: mappings.NpmManager},
	}

	depFromLocation1 := common.Dependency{
		Name:           "smol-toml",
		NormalizedName: "smol-toml",
		Version:        "1.1.11",
		PackageManager: mappings.NpmManager,
		Arch:           "i686",
		DiskPath:       "/aaa/aaa",
	}

	depFromLocation2 := common.Dependency{
		Name:           "smol-toml",
		NormalizedName: "smol-toml",
		Version:        "1.1.11",
		PackageManager: mappings.NpmManager,
		Arch:           "i686",
		DiskPath:       "/bbb/bbb",
	}

	scanResult := ScanResult{
		Vulnerable: []api.PackageVersion{
			vulnerableWith2Locations,
			vulnerableWith1Location,
		},
		AllDependencies: common.DependencyMap{depFromLocation1.Id(): []*common.Dependency{&depFromLocation1, &depFromLocation2}},
	}
	descs := make(map[string][]*shared.DependencyDescriptor)

	// function call
	populateDescriptorsWithDifferentLocation(scanResult, vulnerableWith2Locations, descs)

	// validations
	descsArray := descs[vulnerableWith2Locations.Id()]
	if len(descsArray) != 1 {
		t.Fatalf("shoud have 1 collective descriptor in the populated desc when in fact we have %v", len(descsArray))
	}
	if len(descsArray[0].Locations) != 2 {
		t.Fatalf("shoud have 2 locationss in the populated desc when in fact we have %v", len(descsArray))
	}
	if _, firstLocationForDepExists := descsArray[0].Locations["/aaa/aaa"]; !firstLocationForDepExists {
		t.Fatalf("shoud have first location for dep")
	}
	if _, secondLocationForDep := descsArray[0].Locations["/bbb/bbb"]; !secondLocationForDep {
		t.Fatalf("shoud have second location for dep")
	}

}
