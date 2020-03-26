package main

import (
	// Service needs to be imported here to be instantiated.
	//_ "github.com/ceyhunalp/calypso_experiments/semi_centralized/service"
	_ "github.com/ceyhunalp/calypso_experiments/tournament_lottery/service"
	"go.dedis.ch/onet/simul"
)

func main() {
	simul.Start()
}
