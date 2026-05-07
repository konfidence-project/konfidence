package domain

import (
	"errors"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
)

var (
	// ErrFetchingSourceFailed indicates the source reference could not be parsed or resolved.
	ErrFetchingSourceFailed = errors.New("source resolution failed")
	// ErrInvalidConfiguration indicates the VectorPromotionConfig is invalid.
	ErrInvalidConfiguration = errors.New("invalid promotion configuration")
)

func ClassifyPromotionError(err error) string {
	switch {
	case errors.Is(err, ErrFetchingSourceFailed):
		return global.ReasonPromotionSourceNotFound
	case errors.Is(err, ErrInvalidConfiguration):
		return global.ReasonInvalidPromotionConfiguration
	default:
		return global.ReasonPromotionFailed
	}
}
