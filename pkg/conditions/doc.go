// Package conditions provides types and interfaces for managing conditions in a Kubernetes-like environment.
package conditions

//go:generate mockgen -source=types.go -destination=mocks/mock_types.go -package=mocks
