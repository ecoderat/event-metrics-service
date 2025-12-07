package mockclickhouseconnection

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/mock"
)

type Connection struct {
	mock.Mock
}

var _ clickhouse.Conn = &Connection{}

func (m *Connection) Exec(ctx context.Context, query string, args ...any) error {
	callArgs := []any{ctx, query}
	callArgs = append(callArgs, args...)
	return m.Called(callArgs...).Error(0)
}

func (m *Connection) PrepareBatch(ctx context.Context, query string) (driver.Batch, error) {
	mockArgs := m.Called(ctx, query)
	if v := mockArgs.Get(0); v != nil {
		if batch, ok := v.(driver.Batch); ok {
			return batch, mockArgs.Error(1)
		}
	}
	return nil, mockArgs.Error(1)
}

func (m *Connection) AsyncInsert(ctx context.Context, query string, wait bool) error {
	mockArgs := m.Called(ctx, query, wait)
	return mockArgs.Error(0)
}

func (m *Connection) Close() error {
	mockArgs := m.Called()
	return mockArgs.Error(0)
}

func (m *Connection) Contributors() []string {
	mockArgs := m.Called()
	return mockArgs.Get(0).([]string)
}

func (m *Connection) Ping(ctx context.Context) error {
	mockArgs := m.Called(ctx)
	return mockArgs.Error(0)
}

func (m *Connection) ServerVersion() (*driver.ServerVersion, error) {
	mockArgs := m.Called()
	return mockArgs.Get(0).(*clickhouse.ServerVersion), mockArgs.Error(1)
}

func (m *Connection) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	mockArgs := m.Called(ctx, dest, query, args)
	return mockArgs.Error(0)
}

func (m *Connection) Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(driver.Rows), mockArgs.Error(1)
}

func (m *Connection) QueryRow(ctx context.Context, query string, args ...interface{}) driver.Row {
	mockArgs := m.Called(ctx, query, args)
	return mockArgs.Get(0).(driver.Row)
}

func (m *Connection) Stats() driver.Stats {
	mockArgs := m.Called()
	return mockArgs.Get(0).(driver.Stats)
}
