package promotiongate

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrNotFound is returned when the VectorPromotion does not exist.
	ErrNotFound = errors.New("vectorPromotion not found")
	// ErrGateNotDeclared is returned when the promotion does not declare the gate type.
	ErrGateNotDeclared = errors.New("gate is not declared on the promotion")
	// ErrPromotionNotMutable is returned when the promotion already passed its
	// gate phase (Ready, executing or terminal), so its gates are frozen.
	ErrPromotionNotMutable = errors.New("promotion gates can no longer change")
	// ErrSetByMissing is returned when no setter identity was supplied.
	ErrSetByMissing = errors.New("setBy must not be empty")
	// ErrInvalidState is returned for a state other than Pending, Satisfied or Error.
	ErrInvalidState = errors.New("invalid gate state")
	// ErrTypeMissing is returned when no gate type was supplied.
	ErrTypeMissing = errors.New("gate type must not be empty")
)

// Update describes a gate state change and its provenance.
type Update struct {
	// Type is the gate type to set, e.g. `explicit-approval.konfidence.cloud`.
	Type string
	// State is the new gate state.
	State konfidence.PromotionGateState
	// Reason is a machine-readable CamelCase reason.
	Reason string
	// Message is an optional human-readable description.
	Message string
	// SetBy is the entity setting the gate: a controller name or, for human
	// actions, the authenticated subject.
	SetBy string
}

func (u Update) validate() error {
	switch {
	case u.Type == "":
		return ErrTypeMissing
	case u.SetBy == "":
		return ErrSetByMissing
	}
	switch u.State {
	case konfidence.PromotionGateStatePending, konfidence.PromotionGateStateSatisfied, konfidence.PromotionGateStateError:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidState, u.State)
	}
}

// Apply writes the update onto the in-memory promotion, stamping setAt with
// now. It reports whether the status changed. Setting a gate to the state it
// already has is a no-op that keeps the original provenance, so a gate is set
// to a given state at most once.
func Apply(p *konfidence.VectorPromotion, u Update, now metav1.Time) (bool, error) {
	if err := u.validate(); err != nil {
		return false, err
	}
	if !Declared(p, u.Type) {
		return false, fmt.Errorf("%w: %q", ErrGateNotDeclared, u.Type)
	}
	if !Mutable(p) {
		return false, ErrPromotionNotMutable
	}

	entry := Find(p, u.Type)
	if entry != nil && entry.State == u.State && entry.SetAt != nil {
		return false, nil
	}
	updated := konfidence.PromotionGateStatus{
		Type:    u.Type,
		State:   u.State,
		Reason:  u.Reason,
		Message: u.Message,
		SetBy:   u.SetBy,
		SetAt:   &now,
	}
	if entry != nil {
		*entry = updated
	} else {
		p.Status.Gates = append(p.Status.Gates, updated)
	}
	return true, nil
}

// Option configures Set.
type Option func(*options)

type options struct {
	reader client.Reader
}

// WithReader reads the promotion through r instead of the writing client.
// Pass an uncached reader (e.g. `mgr.GetAPIReader()`) so conflict retries
// converge on the latest resourceVersion instead of a stale cache entry.
func WithReader(r client.Reader) Option {
	return func(o *options) { o.reader = r }
}

// Set sets the state and provenance of one gate on the promotion identified
// by key in a single optimistic-locked status patch, retrying on conflict.
// It returns ErrNotFound, ErrGateNotDeclared, ErrPromotionNotMutable,
// ErrSetByMissing, ErrTypeMissing or ErrInvalidState (wrapped) for the
// corresponding failures.
func Set(ctx context.Context, c client.Client, key types.NamespacedName, u Update, opts ...Option) error {
	if err := u.validate(); err != nil {
		return err
	}
	o := options{reader: c}
	for _, opt := range opts {
		opt(&o)
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vectorPromotion := &konfidence.VectorPromotion{}
		if err := o.reader.Get(ctx, key, vectorPromotion); err != nil {
			if apierrors.IsNotFound(err) {
				return ErrNotFound
			}
			return fmt.Errorf("failed to fetch VectorPromotion %q in namespace %q: %w", key.Name, key.Namespace, err)
		}

		original := vectorPromotion.DeepCopy()
		changed, err := Apply(vectorPromotion, u, metav1.Now())
		if err != nil || !changed {
			return err
		}

		patch := client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})
		if err := c.Status().Patch(ctx, vectorPromotion, patch); err != nil {
			if apierrors.IsConflict(err) {
				return err
			}
			return fmt.Errorf("failed to patch gate %q onto VectorPromotion %q in namespace %q: %w",
				u.Type, key.Name, key.Namespace, err)
		}
		return nil
	})
}
