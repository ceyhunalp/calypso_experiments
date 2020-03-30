package main

import (
	// Service needs to be imported here to be instantiated.
	_ "github.com/ceyhunalp/calypso_experiments/semi_centralized/service"
	"go.dedis.ch/onet/v3/simul"
)

func main() {
	simul.Start()
}
