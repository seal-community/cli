package blackduck

type bdAPITokenResponse struct {
	BearerToken           string `json:"bearerToken"`
	ExpiresInMilliseconds int    `json:"expiresInMilliseconds"`
}
