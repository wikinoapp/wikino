package testutil

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// QueryCounterはquery.DBTXを包み、実行されたSQLを記録するテスト用のラッパー。
// 本文中のリンク数に比例してクエリが増えていないか (N+1が再発していないか) を、
// 発行回数でテストに固定するために使う。
type QueryCounter struct {
	inner query.DBTX

	mu         sync.Mutex
	statements []string
}

// NewQueryCounterはトランザクションを包むQueryCounterを生成する
func NewQueryCounter(tx *sql.Tx) *QueryCounter {
	return &QueryCounter{inner: tx}
}

// Queriesはこのカウンターを通してSQLを実行するquery.Queriesを返す
func (c *QueryCounter) Queries() *query.Queries {
	return query.New(c)
}

// CountContainingは記録したSQLのうち、substrを含むものの数を返す。
// sqlcが生成するSQLの先頭にはクエリ名のコメントが残るため、substrにはクエリ名を渡せる。
func (c *QueryCounter) CountContaining(substr string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for _, statement := range c.statements {
		if strings.Contains(statement, substr) {
			count++
		}
	}

	return count
}

func (c *QueryCounter) record(sqlText string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statements = append(c.statements, sqlText)
}

// ExecContextはquery.DBTXの実装
func (c *QueryCounter) ExecContext(ctx context.Context, sqlText string, args ...interface{}) (sql.Result, error) {
	c.record(sqlText)

	return c.inner.ExecContext(ctx, sqlText, args...)
}

// PrepareContextはquery.DBTXの実装
func (c *QueryCounter) PrepareContext(ctx context.Context, sqlText string) (*sql.Stmt, error) {
	c.record(sqlText)

	return c.inner.PrepareContext(ctx, sqlText)
}

// QueryContextはquery.DBTXの実装
func (c *QueryCounter) QueryContext(ctx context.Context, sqlText string, args ...interface{}) (*sql.Rows, error) {
	c.record(sqlText)

	return c.inner.QueryContext(ctx, sqlText, args...)
}

// QueryRowContextはquery.DBTXの実装
func (c *QueryCounter) QueryRowContext(ctx context.Context, sqlText string, args ...interface{}) *sql.Row {
	c.record(sqlText)

	return c.inner.QueryRowContext(ctx, sqlText, args...)
}
