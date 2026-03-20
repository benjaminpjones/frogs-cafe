package bik

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// KeyCache fetches and caches remote server public keys.
type KeyCache struct {
	mu      sync.RWMutex
	entries map[string]keyCacheEntry
	client  *http.Client
}

type keyCacheEntry struct {
	key     ed25519.PublicKey
	fetchedAt time.Time
}

const keyCacheTTL = 1 * time.Hour

func NewKeyCache() *KeyCache {
	return &KeyCache{
		entries: make(map[string]keyCacheEntry),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchPublicKey returns the Ed25519 public key for a remote BIK server.
func (c *KeyCache) FetchPublicKey(serverDomain string) (ed25519.PublicKey, error) {
	c.mu.RLock()
	entry, ok := c.entries[serverDomain]
	c.mu.RUnlock()
	if ok && time.Since(entry.fetchedAt) < keyCacheTTL {
		return entry.key, nil
	}

	key, err := fetchRemoteKey(c.client, serverDomain)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[serverDomain] = keyCacheEntry{key: key, fetchedAt: time.Now()}
	c.mu.Unlock()
	return key, nil
}

func fetchRemoteKey(client *http.Client, serverDomain string) (ed25519.PublicKey, error) {
	keysURL := fmt.Sprintf("https://%s/.well-known/bik/keys", serverDomain)
	resp, err := client.Get(keysURL)
	if err != nil {
		return nil, fmt.Errorf("bik: fetch keys from %s: %w", serverDomain, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bik: keys endpoint returned %d for %s", resp.StatusCode, serverDomain)
	}

	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("bik: decode JWK set: %w", err)
	}

	for _, jwk := range jwks.Keys {
		if jwk.Kty == "OKP" && jwk.Crv == "Ed25519" {
			raw, err := base64.RawURLEncoding.DecodeString(jwk.X)
			if err != nil {
				return nil, fmt.Errorf("bik: decode public key: %w", err)
			}
			return ed25519.PublicKey(raw), nil
		}
	}
	return nil, fmt.Errorf("bik: no Ed25519 key found at %s", serverDomain)
}

// LookupActor resolves a @user@domain handle to an AP actor URI via WebFinger.
func LookupActor(client *http.Client, handle string) (string, error) {
	// Parse @user@domain
	if len(handle) == 0 || handle[0] != '@' {
		return "", fmt.Errorf("bik: invalid handle %q", handle)
	}
	rest := handle[1:]
	atIdx := -1
	for i, c := range rest {
		if c == '@' {
			atIdx = i
			break
		}
	}
	if atIdx < 0 {
		return "", fmt.Errorf("bik: invalid handle %q", handle)
	}
	domain := rest[atIdx+1:]

	wfURL := fmt.Sprintf("https://%s/.well-known/webfinger?resource=%s",
		domain, url.QueryEscape("acct:"+rest))

	resp, err := client.Get(wfURL)
	if err != nil {
		return "", fmt.Errorf("bik: webfinger for %s: %w", handle, err)
	}
	defer resp.Body.Close()

	var wf WebFingerResponse
	if err := json.NewDecoder(resp.Body).Decode(&wf); err != nil {
		return "", err
	}
	for _, link := range wf.Links {
		if link.Rel == "self" && link.Type == "application/activity+json" {
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("bik: no AP actor link found for %s", handle)
}
