package gsql_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/reddec/gsql/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const initSQL = `
CREATE TABLE book (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT,
    year INTEGER NOT NULL
);

INSERT INTO book (title, author, year) VALUES 
                                           ('Demo','reddec', 2022),
                                           ('Example','Rob Pike', 1980),
                                           ('C','K&R', 1970);
`

type Book struct {
	ID     int64
	Title  string
	Author string
	Year   int
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	conn, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(initSQL)
	require.NoError(t, err)

	book, err := gsql.Get[Book](ctx, conn, "SELECT * FROM book WHERE title = ?", "Demo")
	require.NoError(t, err)

	assert.Equal(t, Book{
		ID:     1,
		Title:  "Demo",
		Author: "reddec",
		Year:   2022,
	}, book)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	conn, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(initSQL)
	require.NoError(t, err)

	list, err := gsql.List[Book](ctx, conn, "SELECT * FROM book ORDER BY id")
	require.NoError(t, err)

	assert.Equal(t, []Book{
		{ID: 1, Title: "Demo", Author: "reddec", Year: 2022},
		{ID: 2, Title: "Example", Author: "Rob Pike", Year: 1980},
		{ID: 3, Title: "C", Author: "K&R", Year: 1970},
	}, list)
}

func TestIterator(t *testing.T) {
	ctx := context.Background()

	conn, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(initSQL)
	require.NoError(t, err)

	list, err := gsql.Iterate[Book](ctx, conn, "SELECT * FROM book ORDER BY id").Collect()
	require.NoError(t, err)

	assert.Equal(t, []Book{
		{ID: 1, Title: "Demo", Author: "reddec", Year: 2022},
		{ID: 2, Title: "Example", Author: "Rob Pike", Year: 1980},
		{ID: 3, Title: "C", Author: "K&R", Year: 1970},
	}, list)
}

func TestCacheGet(t *testing.T) {
	ctx := context.Background()

	conn, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(initSQL)
	require.NoError(t, err)

	cache := gsql.CachedGet[Book](conn, "SELECT * FROM book WHERE title = ?", "Demo")

	book, err := cache.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, Book{
		ID:     1,
		Title:  "Demo",
		Author: "reddec",
		Year:   2022,
	}, book)

	_, err = conn.ExecContext(ctx, "UPDATE book SET year = ? WHERE id = 1", 2023)
	require.NoError(t, err)

	book, err = cache.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, Book{
		ID:     1,
		Title:  "Demo",
		Author: "reddec",
		Year:   2022,
	}, book)

	cache.Invalidate()

	book, err = cache.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, Book{
		ID:     1,
		Title:  "Demo",
		Author: "reddec",
		Year:   2023,
	}, book)
}
