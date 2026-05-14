package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

func SignPayload(secret []byte, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func VerifyPayloadSignature(secret []byte, payload []byte, signature string) error {
	expected := SignPayload(secret, payload)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.New("invalid signature")
	}
	return nil
}
