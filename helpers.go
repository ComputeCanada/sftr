package main

func sina(s string, a []string) bool {
  for _, c := range a {
    if c == s {
      return true
    }
  }
  return false
}


