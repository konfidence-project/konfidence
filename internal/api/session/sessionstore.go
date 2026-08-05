package session

import "context"

type Reader interface {
	Get(ctx context.Context, id string) (*Session, error)
}

type Writer interface {
	Save(ctx context.Context, session *Session) (string, error)
	Delete(ctx context.Context, id string) error
}

type Store interface {
	Reader
	Writer
}
