package collections_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/collections"
	"github.com/stretchr/testify/assert"
)

func TestCounter_MostCommon(t *testing.T) {
	tests := []struct {
		name    string
		counter *collections.Counter[string]
		count   int
		want    []collections.CounterItem[string]
	}{
		{
			name:    "nil counter",
			counter: nil,
			count:   3,
			want:    nil,
		},
		{
			name:    "empty counter",
			counter: collections.NewCounter[string](),
			count:   3,
			want:    nil,
		},
		{
			name:    "single item",
			counter: collections.NewCounter("a"),
			count:   1,
			want:    []collections.CounterItem[string]{{Item: "a", Count: 1}},
		},
		{
			name:    "multiple items with ties",
			counter: collections.NewCounter("a", "b", "a", "c", "b"),
			count:   2,
			want:    []collections.CounterItem[string]{{Item: "a", Count: 2}, {Item: "b", Count: 2}},
		},
		{
			name:    "count exceeds unique items",
			counter: collections.NewCounter("a", "b", "a"),
			count:   5,
			want:    []collections.CounterItem[string]{{Item: "a", Count: 2}, {Item: "b", Count: 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.counter.MostCommon(tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}
