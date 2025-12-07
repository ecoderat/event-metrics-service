package mockclickhousebatch

import (
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/mock"
)

type Batch struct {
	mock.Mock
}

var _ driver.Batch = &Batch{}

func (m *Batch) Append(args ...any) error {
	callArgs := []any{}
	callArgs = append(callArgs, args...)
	return m.Called(callArgs...).Error(0)
}

func (m *Batch) Send() error {
	mockArgs := m.Called()
	return mockArgs.Error(0)
}

func (m *Batch) Abort() error {
	mockArgs := m.Called()
	return mockArgs.Error(0)
}

func (m *Batch) AppendStruct(v any) error {
	mockArgs := m.Called(v)
	return mockArgs.Error(0)
}

func (m *Batch) Column(id int) driver.BatchColumn {
	mockArgs := m.Called(id)
	if v := mockArgs.Get(0); v != nil {
		if column, ok := v.(driver.BatchColumn); ok {
			return column
		}
	}
	return mockArgs.Get(0).(driver.BatchColumn)
}

func (m *Batch) Flush() error {
	mockArgs := m.Called()
	return mockArgs.Error(0)
}

func (m *Batch) IsSent() bool {
	mockArgs := m.Called()
	return mockArgs.Bool(0)
}

type BatchColumn struct {
	mock.Mock
}

var _ driver.BatchColumn = &BatchColumn{}

func (m *BatchColumn) Append(v any) error {
	mockArgs := m.Called(v)
	return mockArgs.Error(0)
}

func (m *BatchColumn) AppendRow(v any) error {
	mockArgs := m.Called(v)
	return mockArgs.Error(0)
}
