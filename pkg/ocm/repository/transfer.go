package repository

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_transfer_executor.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/repository TransferExecutor

import (
	"context"
	"fmt"

	"ocm.software/open-component-model/bindings/go/transform/graph/builder"
	transformv1alpha1 "ocm.software/open-component-model/bindings/go/transform/spec/v1alpha1"
)

// TransferExecutor builds and executes a component transfer from a graph definition.
//
// The definition describes which components to transfer, from which source repositories,
// to which target repositories, and how to handle resources during the transfer.
type TransferExecutor interface {
	Execute(ctx context.Context, definition *transformv1alpha1.TransformationGraphDefinition) error
}

var _ TransferExecutor = (*DefaultTransferExecutor)(nil)

// DefaultTransferExecutor wraps the OCM library's graph builder to execute transfers.
type DefaultTransferExecutor struct {
	builder *builder.Builder
}

// NewDefaultTransferExecutor creates a TransferExecutor backed by the given graph builder.
func NewDefaultTransferExecutor(b *builder.Builder) *DefaultTransferExecutor {
	return &DefaultTransferExecutor{builder: b}
}

// Execute builds a transfer graph from the definition and processes it.
func (e *DefaultTransferExecutor) Execute(
	ctx context.Context,
	definition *transformv1alpha1.TransformationGraphDefinition,
) error {
	graph, err := e.builder.BuildAndCheck(definition)
	if err != nil {
		return fmt.Errorf("building and checking transfer graph: %w", err)
	}
	if err = graph.Process(ctx); err != nil {
		return fmt.Errorf("processing transfer graph: %w", err)
	}
	return nil
}
