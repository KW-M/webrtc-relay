package util

/* contains
 * checks if a string is present in a slice (aka an array)
 * PARAM s: the list/slice of strings to check
 * PARAM str: the string to check for
 * RETURNS: true if the string is present in the slice, false otherwise
 * from: https://freshman.tech/snippets/go/check-if-slice-contains-element/
 */
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

/* removeMatching (UNTESTED)
 * removes all elements from a slice that match elements from another slice
 * PARAM a: the slice to remove elements from
 * PARAM b: the slice of match elements to remove from a
 * RETURNS: the slice a with all matching elements removed
 */
func RemoveMatching(a []string, b []string) []string {
	newMap := map[string]bool{}

	// mask the elements in b with a map
	for _, value := range b {
		newMap[value] = true
	}

	//
	output := make([]string, 0)
	for _, value := range a {
		if _, exists := newMap[value]; !exists {
			output = append(output, value)
		}
	}

	return output
}
