// Package credentials converts Konfidence API credential references into a
// credentials.Resolver that the OCM runtime can use to authenticate against
// OCI registries and resolve signing key material.
//
// The single entry point is [ResolverFromCredentials]. It accepts a
// *konfidence.Credentials value read from a CR spec, reads the referenced
// Secrets and ConfigMaps from the Kubernetes API, and assembles an OCM
// credential graph. The graph is returned as a [credentials.Resolver] that
// both [pkg/ocm/repository] (OCI transport) and [pkg/ocm/crypto] (signature
// verification and signing) consume via their respective WithResolver methods.
//
// When creds is nil or creds.OCM is nil the function returns (nil, nil).
// A nil resolver propagates to the builder layer where each builder decides
// the correct behaviour: [VerifierBuilder] passes nil through to the internal
// OCM verifier which substitutes the system trust store; [SignerBuilder] rejects nil when
// specs are present, surfacing the misconfiguration at construction time rather
// than at the first Sign call.
//
// This package has no knowledge of caching, client construction, or
// reconcile logic. It does one thing: Konfidence API → credentials.Resolver.
package credentials
