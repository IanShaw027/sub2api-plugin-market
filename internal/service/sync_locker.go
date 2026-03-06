package service

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"sync"
)

// SyncLocker abstracts the locking mechanism for sync operations.
// PgAdvisoryLocker provides distributed locking via PostgreSQL advisory locks;
// InMemoryLocker provides process-level locking for tests and single-instance deployments.
type SyncLocker interface {
	TryLock(ctx context.Context, key string) (unlock func(), err error)
}

// PgAdvisoryLocker uses PostgreSQL session-level advisory locks.
// Each lock acquires a dedicated connection from the pool, holding it until unlock.
type PgAdvisoryLocker struct {
	db *sql.DB
}

func NewPgAdvisoryLocker(db *sql.DB) *PgAdvisoryLocker {
	return &PgAdvisoryLocker{db: db}
}

func (l *PgAdvisoryLocker) TryLock(ctx context.Context, key string) (func(), error) {
	lockID := hashToInt64(key)

	conn, err := l.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("sync locker: get db connection: %w", err)
	}

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sync locker: advisory lock query: %w", err)
	}

	if !acquired {
		conn.Close()
		return nil, fmt.Errorf("concurrent sync in progress for %s", key)
	}

	return func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockID)
		conn.Close()
	}, nil
}

// InMemoryLocker provides process-level sync locking via sync.Map + sync.Mutex.
// Suitable for single-instance deployments and tests.
type InMemoryLocker struct {
	locks sync.Map
}

func NewInMemoryLocker() *InMemoryLocker {
	return &InMemoryLocker{}
}

func (l *InMemoryLocker) TryLock(_ context.Context, key string) (func(), error) {
	mu := &sync.Mutex{}
	actual, loaded := l.locks.LoadOrStore(key, mu)
	actualMu := actual.(*sync.Mutex)

	if !actualMu.TryLock() {
		return nil, fmt.Errorf("concurrent sync in progress for %s", key)
	}

	return func() {
		actualMu.Unlock()
		if !loaded {
			l.locks.Delete(key)
		}
	}, nil
}

func hashToInt64(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}
