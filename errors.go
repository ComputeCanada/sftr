package main

import (
	"log"
	"os"
)

func check(ec int, e error) {
	if e != nil {
		log.Fatal(e)
		os.Exit(ec)
	}
}

func debug(s string, a ...interface{}) {
	//log.Printf(s, a...)
}

func info(s string, a ...interface{}) {
	log.Printf(s, a...)
}

func fatal(ec int, s string, a ...interface{}) {
	log.Fatalf(s, a...)
	os.Exit(ec)
}
