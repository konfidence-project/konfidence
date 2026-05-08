package domain

import (
	"errors"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
)

var (
	// ErrFetchingSourceFailed indicates the source reference could not be parsed or resolved.
	ErrFetchingSourceFailed = errors.New("source resolution failed")
)

func ClassifyPromotionError(err error) string {
	switch {
	case errors.Is(err, ErrFetchingSourceFailed):
		return global.ReasonPromotionSourceNotFound
	default:
		return global.ReasonPromotionFailed
	}
}
