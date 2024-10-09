package common

import "testing"

func TestDependencySignatureId(t *testing.T) {
	signature := "bbb|lodash@1.2.3"
	generated := DependencyId("bbb", "lodash", "1.2.3")
	if generated != signature {
		t.Fatalf("wrong dep version signature; generated: '%s' expected: '%s'", generated, signature)
	}
}

func TestDependencyDescriptor(t *testing.T) {
	dep := Dependency{
		Name:           "lodash",
		Version:        "1.2.3",
		PackageManager: "bbb",
	}

	depDescriptor := dep.Descriptor()
	expected := "lodash@1.2.3"
	if depDescriptor != expected {
		t.Fatalf("wrong dep descriptor; got: '%s' expected: %s", depDescriptor, expected)
	}
}
