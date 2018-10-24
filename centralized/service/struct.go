package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/ceyhunalp/centralized_calypso/util"
	bolt "github.com/coreos/bbolt"
	"github.com/dedis/cothority"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
)

//func reencryptData(req *WriteRequest, sk kyber.Scalar) (kyber.Point, kyber.Point, []byte) {
func reencryptData(req *WriteRequest, sk kyber.Scalar, gr kyber.Group) (kyber.Point, kyber.Point, []byte) {
	//symKey, err := util.ElGamalDecrypt(sk, req.K, req.C)
	symKey, err := util.ElGamalDecrypt(gr, sk, req.K, req.C)
	if err != nil {
		log.Error("ElGamal decryption failed")
		return nil, nil, nil
	}
	//return util.ElGamalEncrypt(req.Reader, symKey)
	return util.ElGamalEncrypt(gr, req.Reader, symKey)
}

//func verifyReader(req *ReadRequest, storedWrite *WriteRequest) error {
func verifyReader(req *ReadRequest, storedWrite *WriteRequest, gr kyber.Group) error {
	widBytes, err := hex.DecodeString(req.WriteID)
	if err != nil {
		return err
	}
	ok := bytes.Compare(widBytes, storedWrite.DataHash)
	if ok != 0 {
		return errors.New("WriteIDs do not match")
	}
	return schnorr.Verify(gr, storedWrite.Reader, widBytes, req.Sig)
	//return schnorr.Verify(cothority.Suite, storedWrite.Reader, widBytes, req.Sig)
}

func (cdb *CentralizedCalypsoDB) getFromTx(tx *bolt.Tx, key []byte) (*WriteRequest, error) {
	val := tx.Bucket([]byte(cdb.bucketName)).Get(key)
	if val == nil {
		return nil, errors.New("Key does not exist")
	}

	buf := make([]byte, len(val))
	copy(buf, val)
	_, wr, err := network.Unmarshal(buf, cothority.Suite)
	if err != nil {
		return nil, err
	}
	return wr.(*WriteRequest), nil
}

func (cdb *CentralizedCalypsoDB) GetWrite(wID string) (*WriteRequest, error) {
	var result *WriteRequest
	key, err := hex.DecodeString(wID)
	if err != nil {
		return result, err
	}
	err = cdb.DB.View(func(tx *bolt.Tx) error {
		v, err := cdb.getFromTx(tx, key)
		if err != nil {
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
		return "", errors.New("Hashes do not match")
	}
	val, err := network.Marshal(req)
	if err != nil {
		return "", errors.New("Cannot marshal write request")
	}
	err = cdb.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(cdb.bucketName)
		v := b.Get(dataHash[:])
		if v != nil {
			return errors.New("Key already exists")
		}
		err := b.Put(dataHash[:], val)
		if err != nil {
			return errors.New("Cannot store the value")
		}
		return nil
	})
	if err != nil {
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

type CentralizedCalypsoDB struct {
	*bolt.DB
	bucketName []byte
}

type WriteRequest struct {
	EncData  []byte
	DataHash []byte
	K        kyber.Point
	C        kyber.Point
	Reader   kyber.Point
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
