package bik

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"
)

// Keys holds the server's Ed25519 key pair for signing/verifying BIK tokens.
type Keys struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

// GenerateKeys creates a new Ed25519 key pair.
func GenerateKeys() (*Keys, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Keys{Private: priv, Public: pub}, nil
}

// LoadOrGenerateKeys loads keys from a PEM string, or generates new ones if empty.
func LoadOrGenerateKeys(pemStr string) (*Keys, error) {
	if pemStr == "" {
		return GenerateKeys()
	}
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("bik: invalid PEM data")
	}
	if len(block.Bytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("bik: expected %d byte Ed25519 key, got %d", ed25519.PrivateKeySize, len(block.Bytes))
	}
	priv := ed25519.PrivateKey(block.Bytes)
	return &Keys{Private: priv, Public: priv.Public().(ed25519.PublicKey)}, nil
}

// JWKSet returns the public key as a JWK set for the /.well-known/bik/keys endpoint.
func (k *Keys) JWKSet() JWKSet {
	return JWKSet{
		Keys: []JWK{{
			Kty: "OKP",
			Crv: "Ed25519",
			X:   base64.RawURLEncoding.EncodeToString(k.Public),
		}},
	}
}

// SignToken creates a signed BIK token for a player joining a game.
func (k *Keys) SignToken(player, gameURI string, ttl time.Duration) (string, error) {
	payload := BikToken{
		Player: player,
		Game:   gameURI,
		Exp:    time.Now().Add(ttl).Unix(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(k.Private, data)

	// Token format: base64url(payload).base64url(signature)
	return base64.RawURLEncoding.EncodeToString(data) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyToken verifies a BIK token signed by the given public key.
func VerifyToken(token string, pubKey ed25519.PublicKey) (*BikToken, error) {
	dotIdx := findLastDot(token)
	if dotIdx < 0 {
		return nil, errors.New("bik: malformed token")
	}

	data, err := base64.RawURLEncoding.DecodeString(token[:dotIdx])
	if err != nil {
		return nil, fmt.Errorf("bik: decode payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(token[dotIdx+1:])
	if err != nil {
		return nil, fmt.Errorf("bik: decode signature: %w", err)
	}

	if !ed25519.Verify(pubKey, data, sig) {
		return nil, errors.New("bik: invalid signature")
	}

	var t BikToken
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("bik: decode token: %w", err)
	}
	if time.Now().Unix() > t.Exp {
		return nil, errors.New("bik: token expired")
	}
	return &t, nil
}

func findLastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}
