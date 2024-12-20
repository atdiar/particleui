package ui

import (
	"reflect"
	"testing"
)

// Rest of the code from above

// TestMyersDiff tests the MyersDiff function
func TestMyersDiff(t *testing.T) {
	a := []string{"a", "b", "c", "e", "f", "h"}
	b := []string{"b", "c", "d", "e", "f", "g"}

	expected := []EditOp{
		{Operation: "Remove", ElementID: "a", Index: 0},
		{Operation: "Insert", ElementID: "d", Index: 2},
		{Operation: "Remove", ElementID: "h", Index: 5},
		{Operation: "Insert", ElementID: "g", Index: 5},
	}

	got := MyersDiff(a, b)

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Expected %+v, got %+v", expected, got)
	}
}

// BenchmarkMyersDiff benchmarks the MyersDiff function
func BenchmarkMyersDiff(b *testing.B) {
	a := []string{"a", "b", "c", "e", "f", "h"}
	a2 := []string{"b", "c", "d", "e", "f", "g"}

	for i := 0; i < b.N; i++ {
		MyersDiff(a, a2)
	}
}

// BenchmarkNaiveDiff benchmarks a naive diffing method
func BenchmarkNaiveDiff(b *testing.B) {
	a := []string{"a", "b", "c", "e", "f", "h"}
	a2 := []string{"b", "c", "d", "e", "f", "g"}

	for i := 0; i < b.N; i++ {
		// Naive method: clear and append
		a = []string{}
		for _, element := range a2 {
			a = append(a, element)
		}
	}
}

/*



 */
