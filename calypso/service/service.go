package service

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	//"bytes"
	//"crypto/sha256"
	//"encoding/hex"
	"errors"
	"sync"

	//bolt "github.com/coreos/bbolt"
	//"github.com/dedis/kyber/group/edwards25519"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
)

// ServiceName is used for registration on the onet.
const ServiceName = "CalypsoService"

// Used for tests
var templateID onet.ServiceID

func init() {
	log.Print("init in service")
	var err error
	templateID, err = onet.RegisterNewService(ServiceName, newCalypsoService)
	log.ErrFatal(err)
	network.RegisterMessages(&WriteRequest{}, &WriteReply{})
	//network.RegisterMessages(&storage{}, &WriteRequest{}, &WriteReply{})
}

// Service is our template-service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
	db           *CalypsoDB
	storageMutex sync.Mutex
	//storage *storage
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("Calypso")

// storage is used to save our data.
//type storage struct {
//LoggedWrites map[string]*StoredWrite
//LoggedWrites map[string]*WriteRequest
//Db     *bolt.DB
//Bucket []byte
//sync.Mutex
//}

func (s *Service) Write(req *WriteRequest) (*WriteReply, error) {
	//sw, err := createStoredData(req, edwards25519.NewBlakeSHA256Ed25519())
	//if err != nil {
	//return nil, err
	//}
	//s.storage.Lock()
	//_, exists := s.storage.LoggedWrites[digestStr]
	//if exists {
	//log.Lvl3("Data already exists")
	//s.storage.Unlock()
	//return nil, errors.New("Data already exists")
	//}

	//log.Lvl3("Storing new data")
	//s.storage.LoggedWrites[digestStr] = sw
	//s.storage.LoggedWrites[digestStr] = req
	//s.storage.Unlock()
	//s.save()

	//s.storage.Lock()
	//defer s.storage.Unlock()
	//key, err := s.Load(dataDigest[:])
	//if err != nil {
	//return nil, errors.New("Load failed cannot check if the key already exists")
	//}
	//if key != nil {
	//return nil, errors.New("Key already exists")
	//}
	//err = s.Save(dataDigest[:], val)
	//if err != nil {
	//return nil, errors.New("Cannot store data to the database")
	//}
	storedKey, err := s.db.StoreWrite(req)
	if err != nil {
		return nil, err
	}
	resp := &WriteReply{
		WriteID: storedKey,
		//WriteID: hex.EncodeToString(dataDigest[:]),
	}
	return resp, nil
}

//func (s *Service) Read(req *ReadRequest) (*ReadReply, error) {
//s.storage.Lock()
//if sd, ok := s.storage.LoggedWrites[req.WriteID]; ok {
//_ = verifyReader(edwards25519.NewBlakeSHA256Ed25519(), sd, req.Sig)

//} else {
//log.Lvl3("Data does not exist")
//s.storage.Unlock()
//return nil, errors.New("Data does not exist")
//}

//}

// saves all data.
func (s *Service) save() {
	//s.storage.Lock()
	//defer s.storage.Unlock()
	//err := s.Save(storageID, s.storage)
	s.storageMutex.Lock()
	defer s.storageMutex.Unlock()
	//err := s.Save(storageID, s.storage)
	//if err != nil
	//log.Error("Couldn't save data:", err)
	//}
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
//func (s *Service) tryLoad() error {
//s.storage = &storage{}
/*
 *defer func() {
 *        if len(s.storage.LoggedWrites) == 0 {
 *                s.storage.LoggedWrites = make(map[string]*WriteRequest)
 *                //s.storage.LoggedWrites = make(map[string]*StoredWrite)
 *        }
 *}()
 */
//msg, err := s.Load(storageID)
//log.Lvl3("After calling load")
//if err != nil {
//log.Lvl3("Error is not nil")
//return err
//}
//if msg == nil {
//return nil
//}
//var ok bool
//s.storage, ok = msg.(*storage)
//log.Lvl3("After mesg")
//if !ok {
//return errors.New("Data of wrong type")
//}
//log.Lvl3("Before calling add bucket")
//_, bucket := s.GetAdditionalBucket([]byte("writetransactions"))
////s.storage.Db = db
//s.storage.Bucket = bucket
//return nil
//}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newCalypsoService(c *onet.Context) (onet.Service, error) {
	db, bucket := c.GetAdditionalBucket([]byte("calypsotransactions"))
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		db:               NewCalypsoDB(db, bucket),
	}
	if err := s.RegisterHandlers(s.Write); err != nil {
		//if err := s.RegisterHandlers(s.Write, s.Read); err != nil {
		return nil, errors.New("Couldn't register messages")
	}
	//if err := s.tryLoad(); err != nil {
	//log.Error(err)
	//return nil, err
	//}
	return s, nil
}
