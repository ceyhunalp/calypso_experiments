package main

import (
	"testing"

	"os"

	"go.dedis.ch/onet/log"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestRun(t *testing.T) {
	os.Args = []string{os.Args[0], "--help"}
	main()
}
