package main

import (
	"bytes"
	"errors"
	"github.com/BurntSushi/toml"
	//"github.com/ceyhunalp/centralized_calypso/simple"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
	"time"
)

/*
 * Defines the simulation for the service-template
 */

type ByzcoinData struct {
	Signer darc.Signer
	Roster *onet.Roster
	Cl     *byzcoin.Client
	GMsg   *byzcoin.CreateGenesisBlock
	GDarc  *darc.Darc
	Csr    *byzcoin.CreateGenesisBlockResponse
}

func init() {
	onet.SimulationRegister("Calypso", NewCalypsoService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewCalypsoService(config string) (onet.Simulation, error) {
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

func setupDarcs() (darc.Signer, darc.Signer, *darc.Darc, error) {
	writer := darc.NewSignerEd25519(nil, nil)
	reader := darc.NewSignerEd25519(nil, nil)

	writeDarc := darc.NewDarc(darc.InitRules([]darc.Identity{writer.Identity()},
		[]darc.Identity{writer.Identity()}), []byte("Writer"))
	writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractWriteID),
		expression.InitOrExpr(writer.Identity().String()))
	writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractReadID),
		expression.InitOrExpr(reader.Identity().String()))
	return writer, reader, writeDarc, nil
}

func setupByzcoin(r *onet.Roster) (*ByzcoinData, error) {
	//func setupByzcoin(r *onet.Roster, admin darc.Signer) (*ByzcoinData, error) {
	var err error
	byzd := &ByzcoinData{}
	byzd.Signer = darc.NewSignerEd25519(nil, nil)
	//byzd.Signer = admin
	byzd.GMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + byzcoin.ContractDarcID, "spawn:" + calypso.ContractWriteID, "spawn:" + calypso.ContractReadID}, byzd.Signer.Identity())
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	// TODO: 3-4 seconds block interval
	//byzd.GMsg.BlockInterval = 2 * time.Second
	byzd.GMsg.BlockInterval = 500 * time.Millisecond
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	return byzd, nil
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", s.Rounds)

	//Create a Calypso Client (Byzcoin + Onet)
	//admin := darc.NewSignerEd25519(nil, nil)
	//byzd, err := setupByzcoin(config.Roster, admin)
	byzd, err := setupByzcoin(config.Roster)
	if err != nil {
		return err
	}
	calypsoClient := calypso.NewClient(byzd.Cl)
	ltsReply, err := calypsoClient.CreateLTS()
	if err != nil {
		return err
	}

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		//setDarcs := monitor.NewTimeMeasure("setupDarcs")
		writer, reader, writeDarc, err := setupDarcs()
		if err != nil {
			return err
		}
		//setDarcs.Record()

		_, err = calypsoClient.SpawnDarc(byzd.Signer, *byzd.GDarc, *writeDarc, 0)
		//_, err = calypsoClient.SpawnDarc(admin, *byzd.GDarc, *writeDarc, 0)
		if err != nil {
			return err
		}
		var key [16]byte
		random.Bytes(key[:], random.New())

		writeData := calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarc.GetBaseID(), ltsReply.X, key[:])

		awm := monitor.NewTimeMeasure("add_write_txn")
		writeTxn, err := calypsoClient.AddWrite(writeData, writer, *writeDarc, 2)
		awm.Record()
		if err != nil {
			return err
		}
		log.Info("Write txn instanceID: ", writeTxn.InstanceID)

		wgp := monitor.NewTimeMeasure("write_get_proof")
		wrProofResponse, err := byzd.Cl.GetProof(writeTxn.InstanceID.Slice())
		if err != nil {
			return err
		}
		wrProof := wrProofResponse.Proof
		if !wrProof.InclusionProof.Match() {
			return errors.New("Write inclusion proof does not match")
		}
		wgp.Record()

		arm := monitor.NewTimeMeasure("add_read_txn")
		readTxn, err := calypsoClient.AddRead(&wrProof, reader, *writeDarc, 2)
		arm.Record()
		if err != nil {
			return err
		}

		rgp := monitor.NewTimeMeasure("read_get_proof")
		rProofResponse, err := byzd.Cl.GetProof(readTxn.InstanceID.Slice())
		if err != nil {
			return err
		}
		rProof := rProofResponse.Proof
		if !rProof.InclusionProof.Match() {
			return errors.New("Read inclusion proof does not match")
		}
		rgp.Record()

		dkm := monitor.NewTimeMeasure("decrypt_key")
		dk, err := calypsoClient.DecryptKey(&calypso.DecryptKey{Read: rProof, Write: wrProof})
		if err != nil {
			return err
		}
		if !dk.X.Equal(ltsReply.X) {
			return errors.New("Points not same")
		}
		dkm.Record()

		decode := monitor.NewTimeMeasure("decode_key")
		decodedKey, err := calypso.DecodeKey(cothority.Suite, ltsReply.X, dk.Cs, dk.XhatEnc, reader.Ed25519.Secret)
		decode.Record()
		if err != nil {
			return err
		}

		log.Info("Keys are equal: ", bytes.Equal(decodedKey, key[:]))

	}
	return nil
}
