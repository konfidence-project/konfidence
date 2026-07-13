package promotion

import (
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
)

var (
	// ErrFetchingSourceFailed indicates the source reference could not be parsed or resolved.
	ErrFetchingSourceFailed = errors.New("source resolution failed")
	// ErrSourceVerificationFailed indicates the source descriptor failed signature verification.
	ErrSourceVerificationFailed = errors.New("source verification failed")
)

func ClassifyPromotionError(err error) string {
	switch {
	case errors.Is(err, ErrFetchingSourceFailed):
		return konfidence.ReasonPromotionSourceNotFound
	case errors.Is(err, ErrSourceVerificationFailed):
		return konfidence.ReasonPromotionSourceVerificationFailed
	default:
		return konfidence.ReasonPromotionFailed
	}
}
