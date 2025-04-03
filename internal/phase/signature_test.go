package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"testing"
)

func TestExtractMessage(t *testing.T) {
	expected := "G4Y1XxOn8LkMi2BTwCVDmZlN+7OEPgjWA+KSyhO49nLtXlh5HBDz422uyWmcwvvciLT+EW76f84BaTi3hwQ4GA=="
	res := extractMessage([]byte("aaaa"))
	if res != expected {
		t.Fatalf("got `%s` expected `%s`", res, expected)
	}
}

func TestCreateSignaturesQuery(t *testing.T) {
	pkg := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	fixPkg := api.PackageVersion{
		Version:   "4.17.11-sp1",
		VersionId: "111",
		Library:   api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
	}

	dep := shared.DependencyDescriptor{
		VulnerablePackage: &pkg,
		AvailableFix:      &fixPkg,
	}

	pd := shared.PackageDownload{
		Entry:            dep,
		Data:             []byte("aaaa"),
		ArtifactFileName: "artifact",
	}

	res, err := createSignaturesQuery([]shared.PackageDownload{pd})
	if err != nil {
		t.Fatalf("expected to create signatures query")
	}
	entries := res.Entries

	if len(entries) != 1 {
		t.Fatalf("expected 1 request, got %d", len(entries))
	}

	entry := entries[0]

	if entry.FileName != "artifact" {
		t.Fatalf("expected artifact, got %s", entry.FileName)
	}

	if entry.LibraryVersionId != "111" {
		t.Fatalf("expected lodash, got %s", entry.LibraryVersionId)
	}

	if entry.Architecture != nil {
		t.Fatalf("expected nil, got %v", entry.Architecture)
	}

}

func TestCreateSignaturesQueryWithArch(t *testing.T) {
	pkg := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.ApkManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	fixPkg := api.PackageVersion{
		Version:   "4.17.11-sp1",
		VersionId: "111",
		Library:   api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
	}

	dep := shared.DependencyDescriptor{
		VulnerablePackage: &pkg,
		AvailableFix:      &fixPkg,
		FixedLocations:    []string{""},
		Locations: map[string]common.Dependency{
			"": {
				Arch: "x86_64",
			},
		},
	}

	pd := shared.PackageDownload{
		Entry:            dep,
		Data:             []byte("aaaa"),
		ArtifactFileName: "artifact",
	}

	res, err := createSignaturesQuery([]shared.PackageDownload{pd})
	if err != nil {
		t.Fatalf("expected to create signatures query")
	}
	entries := res.Entries

	if len(entries) != 1 {
		t.Fatalf("expected 1 request, got %d", len(entries))
	}

	entry := entries[0]

	if entry.FileName != "artifact" {
		t.Fatalf("expected artifact, got %s", entry.FileName)
	}

	if entry.LibraryVersionId != "111" {
		t.Fatalf("expected lodash, got %s", entry.LibraryVersionId)
	}

	if *(entry.Architecture) != "x86_64" {
		t.Fatalf("expected x86_64, got %v", *(entry.Architecture))
	}

}

func TestMatchPackageToSignature(t *testing.T) {
	pkg := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	fixPkg := api.PackageVersion{
		Version:   "4.17.11-sp1",
		VersionId: "111",
		Library:   api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
	}

	dep := shared.DependencyDescriptor{
		VulnerablePackage: &pkg,
		AvailableFix:      &fixPkg,
	}

	pd := shared.PackageDownload{
		Entry:            dep,
		Data:             []byte("aaaa"),
		ArtifactFileName: "artifact",
	}

	sig := api.ArtifactMetadataResponse{
		ArtifactUniqueIdentifier: api.ArtifactUniqueIdentifier{
			LibraryVersionId: "111",
			FileName:         "artifact",
			Architecture:     nil,
		},
		SealSignature: "sig",
	}

	res, err := matchPackageToSignature([]shared.PackageDownload{pd}, []api.ArtifactMetadataResponse{sig})
	if err != nil {
		t.Fatalf("expected to match package to signature")
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 match, got %d", len(res))
	}

	if res[0].signature != "sig" && res[0].packageName != "artifact" {
		t.Fatalf("expected sig and artifact, got %s and %s", res[0].signature, res[0].packageName)
	}
}

func TestMatchPackageToSignatureWithArchitecture(t *testing.T) {
	arch := "x86_64"

	pkg := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.ApkManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	fixPkg := api.PackageVersion{
		Version:   "4.17.11-sp1",
		VersionId: "111",
		Library:   api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
	}

	dep := shared.DependencyDescriptor{
		VulnerablePackage: &pkg,
		AvailableFix:      &fixPkg,
		Locations: map[string]common.Dependency{
			"": {
				Arch: arch,
			},
		},
	}

	pd := shared.PackageDownload{
		Entry:            dep,
		Data:             []byte("aaaa"),
		ArtifactFileName: "artifact",
	}

	sig := api.ArtifactMetadataResponse{
		ArtifactUniqueIdentifier: api.ArtifactUniqueIdentifier{
			LibraryVersionId: "111",
			FileName:         "artifact",
			Architecture:     &arch,
		},
		SealSignature: "sig",
	}

	res, err := matchPackageToSignature([]shared.PackageDownload{pd}, []api.ArtifactMetadataResponse{sig})
	if err != nil {
		t.Fatalf("expected to match package to signature")
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 match, got %d", len(res))
	}

	if res[0].signature != "sig" || res[0].packageName != "lodash" {
		t.Fatalf("expected sig and artifact, got %s and %s", res[0].signature, res[0].packageName)
	}
}
