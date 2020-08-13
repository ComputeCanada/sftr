package main

import (
  "path/filepath"
)

func string_occurs_in_array(s string, a []string) bool {
  for _, c := range a {
    if c == s {
      return true
    }
  }
  return false
}


func string_matches_glob_in_array(s string, a[]string) bool {
  for _, c := range a {
    matched, err := filepath.Match(c, s)
    check(ERR_CONFIGURATION, err)
    if matched {
      return true
    }
  }
  return false
}
