package service

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	"errors"
	"sync"

	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// ServiceName is used for registration on the onet.
const ServiceName = "SemiCentralizedService"

// Used for tests
var templateID onet.ServiceID

func init() {
	log.Print("init in service")
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
	storedData, err := s.db.GetStoredData(req.Key)
	if err != nil {
		return nil, err
	}
	sk := s.ServerIdentity().GetPrivate()
	return getDecryptedData(req, storedData, sk)
}

// saves all data.
func (s *Service) save() {
	log.Print("In save")
	s.storage.Lock()
	defer s.storage.Unlock()
	err := s.Save(storageID, s.storage)
	if err != nil {
		log.Error("Couldn't save data:", err)
	}
	log.Print("Exiting save")
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
func (s *Service) tryLoad() error {
	log.Print("In tryLoad")
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
	log.Print("Exiting tryLoad")
	return nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newSemiCentralizedService(c *onet.Context) (onet.Service, error) {
	log.Print("In NewSemiCentralizedService")
	db, bucket := c.GetAdditionalBucket([]byte("semicentralizedtransactions"))
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		db:               NewSemiCentralizedDB(db, bucket),
	}
	log.Print("Registering handlers")
	if err := s.RegisterHandlers(s.StoreData, s.Decrypt); err != nil {
		return nil, errors.New("Couldn't register messages")
	}
	log.Print("Trying to load")
	if err := s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}
	log.Print("Returning from NewSemiCentralizedService")
	return s, nil
}
