package phase

import (
	"fmt"

	b64 "encoding/base64"
)

func buildAuthToken(token string, projectTag string) string {
	if projectTag == "" || token == "" {
		return ""
	}

	raw := fmt.Sprintf("%s:%s",
		projectTag,
		token)

	return b64.StdEncoding.EncodeToString([]byte(raw))
}
