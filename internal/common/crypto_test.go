package common

import "testing"

func TestLoadECDSAPublicKeyFromBase64(t *testing.T) {
	key := "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEi0BRwnEStCPSWz6vpyv7lz8N1NOXTiUlwvVscU7cUqmnC6tM1thDpAoKX+wkPUrrxBbjok+mYgDoH/FSDm1LKw=="
	_, err := LoadECDSAPublicKeyFromBase64(key)
	if err != nil {
		t.Fatalf("failed to load key: %v", err)
	}
}

func TestVerifySignature(t *testing.T) {
	key := "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEi0BRwnEStCPSWz6vpyv7lz8N1NOXTiUlwvVscU7cUqmnC6tM1thDpAoKX+wkPUrrxBbjok+mYgDoH/FSDm1LKw=="
	pubKey, _ := LoadECDSAPublicKeyFromBase64(key)
	sig := "MEQCIHw6auTgJ+3SPJQjvEotHXPzfoBvIaKZDCD8CfjqNGP2AiAlkVkqVpUoz4JbgKGcCc83y/IvYW7SZsmyoxVKXubeEQ=="
	msg := "spCSE9xcfAPtoeHm5XY79QGLDSZPoGUodkjBwV28OVc3ddhZ4kMYPj2jetmkZH5OQC0qxczBwmNuTb/dXEpU8w=="
	res, err := VerifySignature(pubKey, msg, sig)
	if err != nil || !res {
		t.Fatalf("failed to verify signature: %v", err)
	}
}

func TestVerifySignatureFail(t *testing.T) {
	key := "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEi0BRwnEStCPSWz6vpyv7lz8N1NOXTiUlwvVscU7cUqmnC6tM1thDpAoKX+wkPUrrxBbjok+mYgDoH/FSDm1LKw=="
	pubKey, _ := LoadECDSAPublicKeyFromBase64(key)
	sig := "NEQCIHw6auTgJ+3SPJQjvEotHXPzfoBvIaKZDCD8CfjqNGP2AiAlkVkqVpUoz4JbgKGcCc83y/IvYW7SZsmyoxVKXubeEQ=="
	msg := "spCSE9xcfAPtoeHm5XY79QGLDSZPoGUodkjBwV28OVc3ddhZ4kMYPj2jetmkZH5OQC0qxczBwmNuTb/dXEpU8w=="
	res, _ := VerifySignature(pubKey, msg, sig)
	if res {
		t.Fatalf("signature verification should fail")
	}
}
