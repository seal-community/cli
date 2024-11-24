package output

import (
	"cli/internal/api"
	"testing"
)

func TestFormatVulnCsv(t *testing.T) {
	t.Parallel()

	type args struct {
		vulnerability api.Vulnerability
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Format Vuln CSV",
			args: args{
				vulnerability: api.Vulnerability{
					CVE: "CVE-2021-1234",
				},
			},
			want: "CVE-2021-1234",
		},
		{
			name: "Test Format Vuln CSV with embedded via",
			args: args{
				vulnerability: api.Vulnerability{
					CVE: "CVE-2021-1234",
					EmbeddedVia: []api.PublicPackage{
						{
							Name:           "lib1",
							Version:        "1.2.3",
							PackageManager: "Maven",
						},
					},
				},
			},
			want: "CVE-2021-1234(via shaded lib1)",
		},
		{
			name: "Test Format Vuln CSV with multiple embedded via",
			args: args{
				vulnerability: api.Vulnerability{
					CVE: "CVE-2021-1234",
					EmbeddedVia: []api.PublicPackage{
						{
							Name:           "lib1",
							Version:        "1.2.3",
							PackageManager: "Maven",
						},
						{
							Name:           "lib2",
							Version:        "4.5.6.",
							PackageManager: "Maven",
						},
					},
				},
			},
			want: "CVE-2021-1234(via shaded lib1&lib2)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatVulnCsv(tt.args.vulnerability); got != tt.want {
				t.Errorf("FormatVulnCsv() = %v, want %v", got, tt.want)
			}
		})
	}
}
