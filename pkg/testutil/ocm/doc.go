// Package ocm provides test helpers for seeding OCM component descriptors into
// an OCI registry using the konfidence pkg/ocm/repository client.
//
// These helpers are intended for use in integration test suites that spin up a
// real OCI registry (e.g. via testutil/ociregistry) and need to push minimal
// but valid component descriptors as test fixtures.
//
// All exported functions use Gomega's ExpectWithOffset so that failures are
// reported at the call site rather than inside this package.
package ocm
