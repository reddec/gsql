# Generic SQL

[![Go Reference](https://pkg.go.dev/badge/github.com/reddec/gsql.svg)](https://pkg.go.dev/github.com/reddec/gsql)

Tiny wrapper built around standard Go SQL library and SQLx with generic support.

## Installation

    go get -v github.com/reddec/gsql

## Examples

Schema for the example

```sql
CREATE TABLE book
(
    id     INTEGER NOT NULL PRIMARY KEY,
    title  TEXT    NOT NULL,
    author TEXT,
    year   INTEGER NOT NULL
)
```

Type in Go

```go
type Book struct {
    ID     int64
    Title  string
    Author string
    Year   int
}
```

Initialization

```go
// ...
conn, err := sqlx.Open("sqlite", "database.sqlite")
// ...
```

### Get

Get single book by ID

```go
book, err := gsql.Get[Book](ctx, conn, "SELECT * FROM book WHERE id = ?", 1234)
```

### List

List books by author

```go
books, err := gsql.List[Book](ctx, conn, "SELECT * FROM book WHERE author = ?", "O'Really")
```

### Iterate

Iterate each book one by one

```go
iter := gsql.Iterate[Book](ctx, conn, "SELECT * FROM book LIMIT ?", 100)
defer iter.Close()
for iter.Next() {
    book, err := iter.Get()
    // ...
}
if iter.Err() != nil {
    panic(iter.Err()) // use normal handling in real application
}
```

### LazyGet

Save query which returns one object and execute it later

```go
getBooks := gsql.LazyGet[Book](conn, "SELECT * FROM book WHERE id = ?", 123)
// ...
book, err := getBooks(ctx) // can be called many times
```

### LazyList

Save query which returns multiple objects and execute it later

```go
listBooks := gsql.LazyList[Book](conn, "SELECT * FROM book")
// ...
books, err := listBooks(ctx) // can be called many times
```

### CachedGet

Save query which returns one object and execute it later. Once query executed, result will be cached until `Invalidate`
or `Refresh` called.

```go
cache := gsql.CachedGet[Book](conn, "SELECT * FROM book WHERE id = ?", 123)
book, err := cache.Get(ctx) // first time it will execute the query
//...
book2, err := cache.Get(ctx) // second time it will return cached information
//...
cache.Invalidate() // reset cache, the following Get will again execute the query
```

### CachedList

Save query which returns multiple objects and execute it later. Once query executed, result will be cached
until `Invalidate` or `Refresh` called.

```go
cache := gsql.CachedList[Book](conn, "SELECT * FROM book WHERE id = ?", 123)
books, err := cache.Get(ctx) // first time it will execute the query
//...
books2, err := cache.Get(ctx) // second time it will return cached information
//...
cache.Invalidate() // reset cache, the following Get will again execute the query
```
