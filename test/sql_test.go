package test_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/reddec/gsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const initSQL = `
CREATE TABLE book (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT,
    year INTEGER NOT NULL,
	metadata JSON NOT NULL
);

INSERT INTO book (title, author, year,  metadata) VALUES 
                                           ('Demo','reddec', 2022, '{"zip": 1234}'),
                                           ('Example','Rob Pike', 1980, '{"zip": 5678}'),
                                           ('C','K&R', 1970, '{"zip": 91011}');
`

type Metadata struct {
	Zip int
}

type Book struct {
	ID       int64
	Title    string
	Author   string
	Year     int
	Metadata gsql.JSON[Metadata]
}

// valid anserts
var (
	recordReddec = Book{ID: 1, Title: "Demo", Author: "reddec", Year: 2022, Metadata: gsql.AsJSON(Metadata{Zip: 1234})}
	recordPike   = Book{ID: 2, Title: "Example", Author: "Rob Pike", Year: 1980, Metadata: gsql.AsJSON(Metadata{Zip: 5678})}
	recordKnR    = Book{ID: 3, Title: "C", Author: "K&R", Year: 1970, Metadata: gsql.AsJSON(Metadata{Zip: 91011})}
	records      = []Book{recordReddec, recordPike, recordKnR}
)

func TestGet(t *testing.T) {
	ctx := context.Background()

	conn, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(initSQL)
	require.NoError(t, err)

	book, err := gsql.Get[Book](ctx, conn, "SELECT * FROM book WHERE title = ?", recordReddec.Title)
	require.NoError(t, err)

	assert.Equal(t, recordReddec, book)
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

	assert.Equal(t, records, list)
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

	assert.Equal(t, records, list)
}

func TestCacheGet(t *testing.T) {
	ctx := context.Background()

	conn, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(initSQL)
	require.NoError(t, err)

	cache := gsql.CachedGet[Book](conn, "SELECT * FROM book WHERE title = ?", recordReddec.Title)

	book, err := cache.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, recordReddec, book)

	_, err = conn.ExecContext(ctx, "UPDATE book SET year = ? WHERE id = 1", recordReddec.Year)
	require.NoError(t, err)

	book, err = cache.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, recordReddec, book)

	cache.Invalidate()

	book, err = cache.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, recordReddec, book)
}

func TestInsertJSON(t *testing.T) {
	ctx := context.Background()

	conn, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(initSQL)
	require.NoError(t, err)

	var book = Book{
		ID:     4,
		Title:  "fake",
		Author: "Storyteller",
		Metadata: gsql.AsJSON(Metadata{
			Zip: 987654,
		}),
	}

	_, err = conn.Exec("INSERT INTO book (title, author, year, metadata) VALUES (?, ?, ?, ?)", book.Title, book.Author, book.Year, book.Metadata)
	require.NoError(t, err)

	// snaity check for pointer
	_, err = conn.Exec("INSERT INTO book (title, author, year, metadata) VALUES (?, ?, ?, ?)", book.Title, book.Author, book.Year, &book.Metadata)
	require.NoError(t, err)

	saved, err := gsql.Get[Book](ctx, conn, "SELECT * FROM book WHERE id = 4")
	require.NoError(t, err)

	assert.Equal(t, book, saved)
}
