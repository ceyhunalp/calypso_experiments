package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	fc "github.com/ceyhunalp/calypso_experiments/fully_centralized"
	"github.com/ceyhunalp/calypso_experiments/util"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

func runFullyCentralizedCalypso(roster *onet.Roster, serverKey kyber.Point, data []byte) error {
	//data := []byte("On Wisconsin!")
	// Reader keys
	rSk := cothority.Suite.Scalar().Pick(cothority.Suite.RandomStream())
	rPk := cothority.Suite.Point().Mul(rSk, nil)

	wd, err := util.CreateWriteData(data, rPk, serverKey, false)
	if err != nil {
		os.Exit(1)
	}

	// Create write transaction
	wd, err = fc.CreateWriteTxn(roster, wd)
	//wID, err := CreateWriteTxn(roster, encData, k, c, rPk)
	if err != nil {
		os.Exit(1)
	}
	fmt.Println("Write transaction success:", wd.StoredKey)

	// Create read transaction
	kRead, cRead, err := fc.CreateReadTxn(roster, wd.StoredKey, rSk)
	if err != nil {
		os.Exit(1)
	}

	recvData, err := util.RecoverData(wd.Data, rSk, kRead, cRead)
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(string(recvData[:]))
	return nil
}

//func getServerKey(pkPtr *string) (kyber.Point, error) {
//return util.GetServerKey(pkPtr)
//}

//func readRoster(filePtr *string) (*onet.Roster, error) {
//return util.ReadRoster(filePtr)
//}

func main() {
	pkPtr := flag.String("p", "", "pk.txt file")
	dbgPtr := flag.Int("d", 0, "debug level")
	filePtr := flag.String("r", "", "roster.toml file")
	flag.Parse()
	log.SetDebugVisible(*dbgPtr)

	roster, err := util.ReadRoster(filePtr)
	if err != nil {
		os.Exit(1)
	}
	serverKey, err := util.GetServerKey(pkPtr)
	if err != nil {
		os.Exit(1)
	}
	baseStr := "On Wisconsin! -- "
	for i := 0; i < 70; i++ {
		err = runFullyCentralizedCalypso(roster, serverKey, []byte(strings.Join([]string{baseStr, strconv.Itoa(i + 1)}, "")))
		if err != nil {
			log.Errorf("Run FullyCentralizedCalypso failed: %v", err)
		}
	}
	/*
	 *        // Try to create duplicate write transaction
	 *        _, err = CreateWriteTxn(roster, encData, k, c, rPk)
	 *        if err != nil {
	 *                log.Errorf("Write transaction failed: %v", err)
	 *        }
	 *
	 *        // Create unauthorized reader
	 *        newSk := cothority.Suite.Scalar().Pick(cothority.Suite.RandomStream())
	 *        _ = cothority.Suite.Point().Mul(newSk, nil)
	 *
	 *        // Create read transaction with unauthorized reader
	 *        _, _, err = CreateReadTxn(roster, wID, newSk)
	 *        if err != nil {
	 *                log.Errorf("Read transaction failed: %v", err)
	 *                os.Exit(1)
	 *        }
	 */
}
