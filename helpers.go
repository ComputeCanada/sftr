package main

import (
	"path/filepath"
)

// Checks whether the given string occurs exactly in the given array.
func string_occurs_in_array(s string, a []string) bool {
	for _, c := range a {
		if c == s {
			return true
		}
	}
	return false
}

// Checks whether a given string (presumably a file path) matches any of the
// glob patterns in the given array, using path/filepath to match.
func string_matches_glob_in_array(s string, a []string) bool {
	for _, c := range a {
		matched, err := filepath.Match(c, s)
		check(ERR_CONFIGURATION, err)
		if matched {
			return true
		}
	}
	return false
}
