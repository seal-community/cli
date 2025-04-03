package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// loads an ECDSA public key from a base64-encoded string
// and ensures it's using the NIST P-256 curve.
func LoadECDSAPublicKeyFromBase64(base64PublicKey string) (*ecdsa.PublicKey, error) {
	// decode the base64-encoded public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(base64PublicKey)
	if err != nil {
		return nil, err
	}

	// parse the public key from the decoded bytes
	pub, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}

	// assert the key type is ECDSA
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, err
	}

	// check that the curve is P-256
	if ecdsaPub.Curve != elliptic.P256() {
		return nil, fmt.Errorf("invalid curve: %s, expecting P-256", ecdsaPub.Curve.Params().Name)
	}

	return ecdsaPub, nil
}

// verifies an ECDSA signature against a base64-encoded message and signature
func VerifySignature(publicKey *ecdsa.PublicKey, messageBase64 string, signatureBase64 string) (bool, error) {
	// decode the message from base64
	message, err := base64.StdEncoding.DecodeString(messageBase64)
	if err != nil {
		return false, fmt.Errorf("failed to decode base64 message: %v", err)
	}

	// decode the signature from base64
	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, fmt.Errorf("failed to decode base64 signature: %v", err)
	}

	// we use sha256 to hash the message
	hash := sha256.Sum256(message)

	// verify the signature
	valid := ecdsa.VerifyASN1(publicKey, hash[:], signature)
	return valid, nil
}
