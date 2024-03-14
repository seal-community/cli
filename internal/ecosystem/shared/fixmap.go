package shared

import (
	"cli/internal/api"
	"fmt"
)

func FormatFixKey(p *api.PackageVersion) string {
	recommendedId := p.RecommendedId()
	packageId := p.Id()
	return fmt.Sprintf("%s -> %s", packageId, recommendedId)
}
