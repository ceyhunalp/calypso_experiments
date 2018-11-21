package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/ceyhunalp/centralized_calypso/simple"
	"github.com/ceyhunalp/centralized_calypso/util"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

const DATA_SIZE = 1024 * 1024

func init() {
	onet.SimulationRegister("SimpleCalypso", NewSimpleCalypsoService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	BatchSize int
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

	wdList := make([]*util.WriteData, s.BatchSize)
	writeTxnList := make([]*simple.TransactionReply, s.BatchSize)
	readTxnList := make([]*simple.TransactionReply, s.BatchSize)
	wrProofList := make([]*byzcoin.Proof, s.BatchSize)
	readProofList := make([]*byzcoin.Proof, s.BatchSize)
	//decReqList := make([]*simpServ.DecryptReply, s.BatchSize)

	log.Info("Roster size is:", len(config.Roster.List))

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		byzd, err := simple.SetupByzcoin(config.Roster)
		if err != nil {
			log.Errorf("Setting up Byzcoin failed: %v", err)
			return err
		}

		writer, reader, wDarc, err := simple.SetupDarcs()
		if err != nil {
			return err
		}

		_, err = byzd.SpawnDarc(*wDarc, 4)
		if err != nil {
			return err
		}

		for i := 0; i < s.BatchSize; i++ {
			data := make([]byte, DATA_SIZE)
			for j := 0; j < DATA_SIZE; j++ {
				data[j] = byte(i)
			}
			wdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
			if err != nil {
				return err
			}
		}

		//sed := monitor.NewTimeMeasure("store_enc_data")

		awt := monitor.NewTimeMeasure("AddWriteTxn")
		for i := 0; i < s.BatchSize; i++ {
			err = simple.StoreEncryptedData(config.Roster, wdList[i])
			if err != nil {
				return err
			}
		}
		for i := 0; i < s.BatchSize; i++ {
			wait := 0
			if i == s.BatchSize-1 {
				wait = 3
			}
			writeTxnList[i], err = byzd.AddWriteTransaction(wdList[i], writer, *wDarc, wait)
			if err != nil {
				return err
			}
		}
		awt.Record()

		wwp := monitor.NewTimeMeasure("WriteGetProof")
		for i := 0; i < s.BatchSize; i++ {
			wrProofResponse, err := byzd.GetProof(writeTxnList[i].InstanceID)
			if err != nil {
				return err
			}
			wrProof := wrProofResponse.Proof
			if !wrProof.InclusionProof.Match() {
				return errors.New("Write inclusion proof does not match")
			}
			wrProofList[i] = &wrProof
		}
		wwp.Record()

		art := monitor.NewTimeMeasure("AddReadTxn")
		for i := 0; i < s.BatchSize; i++ {
			wait := 0
			if i == s.BatchSize-1 {
				wait = 3
			}
			readTxnList[i], err = byzd.AddReadTransaction(wrProofList[i], reader, *wDarc, wait)
			if err != nil {
				return err
			}
		}
		art.Record()

		rwp := monitor.NewTimeMeasure("ReadGetProof")
		for i := 0; i < s.BatchSize; i++ {
			rProofResponse, err := byzd.GetProof(readTxnList[i].InstanceID)
			if err != nil {
				return err
			}
			rProof := rProofResponse.Proof
			if !rProof.InclusionProof.Match() {
				return errors.New("Read inclusion proof does not match")
			}
			readProofList[i] = &rProof
		}
		rwp.Record()

		decReq := monitor.NewTimeMeasure("DecRequest")
		for i := 0; i < s.BatchSize; i++ {
			dr, err := byzd.DecryptRequest(config.Roster, wrProofList[i], readProofList[i], wdList[i].StoredKey, reader.Ed25519.Secret)
			//dr, err := byzd.DecryptRequest(config.Roster, wrProofList[i], readProofList[i], wd.StoredKey, reader.Ed25519.Secret)
			if err != nil {
				return err
			}

			//dr := decReqList[i]
			_, err = util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
			if err != nil {
				return err
			}
			//log.Info("Data recovered: ", bytes.Compare(recvData, data))
		}
		decReq.Record()

		//rd.Record()

		//awt := monitor.NewTimeMeasure("add_write_txn")
		//writeTxn, err := byzd.AddWriteTransaction(wd, writer, *wDarc, 3)
		//awt.Record()
		//if err != nil {
		//return err
		//}

		//wwp := monitor.NewTimeMeasure("write_get_proof")
		//wrProofResponse, err := byzd.GetProof(writeTxn.InstanceID)
		//if err != nil {
		//return err
		//}
		//wrProof := wrProofResponse.Proof
		//if !wrProof.InclusionProof.Match() {
		//return errors.New("Write inclusion proof does not match")
		//}
		//wwp.Record()

		//art := monitor.NewTimeMeasure("add_read_txn")
		//readTxn, err := byzd.AddReadTransaction(&wrProof, reader, *wDarc, 3)
		//art.Record()
		//if err != nil {
		//return err
		//}

		//rwp := monitor.NewTimeMeasure("read_get_proof")
		//rProofResponse, err := byzd.GetProof(readTxn.InstanceID)
		//if err != nil {
		//return err
		//}
		//rProof := rProofResponse.Proof
		//if !rProof.InclusionProof.Match() {
		//return errors.New("Read inclusion proof does not match")
		//}
		//rwp.Record()

		//decReq := monitor.NewTimeMeasure("dec_req")
		//dr, err := byzd.DecryptRequest(config.Roster, &wrProof, &rProof, wd.StoredKey, reader.Ed25519.Secret)
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
	}
	return nil
}
