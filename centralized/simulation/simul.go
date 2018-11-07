package main

import (
	// Service needs to be imported here to be instantiated.
	_ "github.com/ceyhunalp/centralized_calypso/centralized/service"
	"github.com/dedis/onet/simul"
)

func main() {
	simul.Start()
}
