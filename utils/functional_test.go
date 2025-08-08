package utils

import (
	"testing"
)

func TestMap(t *testing.T) {
	input := []int{1, 2, 3}
	output := Map(input, func(x int) int {
		return x * 2
	})

	expected := []int{2, 4, 6}
	for i, v := range expected {
		if output[i] != v {
			t.Errorf("expected %d, got %d", v, output[i])
		}
	}
}

func TestFilterMap(t *testing.T) {
	input := [][]string{
		{"https://link1.com"},
		{""},
		{"https://link2.com"},
	}

	result := FilterMap(input, func(row []string) (string, bool) {
		if len(row) == 0 || row[0] == "" {
			return "", false
		}
		return row[0], true
	})

	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}
