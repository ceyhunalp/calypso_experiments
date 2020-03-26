package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/ceyhunalp/calypso_experiments/util"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	bolt "go.etcd.io/bbolt"
)

type CentralizedCalypsoDB struct {
	*bolt.DB
	bucketName []byte
}

type WriteRequest struct {
	EncData   []byte
	DataHash  []byte
	K         kyber.Point
	C         kyber.Point
	Reader    kyber.Point
	EncReader []byte
}

type WriteReply struct {
	WriteID string
}

type ReadRequest struct {
	WriteID string
	Sig     []byte
}

type ReadReply struct {
	K kyber.Point
	C kyber.Point
}

func reencryptData(rr *ReadRequest, sw *WriteRequest, sk kyber.Scalar) (kyber.Point, kyber.Point, error) {
	// Check that the writeIDs match
	widBytes, err := hex.DecodeString(rr.WriteID)
	if err != nil {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, err
	}
	ok := bytes.Compare(widBytes, sw.DataHash)
	if ok != 0 {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, errors.New("WriteIDs do not match")
	}

	// Verify the signature on read request against the policy in WR
	err = schnorr.Verify(cothority.Suite, sw.Reader, widBytes, rr.Sig)
	if err != nil {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, err
	}

	// Get the symmetric key
	symKey, err := util.ElGamalDecrypt(sk, sw.K, sw.C)
	if err != nil {
		log.Errorf("reencryptData error: %v", err)
		return nil, nil, err
	}
	// Check that the reader is "the reader"
	//decReader, err := util.AeadOpen(symKey, sw.EncReader)
	//if err != nil {
	//log.Errorf("reencryptData error: %v", err)
	//return nil, nil, err
	//}

	//ok, err = util.CompareKeys(sw.Reader, decReader)
	//if err != nil {
	//log.Errorf("reencryptData error: %v", err)
	//return nil, nil, err
	//}
	//if ok != 0 {
	//log.Errorf("reencryptData error: Reader public key does not match")
	//return nil, nil, errors.New("Reader public key does not match")
	//}
	// Reencrypt the symmetric key for the reader
	k, c, _ := util.ElGamalEncrypt(sw.Reader, symKey)
	return k, c, nil
}

func (cdb *CentralizedCalypsoDB) getFromTx(tx *bolt.Tx, key []byte) (*WriteRequest, error) {
	val := tx.Bucket([]byte(cdb.bucketName)).Get(key)
	if val == nil {
		log.Errorf("getFromTx error: Key does not exist")
		return nil, errors.New("Key does not exist")
	}

	buf := make([]byte, len(val))
	copy(buf, val)
	_, wr, err := network.Unmarshal(buf, cothority.Suite)
	if err != nil {
		log.Errorf("getFromTx error: %v", err)
		return nil, err
	}
	return wr.(*WriteRequest), nil
}

func (cdb *CentralizedCalypsoDB) GetWrite(wID string) (*WriteRequest, error) {
	var result *WriteRequest
	key, err := hex.DecodeString(wID)
	if err != nil {
		log.Errorf("GetWrite error: %v", err)
		return result, err
	}
	err = cdb.DB.View(func(tx *bolt.Tx) error {
		v, err := cdb.getFromTx(tx, key)
		if err != nil {
			log.Errorf("GetWrite error: %v", err)
			return err
		}
		result = v
		return nil
	})
	return result, err
}

func (cdb *CentralizedCalypsoDB) StoreWrite(req *WriteRequest) (string, error) {
	dataHash := sha256.Sum256(req.EncData)
	if bytes.Compare(dataHash[:], req.DataHash) != 0 {
		log.Errorf("StoreWrite error: Hashes do not match")
		return "", errors.New("Hashes do not match")
	}
	val, err := network.Marshal(req)
	if err != nil {
		log.Errorf("StoreWrite error: Cannot marshal write request")
		return "", errors.New("Cannot marshal write request")
	}
	err = cdb.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(cdb.bucketName)
		v := b.Get(dataHash[:])
		if v != nil {
			log.Errorf("StoreWrite error: Key already exists")
			return errors.New("Key already exists")
		}
		err := b.Put(dataHash[:], val)
		if err != nil {
			log.Errorf("StoreWrite error: Cannot store the value")
			return errors.New("Cannot store the value")
		}
		return nil
	})
	if err != nil {
		log.Errorf("StoreWrite error: %v", err)
		return "", err
	}
	return hex.EncodeToString(dataHash[:]), nil
}

func NewCentralizedCalypsoDB(db *bolt.DB, bn []byte) *CentralizedCalypsoDB {
	return &CentralizedCalypsoDB{
		DB:         db,
		bucketName: bn,
	}
}
