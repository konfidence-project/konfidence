package promotion

import (
	"errors"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
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
		return galaxy.ReasonPromotionSourceNotFound
	case errors.Is(err, ErrSourceVerificationFailed):
		return galaxy.ReasonPromotionSourceVerificationFailed
	default:
		return galaxy.ReasonPromotionFailed
	}
}
