package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Key   string
	Value int
}

func TestDeduplicate(t *testing.T) {
	keyOf := func(i testItem) string { return i.Key }

	tests := []struct {
		name     string
		input    []testItem
		expected []testItem
	}{
		{
			name:     "empty slice returns empty",
			input:    []testItem{},
			expected: []testItem{},
		},
		{
			name: "single element",
			input: []testItem{
				{Key: "a", Value: 1},
			},
			expected: []testItem{
				{Key: "a", Value: 1},
			},
		},
		{
			name: "no duplicates preserves order",
			input: []testItem{
				{Key: "a", Value: 1},
				{Key: "b", Value: 2},
				{Key: "c", Value: 3},
			},
			expected: []testItem{
				{Key: "a", Value: 1},
				{Key: "b", Value: 2},
				{Key: "c", Value: 3},
			},
		},
		{
			// Repository Upsert contract: on duplicates, the last version must be
			// persisted to the DB, not the first
			name: "all duplicates keeps last (last-write-wins contract)",
			input: []testItem{
				{Key: "a", Value: 1},
				{Key: "a", Value: 2},
				{Key: "a", Value: 3},
			},
			expected: []testItem{
				{Key: "a", Value: 3},
			},
		},
		{
			name: "mixed duplicates keeps last occurrence of each key",
			input: []testItem{
				{Key: "a", Value: 1},
				{Key: "b", Value: 2},
				{Key: "a", Value: 3},
				{Key: "c", Value: 4},
				{Key: "b", Value: 5},
			},
			expected: []testItem{
				{Key: "a", Value: 3},
				{Key: "c", Value: 4},
				{Key: "b", Value: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicate(tt.input, keyOf)
			require.Len(t, got, len(tt.expected))
			assert.Equal(t, tt.expected, got)
		})
	}
}
