package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	. "github.com/onsi/gomega" //nolint:staticcheck
)

// RSAKeyPair holds an in-memory RSA key pair as raw PEM bytes.
type RSAKeyPair struct {
	PrivateKeyPEM  []byte // PKCS#1 PEM block "RSA PRIVATE KEY"
	CertificatePEM []byte // Self-signed X.509 cert PEM "CERTIFICATE"
}

// GenerateRSAKeyPair generates a 2048-bit RSA key pair and a self-signed certificate
// with the given CN. All bytes are in-memory only.
// Fails the current Gomega test on any error, reporting the failure at the caller's location.
func GenerateRSAKeyPair(cn string) RSAKeyPair {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "generate RSA key for %s", cn)

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "create certificate for %s", cn)

	return RSAKeyPair{
		PrivateKeyPEM:  privPEM,
		CertificatePEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
	}
}
