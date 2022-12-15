package gsql

import (
	"context"
	"database/sql"
	"sync"

	"github.com/jmoiron/sqlx"
)

type Selector interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Get single result or return an error.
func Get[T any](ctx context.Context, db sqlx.QueryerContext, query string, args ...any) (T, error) {
	var result T
	return result, sqlx.GetContext(ctx, db, &result, query, args...)
}

// List creates slice of results based on SQL query. In case of zero results it will return non-nil empty slice.
func List[T any](ctx context.Context, db sqlx.QueryerContext, query string, args ...any) ([]T, error) {
	var results []T
	results = make([]T, 0)

	return results, sqlx.SelectContext(ctx, db, &results, query, args...)
}

// LazyGet generates closure which wraps prepared query with arguments and executes query with single result on-demand.
// The function doesn't cache anything, use [Cache] for lazy cached queries.
func LazyGet[T any](db sqlx.QueryerContext, query string, args ...any) func(ctx context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		return Get[T](ctx, db, query, args...)
	}
}

// LazyList generates closure which wraps prepared query with arguments and executes query on-demand.
// The function doesn't cache anything, use [Cache] for lazy cached queries.
func LazyList[T any](db sqlx.QueryerContext, query string, args ...any) func(ctx context.Context) ([]T, error) {
	return func(ctx context.Context) ([]T, error) {
		return List[T](ctx, db, query, args...)
	}
}

// Cache stores data and factory to get data.
type Cache[T any] struct {
	lock    sync.RWMutex
	data    T
	valid   bool
	factory func(ctx context.Context) (T, error)
}

// NewCache creates new concurrent-safe cache for internal data.
func NewCache[T any](factory func(ctx context.Context) (T, error)) *Cache[T] {
	return &Cache[T]{factory: factory}
}

// Get content from cache or from storage. Once data fetched, it will be stored internally.
func (ct *Cache[T]) Get(ctx context.Context) (T, error) {
	ct.lock.RLock()
	valid := ct.valid
	data := ct.data
	ct.lock.RUnlock()
	if valid {
		return data, nil
	}
	ct.lock.Lock()
	defer ct.lock.Unlock()
	if ct.valid {
		return ct.data, nil
	}

	value, err := ct.factory(ctx)
	if err != nil {
		return ct.data, err
	}
	ct.data = value
	ct.valid = true
	return ct.data, nil
}

// Refresh cache regardless of validity.
func (ct *Cache[T]) Refresh(ctx context.Context) error {
	ct.lock.Lock()
	defer ct.lock.Unlock()
	value, err := ct.factory(ctx)
	if err != nil {
		return err
	}
	ct.data = value
	ct.valid = true
	return nil
}

// Invalidate cache and cause refresh on next [Cache.Get] operation.
func (ct *Cache[T]) Invalidate() {
	ct.lock.Lock()
	defer ct.lock.Unlock()
	ct.valid = false
}

// CachedGet is alias of [NewCache]([LazyGet]) and provides cached information from database.
func CachedGet[T any](db sqlx.QueryerContext, query string, args ...any) *Cache[T] {
	return NewCache(LazyGet[T](db, query, args...))
}

// CachedList is alias of [NewCache]([LazyList]) and provides cached information from database.
func CachedList[T any](db sqlx.QueryerContext, query string, args ...any) *Cache[[]T] {
	return NewCache(LazyList[T](db, query, args...))
}

// Iterate over multiple results and automatically scans each row to the provided type.
func Iterate[T any](ctx context.Context, db sqlx.QueryerContext, query string, args ...any) *Iterator[T] {
	rows, err := db.QueryxContext(ctx, query, args...)
	return &Iterator[T]{err: err, rows: rows}
}

// Rows is a simple wrapper around [sqlx.Rows] which is automatically scans row to the provided type.
func Rows[T any](rows *sqlx.Rows) *Iterator[T] {
	return &Iterator[T]{rows: rows}
}

// Iterator is typed wrapper around [sql.Rows] which is automatically scans row to the provided type.
type Iterator[T any] struct {
	err  error
	rows *sqlx.Rows
}

// Next is reads next row and returns true if data is available.
func (it *Iterator[T]) Next() bool {
	if it.err != nil {
		return false
	}
	return it.rows.Next()
}

// Err returns an error from the rows or from the initial query.
func (it *Iterator[T]) Err() error {
	if it.err != nil {
		return it.err
	}
	return it.rows.Err()
}

// Close database cursor and allocated resources.
func (it *Iterator[T]) Close() error {
	return it.rows.Close()
}

// Get row and scan data to the type.
func (it *Iterator[T]) Get() (T, error) {
	var result T
	if err := it.rows.Err(); err != nil {
		return result, err
	}

	err := it.rows.StructScan(&result)
	return result, err
}

// Collect all results as slice.
func (it *Iterator[T]) Collect() ([]T, error) {
	var result []T
	for it.Next() {
		r, err := it.Get()
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, it.Err()
}
