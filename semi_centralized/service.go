package semicentralized

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/ceyhunalp/calypso_experiments/util"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
)

// Used for tests
var templateID onet.ServiceID

func init() {
	var err error
	templateID, err = onet.RegisterNewService(ServiceName, newSemiCentralizedService)
	log.ErrFatal(err)
	network.RegisterMessages(&storage{}, &StoreRequest{}, &StoreReply{}, &DecryptRequest{}, &DecryptReply{})
}

// Service is our template-service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
	db      *SemiCentralizedDB
	storage *storage
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("SemiCentralized")

// storage is used to save our data.
type storage struct {
	//Suite *edwards25519.SuiteEd25519
	sync.Mutex
}

func (s *Service) StoreData(req *StoreRequest) (*StoreReply, error) {
	storedKey, err := s.db.StoreData(req)
	if err != nil {
		return nil, err
	}
	reply := &StoreReply{
		StoredKey: storedKey,
	}
	return reply, nil
}

func (s *Service) Decrypt(req *DecryptRequest) (*DecryptReply, error) {
	sk := s.ServerIdentity().GetPrivate()
	storedData, err := s.db.GetStoredData(req.Key)
	if err != nil {
		return nil, err
	}
	writeTxn, err := verifyDecryptRequest(req, storedData, sk)
	if err != nil {
		log.Errorf("getDecryptedData error: %v", err)
		return nil, err
	}
	k, c, err := reencryptData(writeTxn, sk)
	if err != nil {
		log.Errorf("getDecryptedData error: %v", err)
		return nil, err
	}
	return &DecryptReply{Data: storedData.Data, DataHash: storedData.DataHash, K: k, C: c}, nil
	//return getDecryptedData(req, storedData, sk)
}

func reencryptData(wt *calypso.SemiWrite, sk kyber.Scalar) (kyber.Point, kyber.Point, error) {
	symKey, err := util.ElGamalDecrypt(sk, wt.K, wt.C)
	if err != nil {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, err
	}

	decReader, err := util.AeadOpen(symKey, wt.EncReader)
	if err != nil {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, err
	}

	ok, err := util.CompareKeys(wt.Reader, decReader)
	if err != nil {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, err
	}
	if ok != 0 {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, errors.New("Reader public key does not match")
	}

	k, c, _ := util.ElGamalEncrypt(wt.Reader, symKey)
	return k, c, nil
}

func verifyDecryptRequest(req *DecryptRequest, storedData *StoreRequest, sk kyber.Scalar) (*calypso.SemiWrite, error) {
	log.Lvl2("Re-encrypt the key to the public key of the reader")

	var read calypso.Read
	if err := req.Read.ContractValue(cothority.Suite, calypso.ContractReadID, &read); err != nil {
		log.Errorf("verifyDecryptRequest error: didn't get a read instance " + err.Error())
		return nil, errors.New("didn't get a read instance: " + err.Error())
	}
	var write calypso.SemiWrite
	if err := req.Write.ContractValue(cothority.Suite, calypso.ContractSemiWriteID, &write); err != nil {
		log.Errorf("verifyDecryptRequest error: didn't get a write instance " + err.Error())
		return nil, errors.New("didn't get a write instance: " + err.Error())
	}
	if !read.Write.Equal(byzcoin.NewInstanceID(req.Write.InclusionProof.Key)) {
		log.Errorf("verifyDecryptRequest error: read doesn't point to passed write")
		return nil, errors.New("read doesn't point to passed write")
	}
	if err := req.Read.Verify(req.SCID); err != nil {
		log.Errorf("verifyDecryptRequest error: read proof cannot be verified to come from scID" + err.Error())
		return nil, errors.New("read proof cannot be verified to come from scID: " + err.Error())
	}
	if err := req.Write.Verify(req.SCID); err != nil {
		log.Errorf("verifyDecryptRequest error: write proof cannot be verified to come from scID" + err.Error())
		return nil, errors.New("write proof cannot be verified to come from scID: " + err.Error())
	}

	keyBytes, err := hex.DecodeString(req.Key)
	if err != nil {
		log.Errorf("verifyDecryptRequest error: %v", err)
		return nil, err
	}
	ok := bytes.Compare(keyBytes, storedData.DataHash)
	if ok != 0 {
		log.Errorf("verifyDecryptRequest error: Keys do not match")
		return nil, errors.New("Keys do not match")
	}
	err = schnorr.Verify(cothority.Suite, write.Reader, keyBytes, req.Sig)
	if err != nil {
		log.Errorf("verifyDecryptRequest error: %v", err)
		return nil, err
	}
	return &write, nil
}

// saves all data.
func (s *Service) save() {
	s.storage.Lock()
	defer s.storage.Unlock()
	err := s.Save(storageID, s.storage)
	if err != nil {
		log.Error("Couldn't save data:", err)
	}
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
func (s *Service) tryLoad() error {
	s.storage = &storage{}
	//defer func() {
	//if s.storage.Suite == nil {
	//s.storage.Suite = edwards25519.NewBlakeSHA256Ed25519()
	//}
	//}()
	msg, err := s.Load(storageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}
	var ok bool
	s.storage, ok = msg.(*storage)
	if !ok {
		return errors.New("Data of wrong type")
	}
	return nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newSemiCentralizedService(c *onet.Context) (onet.Service, error) {
	db, bucket := c.GetAdditionalBucket([]byte("semicentralizedtransactions"))
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		db:               NewSemiCentralizedDB(db, bucket),
	}
	if err := s.RegisterHandlers(s.StoreData, s.Decrypt); err != nil {
		return nil, errors.New("Couldn't register messages")
	}
	if err := s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}
