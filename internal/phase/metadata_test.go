//go:build !mockserver
// +build !mockserver

package phase

import (
	"cli/internal/ecosystem/shared"
	"testing"
)

func TestManagerMetadata(t *testing.T) {
	fakeManager := shared.FakePackageManager{
		ManagerName: "fakename",
		Version:     "1.2.3",
	}

	res, err := managerMetadata(&fakeManager)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if res.Name != fakeManager.ManagerName {
		t.Fatalf("wrong manager name: `%s`", res.Name)
	}

	if res.Version != fakeManager.Version {
		t.Fatalf("wrong manager version: `%s`", res.Version)
	}
}

func TestGatherMetadata(t *testing.T) {
	fakeManager := shared.FakePackageManager{
		ManagerName: "fakename",
		Version:     "1.2.3",
	}

	res, err := gatherMetadata(&fakeManager)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	data, found := res[fakeManager.ManagerName]
	if !found {
		t.Fatalf("did not find manager metadata in: %v", res)
	}

	mngrData := data.(*PackageManagerMetadata)
	if mngrData.Name != fakeManager.ManagerName {
		t.Fatalf("wrong manager name: `%s`", mngrData.Name)
	}

	if mngrData.Version != fakeManager.Version {
		t.Fatalf("wrong manager version: `%s`", mngrData.Version)
	}
}
