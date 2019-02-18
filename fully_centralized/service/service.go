package service

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	"errors"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
	"sync"
)

// ServiceName is used for registration on the onet.
const ServiceName = "CentralizedCalypsoService"

// Used for tests
var templateID onet.ServiceID

func init() {
	log.Print("init in service")
	var err error
	templateID, err = onet.RegisterNewService(ServiceName, newCentralizedCalypsoService)
	log.ErrFatal(err)
	network.RegisterMessages(&storage{}, &WriteRequest{}, &WriteReply{})
}

// Service is our template-service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
	db      *CentralizedCalypsoDB
	storage *storage
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("CentralizedCalypso")

// storage is used to save our data.
type storage struct {
	//Suite *edwards25519.SuiteEd25519
	sync.Mutex
}

func (s *Service) Write(req *WriteRequest) (*WriteReply, error) {
	storedKey, err := s.db.StoreWrite(req)
	if err != nil {
		log.Errorf("Write error: %v", err)
		return nil, err
	}
	reply := &WriteReply{
		WriteID: storedKey,
	}
	return reply, nil
}

func (s *Service) Read(req *ReadRequest) (*ReadReply, error) {
	sk := s.ServerIdentity().GetPrivate()
	storedWrite, err := s.db.GetWrite(req.WriteID)
	if err != nil {
		log.Errorf("Read error: %v", err)
		return nil, err
	}
	k, c, err := reencryptData(req, storedWrite, sk)
	if k == nil || c == nil {
		log.Errorf("Read error: %v", err)
		return nil, err
		//return nil, errors.New("Could not reencrypt symmetric key")
	}
	resp := &ReadReply{
		K: k,
		C: c,
	}
	return resp, nil
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
	log.Lvl3("After mesg")
	if !ok {
		return errors.New("Data of wrong type")
	}
	return nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newCentralizedCalypsoService(c *onet.Context) (onet.Service, error) {
	db, bucket := c.GetAdditionalBucket([]byte("centralizedcalypsotransactions"))
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		db:               NewCentralizedCalypsoDB(db, bucket),
	}
	if err := s.RegisterHandlers(s.Write, s.Read); err != nil {
		return nil, errors.New("Couldn't register messages")
	}
	if err := s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}
