package files

import (
	"testing"
)

func TestParseJarPath(t *testing.T) {
	cases := []struct {
		jarPath    string
		artifactId string
		version    string
		err        bool
	}{
		{
			jarPath:    "path/to/jar/dependency-1.0.0.jar",
			artifactId: "dependency",
			version:    "1.0.0",
			err:        false,
		},
		{
			jarPath:    "path/to/jar/dependency-1.0.0-SNAPSHOT.jar",
			artifactId: "dependency",
			version:    "1.0.0-SNAPSHOT",
			err:        false,
		},
		{
			jarPath:    "path/to/jar/bad-dependency-1.0.0-SNAPSHOT-1.jar",
			artifactId: "bad-dependency",
			version:    "1.0.0-SNAPSHOT-1",
			err:        false,
		},
		{
			jarPath:    "asciidoctor-gradle-plugin-0.2.2-20130326.120112-1.jar",
			artifactId: "asciidoctor-gradle-plugin",
			version:    "0.2.2-20130326.120112-1",
			err:        false,
		},
		{
			jarPath:    "hadoop-shaded-protobuf_3_21-1.2.0+sp1.jar",
			artifactId: "hadoop-shaded-protobuf_3_21",
			version:    "1.2.0+sp1",
			err:        false,
		},
		{
			jarPath:    "spring-beans-4.0.9.RELEASE+sp1.jar",
			artifactId: "spring-beans",
			version:    "4.0.9.RELEASE+sp1",
			err:        false,
		},
	}

	for _, c := range cases {
		artifactId, version, err := parseJarPath(c.jarPath)
		if artifactId != c.artifactId || version != c.version {
			t.Fatalf("got %v %v %v", artifactId, version, err)
		}
		if c.err && err == nil {
			t.Fatalf("expected error")
		}
	}
}
