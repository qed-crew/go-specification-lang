package sorting

/*
@
requires len(a) == 1;
requires a[0] < max_int;
ensures a[0] == \old(a[0]) + 1;
ensures \result == a[0];
*/
func Bump(a []int) int {
	a[0]++
	return a[0]
}

/*
@
requires 0 <= x && x <= 10;
requires 0 <= y && y <= 10;
ensures \result == 10*x + y;
*/
func Pack(x, y int) int { return 10*x + y }

// Both Bump calls mutate the same slice. Go requires them left-to-right.
// A naive WhyML application can reverse their order and return 21 instead.
/*@
requires len(a) == 1;
requires a[0] == 0;
ensures a[0] == 2;
ensures \result == 12;
*/
func CallOrder(a []int) int { return Pack(Bump(a), Bump(a)) }

/*
@
requires len(a) == 0;
ensures \result == \false;
*/
func ShortCircuit(a []int) bool { return len(a) > 0 && a[0] > 0 }

/*
@
requires len(a) >= 2;
ensures a[0] == 7;
ensures a[1] == \old(a[1]);
ensures \result == 1;
*/
func ParallelAssignment(a []int) int {
	i := 0
	i, a[i] = 1, 7
	return i
}

/*
@
requires 3 <= n && n <= 100;
ensures \result == n - 3;
*/
func ContinuePost(n int) int {
	var count int
	/*@
	loop invariant 0 <= i && i <= n;
	loop invariant (i <= 3 && count == 0) || (i > 3 && count == i - 3);
	loop variant n - i;
	*/
	for i := 0; i < n; i++ {
		if i < 3 {
			continue
		}
		count++
	}
	return count
}

/*
@
requires 0 <= n && n <= 100;
ensures \result == n;
*/
func BreakLoop(n int) int {
	i := 0
	/*@
	loop invariant 0 <= i && i <= n;
	loop variant n - i;
	*/
	for {
		if i == n {
			break
		}
		i++
	}
	return i
}

/*
@
requires x > min_int;
ensures \result >= 0;
ensures (x < 0 && \result == -x) || (x >= 0 && \result == x);
*/
func Abs(x int) int {
	if x < 0 {
		return -x
	} else if x == 0 {
		return 0
	}
	return x
}

/*@ ensures \result == 3; */
func Shadow() int {
	x := 3
	{
		x := 8
		x++
	}
	return x
}
