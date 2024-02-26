package common

import "testing"

func TestDependencySignatureId(t *testing.T) {
	signature := "bbb|lodash@1.2.3"
	generated := DependencyId("bbb", "lodash", "1.2.3")
	if generated != signature {
		t.Fatalf("wrong dep version signature; generated: '%s' expected: '%s'", generated, signature)
	}
}
