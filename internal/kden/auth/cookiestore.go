package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/zalando/go-keyring"
)

const keyringService = "kden"

type CookieStore interface {
	Load(endpoint string) (*http.Cookie, error)
	Save(endpoint string, cookie *http.Cookie) error
	Delete(endpoint string) error
}

type KeyringCookieStore struct{}

func (KeyringCookieStore) Load(endpoint string) (*http.Cookie, error) {
	value, err := keyring.Get(keyringService, credentialAccount(endpoint))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading session cookie from OS keyring failed: %w", err)
	}

	var cookie http.Cookie
	if err := json.Unmarshal([]byte(value), &cookie); err != nil {
		return nil, fmt.Errorf("decoding stored session cookie failed: %w", err)
	}

	if cookie.Name == "" || cookie.Value == "" {
		return nil, fmt.Errorf("stored session cookie is invalid")
	}

	return &cookie, nil
}

func (KeyringCookieStore) Save(endpoint string, cookie *http.Cookie) error {
	if cookie == nil || cookie.Name == "" || cookie.Value == "" {
		return fmt.Errorf("session cookie is empty")
	}

	value, err := json.Marshal(cookie)
	if err != nil {
		return fmt.Errorf("encoding session cookie: %w", err)
	}

	if err := keyring.Set(keyringService, credentialAccount(endpoint), string(value)); err != nil {
		return fmt.Errorf("saving session cookie in OS keyring: %w", err)
	}

	return nil
}

func (KeyringCookieStore) Delete(endpoint string) error {
	err := keyring.Delete(keyringService, credentialAccount(endpoint))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deleting session cookie from OS keyring: %w", err)
	}

	return nil
}

func credentialAccount(endpoint string) string {
	normalized := strings.TrimRight(endpoint, "/")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
