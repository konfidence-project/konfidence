package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// global keys for simplicity in simulation
var (
	privateKey *rsa.PrivateKey
	kid        = "konfidence-test-key-id"
	issuerURL  = "https://id.localhost"
	subject    = "repo:konfidence-project/konfidence:*"
	audience   = "https://konfidence.example/api"
)

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func main() {
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA key: %v", err)
	}

	http.HandleFunc("/.well-known/openid-configuration", handleOIDCConfiguration)
	http.HandleFunc("/openid/v1/jwks", handleJWKS)
	http.HandleFunc("/token", handleGenerateToken) // Helper endpoint to mint tokens

	addr := "localhost:8092"
	fmt.Printf("Workload Identity Simulator running on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// handleOIDCConfiguration serves the discovery metadata
func handleOIDCConfiguration(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"issuer":                                issuerURL,
		"jwks_uri":                              fmt.Sprintf("%s/openid/v1/jwks", issuerURL),
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// handleJWKS exposes the public key so external cloud providers can verify tokens
func handleJWKS(w http.ResponseWriter, r *http.Request) {
	publicKey := &privateKey.PublicKey

	// encode RSA components to Base64URL padding-less format required by JWK
	nStr := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	eBytes := big.NewInt(int64(publicKey.E)).Bytes()
	eStr := base64.RawURLEncoding.EncodeToString(eBytes)

	jwks := JWKS{
		Keys: []JWK{
			{
				Kty: "RSA",
				Alg: "RS256",
				Use: "sig",
				Kid: kid,
				N:   nStr,
				E:   eStr,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

// handleGenerateToken simulates a workload minting its own identity token
func handleGenerateToken(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	// Create standard Workload Identity claims
	claims := jwt.MapClaims{
		"iss": issuerURL,
		"sub": subject,
		"aud": audience,
		"exp": now.Add(1 * time.Hour).Unix(),
		"iat": now.Unix(),
		"nbf": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		http.Error(w, "Failed to sign token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id_token": tokenString})
}
