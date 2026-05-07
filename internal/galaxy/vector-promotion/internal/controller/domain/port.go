package domain

import (
	"context"
)

// PromotionPort abstracts the OCM operations required by the promotion controller.
type PromotionPort interface {
	// Promote executes the full promotion flow:
	// 1. Resolves the source reference to a concrete version
	// 2. Copies the component to the target repository (if cross-repo)
	// 3. Moves the target alias to point to the resolved version
	Promote(ctx context.Context, source, target string) error
}
