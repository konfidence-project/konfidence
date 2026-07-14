// Package cmd provides shared startup wiring for the konfidence operator
// binary. It exposes one concern:
//
//   - FilterEnabledControllers — selecting which controllers the operator runs at startup,
//     driven by the operator's --controllers flag.
//
// # Selecting controllers with --controllers
//
// The operator binary registers all controllers it could run, then enables a
// subset based on a comma-separated glob expression supplied via the
// --controllers flag. The default value is "*", which runs every registered
// controller (the original, pre-flag behavior). The registry itself can be
// narrowed before filtering: the konfidence binary's --enable-galaxy=false
// leaves the galaxy controllers unregistered, so naming one of them in
// --controllers becomes a startup error.
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
//	konfidence --controllers='*'
//
//	# Run only one controller
//	konfidence --controllers=VectorAssembly
//
//	# Run everything except one
//	konfidence --controllers='!VectorAssembly,*'
//
//	# Run an explicit subset
//	konfidence --controllers=VectorAssembly,VectorPromotion
//
//	# Run all controllers whose name starts with Vector
//	konfidence --controllers='Vector*'
//
// Whitespace around tokens is tolerated. Quote the spec in shells where "!" or
// "*" would otherwise be interpreted by the shell.
//
// # Errors
//
// FilterEnabledControllers rejects two classes of input to fail fast at startup rather than
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
// Operator binaries call FilterEnabledControllers once after building their registry of
// controller setups, then iterate the registry and skip entries whose names
// are not in the returned set. Because FilterEnabledControllers returns a map[string]bool the
// caller checks membership directly without any intermediate conversion:
//
//	controllerSetups := map[string]func() error{
//		"StageConfiguration": func() error { ... },
//		"VectorAssembly":     func() error { ... },
//		"VectorPromotion":    func() error { ... },
//	}
//
//	enabled, err := cmd.FilterEnabledControllers(controllersSpec, controllerSetups)
//	if err != nil {
//		return err
//	}
//
//	for name, setup := range controllerSetups {
//		if !enabled[name] {
//			continue
//		}
//		if err := setup(); err != nil {
//			return err
//		}
//	}
//
// Side-effecting work that conceptually belongs to a single controller (for
// example the stageVersion garbage collector goroutine, which is owned by the
// star Stage controller) should be launched from inside that controller's
// Setup closure so it is naturally gated on the same flag.
package cmd
