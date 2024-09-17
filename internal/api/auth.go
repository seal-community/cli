package api

import (
	"fmt"
)

func BuildBasicAuthHeader(token string) StringPair {
	return StringPair{"Authorization", fmt.Sprintf("Basic %s", token)}
}

func BuildBearerAuthHeader(token string) StringPair {
	return StringPair{"Authorization", fmt.Sprintf("Bearer %s", token)}
}
