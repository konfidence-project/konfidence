// Package pki provides in-memory RSA key generation helpers for use in test suites.
//
// GenerateRSAKeyPair produces a 2048-bit RSA private key and a self-signed X.509
// certificate as raw PEM bytes.
//
// All exported functions use Gomega's ExpectWithOffset so that failures are
// reported at the call site rather than inside this package.
package pki
