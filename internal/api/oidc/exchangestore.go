package oidc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maypok86/otter/v2"
)

type Exchange struct {
	SessionID     string
	CodeChallenge string
}

type ExchangeStore interface {
	Save(ctx context.Context, code string, exchange Exchange) error
	Consume(ctx context.Context, code string) (*Exchange, error)
}

type ExchangeCacheStore struct {
	mu    sync.Mutex
	cache otter.Cache[string, Exchange]
}

func NewExchangeCacheStore(expiration time.Duration) *ExchangeCacheStore {
	cache := otter.Must(&otter.Options[string, Exchange]{
		ExpiryCalculator: otter.ExpiryCreating[string, Exchange](expiration),
	})

	return &ExchangeCacheStore{cache: *cache}
}

func (s *ExchangeCacheStore) Save(_ context.Context, code string, exchange Exchange) error {
	if code == "" {
		return fmt.Errorf("exchange code must not be empty")
	}
	if exchange.SessionID == "" {
		return fmt.Errorf("exchange session ID must not be empty")
	}
	if exchange.CodeChallenge == "" {
		return fmt.Errorf("exchange code challenge must not be empty")
	}

	s.cache.Set(code, exchange)
	return nil
}

func (s *ExchangeCacheStore) Consume(_ context.Context, code string) (*Exchange, error) {
	if code == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	exchange, ok := s.cache.GetIfPresent(code)
	if !ok {
		return nil, nil
	}

	s.cache.Invalidate(code)
	return &exchange, nil
}
