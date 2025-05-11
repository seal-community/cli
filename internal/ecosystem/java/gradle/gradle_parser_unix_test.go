//go:build !windows

package gradle

import (
	"cli/internal/ecosystem/java/utils"
	"os"
	"path/filepath"
	"testing"
)

func getTestFile(name string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func TestParseGradleVersion_8_14(t *testing.T) {
	data := getTestFile("gradle_8.14_version_macos.txt")
	version := parseVersionOutput(data)
	if version != "8.14" {
		t.Fatalf("unexpected version: `%s`", version)
	}
}

func TestParseGradleAllProjects_8_14(t *testing.T) {
	data := getTestFile("gradle_8.14_allprojects_macos.txt")
	projects := parseProjectsOutput(data)
	if projects == nil {
		t.Fatalf("unexpected nil")
	}

	if len(projects) != 3 {
		t.Fatalf("unexpected result: `%v`", projects)
	}

	// root project
	if p := projects[0]; p != "" {
		t.Fatalf("unexpected project name: `%s`", p)
	}

	if p := projects[1]; p != "app" {
		t.Fatalf("unexpected project name: `%s`", p)
	}

	if p := projects[2]; p != "seal-dummy-project" {
		t.Fatalf("unexpected project name: `%s`", p)
	}
}

func TestParseHomeDir(t *testing.T) {
	data := getTestFile("gradle_8.14_status_for_cache_macos.txt")
	homeDir := parseHomeDir(data)
	if homeDir != "/Users/senshou/Downloads/untitled folder/gr/.seal" {
		t.Fatalf("unexpected homedir: `%s`", homeDir)
	}
}

func TestParseDependenciesDupSubTree(t *testing.T) {
	data := getTestFile("gradle_8.14_deps_dup_tree.txt")
	packages := parsePackages(data, CompileClasspath)

	if packages == nil {
		t.Fatalf("got deps: %v", packages)
	}

	if len(packages) != 8 {
		t.Fatalf("wrong number of deps: %v", packages)
	}

	d := make(map[string]*utils.JavaPackageInfo)
	for _, p := range packages {

		d[p.Id()] = &p
	}

	if foundPackage := d["com.google.guava:guava:32.1.2-jre:compileClasspath"]; foundPackage == nil {
		t.Fatalf("did not detect dep with 'dup' subtree in deps: %v", d)
	}
}

func TestParseDependencies(t *testing.T) {
	data := getTestFile("gradle_8.14_deps_no_lock_macos.txt")
	packages := parsePackages(data, CompileClasspath)

	if packages == nil {
		t.Fatalf("got deps: %v", packages)
	}

	if len(packages) != 10 {
		t.Fatalf("wrong number of deps: %v", packages)
	}

	p := packages[0]
	if p.ArtifactName != "guava" || p.Version != "33.4.5-jre" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[1]
	if p.ArtifactName != "failureaccess" || p.Version != "1.0.3" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[2]
	if p.ArtifactName != "listenablefuture" || p.Version != "9999.0-empty-to-avoid-conflict-with-guava" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[3]
	if p.ArtifactName != "jspecify" || p.Version != "1.0.0" || p.OrgName != "org.jspecify" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[4]
	if p.ArtifactName != "error_prone_annotations" || p.Version != "2.36.0" || p.OrgName != "com.google.errorprone" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[5]
	if p.ArtifactName != "j2objc-annotations" || p.Version != "3.0.0" || p.OrgName != "com.google.j2objc" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[6]
	if p.ArtifactName != "commons-io" || p.Version != "2.2" || p.OrgName != "commons-io" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[7]
	if p.ArtifactName != "spring-beans" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[8]
	if p.ArtifactName != "spring-core" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[9]
	if p.ArtifactName != "spring-jcl" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}
}

func TestParseDependenciesWithLock(t *testing.T) {
	data := getTestFile("gradle_8.14_deps_with_lock_macos.txt")
	packages := parsePackages(data, CompileClasspath)

	if packages == nil {
		t.Fatalf("got deps: %v", packages)
	}

	if len(packages) != 10 {
		t.Fatalf("wrong number of deps: %v", packages)
	}

	p := packages[0]
	if p.ArtifactName != "guava" || p.Version != "33.4.5-jre" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[1]
	if p.ArtifactName != "failureaccess" || p.Version != "1.0.3" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[2]
	if p.ArtifactName != "listenablefuture" || p.Version != "9999.0-empty-to-avoid-conflict-with-guava" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[3]
	if p.ArtifactName != "jspecify" || p.Version != "1.0.0" || p.OrgName != "org.jspecify" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[4]
	if p.ArtifactName != "error_prone_annotations" || p.Version != "2.36.0" || p.OrgName != "com.google.errorprone" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[5]
	if p.ArtifactName != "j2objc-annotations" || p.Version != "3.0.0" || p.OrgName != "com.google.j2objc" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[6]
	if p.ArtifactName != "commons-io" || p.Version != "2.2" || p.OrgName != "commons-io" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[7]
	if p.ArtifactName != "spring-beans" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[8]
	if p.ArtifactName != "spring-core" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[9]
	if p.ArtifactName != "spring-jcl" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}
}

func TestParseDependenciesWithLockMissingDeps(t *testing.T) {
	// when they are missing, we hafe "FAILED" string after the version, but if they are installed again they should appear as its because of the lock. so we include them
	data := getTestFile("gradle_8.14_deps_with_lock_with_missing_deps_macos.txt")
	packages := parsePackages(data, CompileClasspath)

	if packages == nil {
		t.Fatalf("got deps: %v", packages)
	}

	if len(packages) != 11 {
		t.Fatalf("wrong number of deps: %v", packages)
	}

	p := packages[0]
	if p.ArtifactName != "guava" || p.Version != "33.4.5-jre" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[1]
	if p.ArtifactName != "failureaccess" || p.Version != "1.0.3" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[2]
	if p.ArtifactName != "listenablefuture" || p.Version != "9999.0-empty-to-avoid-conflict-with-guava" || p.OrgName != "com.google.guava" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[3]
	if p.ArtifactName != "jspecify" || p.Version != "1.0.0" || p.OrgName != "org.jspecify" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[4]
	if p.ArtifactName != "error_prone_annotations" || p.Version != "2.36.0" || p.OrgName != "com.google.errorprone" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[5]
	if p.ArtifactName != "j2objc-annotations" || p.Version != "3.0.0" || p.OrgName != "com.google.j2objc" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[6]
	if p.ArtifactName != "commons-io" || p.Version != "2.2" || p.OrgName != "commons-io" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	// managed to have it twice in the gradle build after messing with dependencies, one was FAILED
	p = packages[7]
	if p.ArtifactName != "commons-io" || p.Version != "2.2" || p.OrgName != "commons-io" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}

	p = packages[8]
	if p.ArtifactName != "spring-core" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}
	p = packages[9]
	if p.ArtifactName != "spring-beans" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}
	p = packages[10]
	if p.ArtifactName != "spring-jcl" || p.Version != "5.3.10" || p.OrgName != "org.springframework" || p.Scope != string(CompileClasspath) {
		t.Fatalf("wrong parsed package: %v", p)
	}
}
