package sorting

import (
	"math/rand"
	"slices"
	"sort"
	"testing"
)

func TestInsertionSort(t *testing.T) {
	check := func(input []int) {
		t.Helper()
		a := slices.Clone(input)
		want := slices.Clone(input)
		sort.Ints(want)
		InsertionSort(a)
		if !slices.Equal(a, want) {
			t.Fatalf("sort(%v) = %v, want %v", input, a, want)
		}
	}
	check(nil)
	check([]int{})
	max := int(^uint(0) >> 1)
	check([]int{max, -max - 1, 0, max, -1, -max - 1})
	// All 3^0 + ... + 3^7 = 3280 arrays, including all duplicate patterns.
	var enumerate func([]int, int)
	enumerate = func(a []int, n int) {
		if n == 0 {
			check(a)
			return
		}
		for v := -1; v <= 1; v++ {
			enumerate(append(a, v), n-1)
		}
	}
	for n := 0; n <= 7; n++ {
		enumerate(nil, n)
	}
	r := rand.New(rand.NewSource(42))
	for trial := 0; trial < 1000; trial++ {
		a := make([]int, r.Intn(100))
		for i := range a {
			a[i] = r.Intn(200) - 100
		}
		check(a)
	}
	backing := []int{91, 3, 1, 2, 92}
	InsertionSort(backing[1:4])
	if !slices.Equal(backing, []int{91, 1, 2, 3, 92}) {
		t.Fatal("modified data outside the slice", backing)
	}
}

func TestGoSemanticExamples(t *testing.T) {
	a := []int{0}
	if got := CallOrder(a); got != 12 || a[0] != 2 {
		t.Fatal(got, a)
	}
	if ShortCircuit(nil) {
		t.Fatal("short circuit")
	}
	a = []int{10, 11}
	if ParallelAssignment(a) != 1 || !slices.Equal(a, []int{7, 11}) {
		t.Fatal(a)
	}
	for n := 3; n <= 100; n++ {
		if ContinuePost(n) != n-3 {
			t.Fatal("continue", n)
		}
	}
	for n := 0; n <= 100; n++ {
		if BreakLoop(n) != n {
			t.Fatal("break", n)
		}
	}
	for _, n := range []int{-9, 0, 8} {
		if Abs(n) < 0 {
			t.Fatal("return", n)
		}
	}
	if Shadow() != 3 {
		t.Fatal("shadowing")
	}
}
