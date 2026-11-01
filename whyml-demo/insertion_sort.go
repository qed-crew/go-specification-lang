package sorting

// Swap's contract exposes the values and frame as well as permutation.
// These are proved from the parallel Go assignment, not assumed.
/*@
requires 0 <= i && i < len(a);
requires 0 <= j && j < len(a);
ensures exchange(\old(a), a, i, j);
ensures a[i] == \old(a[j]);
ensures a[j] == \old(a[i]);
ensures \forall k integer; (0 <= k && k < len(a) && k != i && k != j) ==> a[k] == \old(a[k]);
ensures permut_all(\old(a), a);
*/
func Swap(a []int, i, j int) {
	a[i], a[j] = a[j], a[i]
}

// InsertionSort inserts each new element by swapping it leftward.
// It handles nil/empty slices, duplicate values and negative values.
/*@
ensures sorted(a);
ensures permut_all(\old(a), a);
*/
func InsertionSort(a []int) {
	/*@
	loop invariant 0 <= i && i <= len(a);
	loop invariant sorted_sub(a, 0, i);
	loop invariant permut_all(\old(a), a);
	loop variant len(a) - i;
	*/
	for i := 0; i < len(a); i++ {
		j := i
		/*@
		loop invariant 0 <= j && j <= i && i < len(a);
		loop invariant \forall k, l integer; (0 <= k && k < l && l <= i && l != j) ==> a[k] <= a[l];
		loop invariant permut_all(\old(a), a);
		loop variant j;
		*/
		for j > 0 && a[j-1] > a[j] {
			Swap(a, j-1, j)
			j--
		}
	}
}
