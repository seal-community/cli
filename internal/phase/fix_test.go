package phase

import (
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
		Version:        "3.2.17",
		PackageManager: mappings.PythonManager,
		DiskPath:       "abc",
	}

	vulnerable := api.PackageVersion{
		Version:                         "3.2.17",
		Library:                         api.Package{Name: dep.Name, PackageManager: dep.PackageManager},
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
		Library:                         api.Package{Name: dep.Name, PackageManager: dep.PackageManager},
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

			descs, err := buildDescriptorsForFixes(res, []api.PackageVersion{fixed}, ovride)

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
		Library:                     api.Package{Name: "django", PackageManager: mappings.PythonManager},
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
		Library:                     api.Package{Name: "django", PackageManager: mappings.PythonManager},
		OriginVersionId:             "111",
		RecommendedLibraryVersionId: "",
	}

	queries := buildRemoteOverrideQuery([]api.PackageVersion{djangoVulnerable})
	if len(queries) != 0 {
		t.Fatalf("shoud not have queries %v", queries)
	}
}
