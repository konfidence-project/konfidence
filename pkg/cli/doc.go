// Package cli provides shared startup wiring for the konfidence operator
// binaries (galaxy and star). It currently exposes two concerns:
//
//   - Filter — selecting which controllers an operator binary runs at startup,
//     driven by the operator's --controllers flag.
//   - ResolveCryptoConfig — resolving OCM signing/verification dependencies
//     from environment variables and registering the necessary watches with a
//     controller-runtime manager.
//
// # Selecting controllers with --controllers
//
// Each operator binary registers all controllers it could run, then enables a
// subset based on a comma-separated glob expression supplied via the
// --controllers flag. The default value is "*", which runs every registered
// controller (the original, pre-flag behavior).
//
// The grammar is:
//
//	spec   = token ("," token)*
//	token  = ["!"] glob
//	glob   = a path.Match pattern, supporting "*", "?", and "[...]"
//
// Semantics are set-based and order-independent: positive tokens contribute
// matching controllers to a union, negative tokens (those prefixed with "!")
// contribute matching controllers to an exclusion set, and the result is
// (positives) minus (negatives). The position of a token within the spec does
// not change the outcome — "!Foo,*" and "*,!Foo" are equivalent.
//
// # Examples
//
// Given an operator that registers StageConfiguration, VectorPromotion, and
// VectorAssembly:
//
//	# Run everything (also the default if --controllers is omitted)
//	galaxy --controllers='*'
//
//	# Run only one controller
//	galaxy --controllers=VectorAssembly
//
//	# Run everything except one
//	galaxy --controllers='!VectorAssembly,*'
//
//	# Run an explicit subset
//	galaxy --controllers=VectorAssembly,VectorPromotion
//
//	# Run all controllers whose name starts with Vector
//	galaxy --controllers='Vector*'
//
// Whitespace around tokens is tolerated. Quote the spec in shells where "!" or
// "*" would otherwise be interpreted by the shell.
//
// # Errors
//
// Filter rejects two classes of input to fail fast at startup rather than
// silently running an unintended set of controllers:
//
//   - A malformed glob pattern (path.ErrBadPattern), e.g. "[".
//   - A literal token (one containing none of "*", "?", "[") that matches no
//     registered controller. This guards against typos and against stale
//     references to controllers that have been removed or are not built into
//     this particular binary. Wildcard tokens that happen to match nothing
//     (e.g. "Foo*" with no Foo-prefixed controllers registered) are allowed.
//
// # Usage in operator wiring
//
// Operator binaries call Filter once after building their registry of
// controller setups, then iterate the registry and skip entries whose names
// are not in the returned set. Because Filter returns a map[string]bool the
// caller checks membership directly without any intermediate conversion:
//
//	setups := []struct {
//	    Name  string
//	    Setup func() error
//	}{
//	    {Name: "StageConfiguration", Setup: func() error { ... }},
//	    {Name: "VectorPromotion",    Setup: func() error { ... }},
//	    {Name: "VectorAssembly",     Setup: func() error { ... }},
//	}
//
//	names := make([]string, len(setups))
//	for i, s := range setups {
//	    names[i] = s.Name
//	}
//	enabled, err := cli.Filter(controllersSpec, names)
//	if err != nil {
//	    return err
//	}
//	for _, s := range setups {
//	    if !enabled[s.Name] {
//	        continue
//	    }
//	    if err := s.Setup(); err != nil {
//	        return err
//	    }
//	}
//
// Side-effecting work that conceptually belongs to a single controller (for
// example the stageVersion garbage collector goroutine, which is owned by the
// star Stage controller) should be launched from inside that controller's
// Setup closure so it is naturally gated on the same flag.
//
// # Crypto configuration
//
// ResolveCryptoConfig is described on its own declaration. Briefly: it reads
// OCM_VECTOR_VERIFY, OCM_ARTIFACT_VERIFY, OCM_VECTOR_SIGN and the associated
// trust-anchor / signing-credential references from the environment, sets up
// the corresponding ConfigMap and Secret watches with the supplied
// controller-runtime manager, and returns a CryptoConfig with verifier and
// signer implementations (or noop implementations when the corresponding
// feature is disabled).
package cli
