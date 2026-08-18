// Package ocm provides test helpers for pushing OCM component descriptors,
// signing them out-of-band, and building credential Secrets for envtest suites.
//
// Push helpers (push.go): PushComponent and PushVector push plain descriptors;
// PushSignedComponent and PushSignedVector sign descriptors in-place before pushing.
//
// Signing helper (sign.go): SignDescriptor signs a descriptor using a transient
// in-memory credential graph built directly from RSAKeyPair material.
//
// Credential helpers (secrets.go): OCMConfigSecret builds a .ocmconfig Secret
// with RSACredentials/v1 consumers bound to named signatures; DockerConfigSecret
// builds a .dockerconfigjson Secret for OCI registry authentication. Both are
// used to populate the credential graph fed to ResolverFromRefs in envtest suites.
//
// All exported functions use Gomega's ExpectWithOffset so that failures are
// reported at the call site rather than inside this package.
package ocm
