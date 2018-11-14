package main

import (
	"bytes"
	"errors"
	//"fmt"
	"github.com/BurntSushi/toml"
	"github.com/ceyhunalp/centralized_calypso/simple"
	"github.com/ceyhunalp/centralized_calypso/util"
	//"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

const DATA_SIZE = 1024 * 1024
const BATCH_SIZE = 100

func init() {
	onet.SimulationRegister("SimpleCalypso", NewSimpleCalypsoService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimpleCalypsoService(config string) (onet.Simulation, error) {
	es := &SimulationService{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationService) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	s.CreateRoster(sc, hosts, 2000)
	err := s.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Node can be used to initialize each node before it will be run
// by the server. Here we call the 'Node'-method of the
// SimulationBFTree structure which will load the roster- and the
// tree-structure to speed up the first round.
func (s *SimulationService) Node(config *onet.SimulationConfig) error {
	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)
	return s.SimulationBFTree.Node(config)
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	log.Info("Total # of rounds is:", s.Rounds)
	serverPk := config.Roster.Publics()[0]
	size := config.Tree.Size()
	log.Info("Size of the tree:", size)

	//writeTxnList := make([]*simple.TransactionReply, BATCH_SIZE)
	//readTxnList := make([]*simple.TransactionReply, BATCH_SIZE)
	//wrProofList := make([]*byzcoin.Proof, BATCH_SIZE)
	//readProofList := make([]*byzcoin.Proof, BATCH_SIZE)

	data := make([]byte, DATA_SIZE)
	for i := 0; i < DATA_SIZE; i++ {
		data[i] = 'w'
	}

	byzd, err := simple.SetupByzcoin(config.Roster)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		return err
	}

	writer, reader, wDarc, err := simple.SetupDarcs()
	if err != nil {
		return err
	}

	_, err = byzd.SpawnDarc(*wDarc, 0)
	if err != nil {
		return err
	}

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		cwd := monitor.NewTimeMeasure("creat_wr_data")
		wd, err := util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
		cwd.Record()
		if err != nil {
			return err
		}

		sed := monitor.NewTimeMeasure("store_enc_data")
		err = simple.StoreEncryptedData(config.Roster, wd)
		sed.Record()
		if err != nil {
			return err
		}

		//log.Info("Adding writes start")
		//awt := monitor.NewTimeMeasure("add_write_txn")
		//for i := 0; i < BATCH_SIZE-1; i++ {
		//log.Info("In the for loop", i)
		//writeTxnList[i], err = byzd.AddWriteTransaction(wd, writer, *wDarc, 0)
		//if err != nil {
		//return err
		//}
		//}
		//writeTxnList[BATCH_SIZE-1], err = byzd.AddWriteTransaction(wd, writer, *wDarc, 2)
		//if err != nil {
		//return err
		//}
		//awt.Record()
		//log.Info("Adding writes finished")

		//log.Info("Adding reads start")
		//for i := 0; i < BATCH_SIZE; i++ {
		//wwp := monitor.NewTimeMeasure("write_get_proof")
		//wrProofResponse, err := byzd.GetProof(writeTxnList[i].InstanceID)
		//if err != nil {
		//return err
		//}
		//wrProof := wrProofResponse.Proof
		//if !wrProof.InclusionProof.Match() {
		//return errors.New("Write inclusion proof does not match")
		//}
		//wwp.Record()
		//wrProofList[i] = &wrProof
		//}
		//log.Info("Adding reads finished")

		//art := monitor.NewTimeMeasure("add_read_txn")
		//for i := 0; i < BATCH_SIZE-1; i++ {
		//readTxnList[i], err = byzd.AddReadTransaction(wrProofList[i], reader, *wDarc, 0)
		//if err != nil {
		//return err
		//}
		//}
		//readTxnList[BATCH_SIZE-1], err = byzd.AddReadTransaction(wrProofList[BATCH_SIZE-1], reader, *wDarc, 20)
		//if err != nil {
		//return err
		//}
		//art.Record()

		//for i := 0; i < BATCH_SIZE; i++ {
		//rwp := monitor.NewTimeMeasure("read_get_proof")
		//rProofResponse, err := byzd.GetProof(readTxnList[i].InstanceID)
		//if err != nil {
		//return err
		//}
		//rProof := rProofResponse.Proof
		//if !rProof.InclusionProof.Match() {
		//return errors.New("Read inclusion proof does not match")
		//}
		//rwp.Record()
		//readProofList[i] = &rProof
		//}

		//for i := 0; i < BATCH_SIZE; i++ {
		//decReq := monitor.NewTimeMeasure("dec_req")
		//dr, err := byzd.DecryptRequest(config.Roster, wrProofList[i], readProofList[i], wd.StoredKey, reader.Ed25519.Secret)
		//decReq.Record()
		//if err != nil {
		//return err
		//}

		//rd := monitor.NewTimeMeasure("rec_data")
		//recvData, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
		//rd.Record()

		//if err != nil {
		//return err
		//}
		//log.Info("Data recovered: ", bytes.Compare(recvData, data))
		//}

		awt := monitor.NewTimeMeasure("add_write_txn")
		writeTxn, err := byzd.AddWriteTransaction(wd, writer, *wDarc, 3)
		awt.Record()
		if err != nil {
			return err
		}

		wwp := monitor.NewTimeMeasure("write_get_proof")
		wrProofResponse, err := byzd.GetProof(writeTxn.InstanceID)
		if err != nil {
			return err
		}
		wrProof := wrProofResponse.Proof
		if !wrProof.InclusionProof.Match() {
			return errors.New("Write inclusion proof does not match")
		}
		wwp.Record()

		art := monitor.NewTimeMeasure("add_read_txn")
		readTxn, err := byzd.AddReadTransaction(&wrProof, reader, *wDarc, 3)
		art.Record()
		if err != nil {
			return err
		}

		rwp := monitor.NewTimeMeasure("read_get_proof")
		rProofResponse, err := byzd.GetProof(readTxn.InstanceID)
		if err != nil {
			return err
		}
		rProof := rProofResponse.Proof
		if !rProof.InclusionProof.Match() {
			return errors.New("Read inclusion proof does not match")
		}
		rwp.Record()

		decReq := monitor.NewTimeMeasure("dec_req")
		dr, err := byzd.DecryptRequest(config.Roster, &wrProof, &rProof, wd.StoredKey, reader.Ed25519.Secret)
		decReq.Record()
		if err != nil {
			return err
		}

		rd := monitor.NewTimeMeasure("rec_data")
		recvData, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
		rd.Record()

		if err != nil {
			return err
		}
		log.Info("Data recovered: ", bytes.Compare(recvData, data))
	}
	return nil
}
