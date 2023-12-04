package phase

import (
	"cli/internal/config"
	"fmt"

	b64 "encoding/base64"
)

func buildAuthToken(configuration *config.Config) string {
	if configuration.Project == "" || configuration.Token == "" {
		return ""
	}

	raw := fmt.Sprintf("%s:%s",
		configuration.Project,
		configuration.Token)

	return b64.StdEncoding.EncodeToString([]byte(raw))
}
