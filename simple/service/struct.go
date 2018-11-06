package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/ceyhunalp/centralized_calypso/util"
	bolt "github.com/coreos/bbolt"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/skipchain"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
)

type SimpleCalypsoDB struct {
	*bolt.DB
	bucketName []byte
}

type StoreRequest struct {
	Data     []byte
	DataHash []byte
}

type StoreReply struct {
	StoredKey string
}

type DecryptRequest struct {
	Write *byzcoin.Proof
	Read  *byzcoin.Proof
	SCID  skipchain.SkipBlockID
	Key   string
	Sig   []byte
}

type DecryptReply struct {
	Data     []byte
	DataHash []byte
	K        kyber.Point
	C        kyber.Point
}

func reencryptData(wt *calypso.SimpleWrite, sk kyber.Scalar) (kyber.Point, kyber.Point, error) {
	symKey, err := util.ElGamalDecrypt(sk, wt.K, wt.C)
	if err != nil {
		return nil, nil, err
	}

	decReader, err := util.AeadOpen(symKey, wt.EncReader)
	if err != nil {
		return nil, nil, err
	}

	ok, err := util.CompareKeys(wt.Reader, decReader)
	if err != nil {
		return nil, nil, err
	}
	if ok != 0 {
		return nil, nil, errors.New("Reader public key does not match")
	}

	k, c, _ := util.ElGamalEncrypt(wt.Reader, symKey)
	return k, c, nil
}

func verifyDecryptRequest(req *DecryptRequest, storedData *StoreRequest, sk kyber.Scalar) (*calypso.SimpleWrite, error) {
	log.Lvl2("Re-encrypt the key to the public key of the reader")

	var read calypso.Read
	if err := req.Read.ContractValue(cothority.Suite, calypso.ContractReadID, &read); err != nil {
		return nil, errors.New("didn't get a read instance: " + err.Error())
	}
	var write calypso.SimpleWrite
	if err := req.Write.ContractValue(cothority.Suite, calypso.ContractSimpleWriteID, &write); err != nil {
		return nil, errors.New("didn't get a write instance: " + err.Error())
	}
	if !read.Write.Equal(byzcoin.NewInstanceID(req.Write.InclusionProof.Key)) {
		return nil, errors.New("read doesn't point to passed write")
	}
	if err := req.Read.Verify(req.SCID); err != nil {
		return nil, errors.New("read proof cannot be verified to come from scID: " + err.Error())
	}
	if err := req.Write.Verify(req.SCID); err != nil {
		return nil, errors.New("write proof cannot be verified to come from scID: " + err.Error())
	}

	keyBytes, err := hex.DecodeString(req.Key)
	if err != nil {
		return nil, err
	}
	ok := bytes.Compare(keyBytes, storedData.DataHash)
	if ok != 0 {
		return nil, errors.New("Keys do not match")
	}
	err = schnorr.Verify(cothority.Suite, write.Reader, keyBytes, req.Sig)
	if err != nil {
		return nil, err
	}
	return &write, nil
}

func getDecryptedData(req *DecryptRequest, storedData *StoreRequest, sk kyber.Scalar) (*DecryptReply, error) {
	writeTxn, err := verifyDecryptRequest(req, storedData, sk)
	if err != nil {
		return nil, err
	}
	k, c, err := reencryptData(writeTxn, sk)
	if err != nil {
		return nil, err
	}
	return &DecryptReply{Data: storedData.Data, DataHash: storedData.DataHash, K: k, C: c}, nil
}

func (sdb *SimpleCalypsoDB) getFromTx(tx *bolt.Tx, key []byte) (*StoreRequest, error) {
	val := tx.Bucket([]byte(sdb.bucketName)).Get(key)
	if val == nil {
		return nil, errors.New("Key does not exist")
	}

	buf := make([]byte, len(val))
	copy(buf, val)
	_, sr, err := network.Unmarshal(buf, cothority.Suite)
	if err != nil {
		return nil, err
	}
	return sr.(*StoreRequest), nil
}

func (sdb *SimpleCalypsoDB) GetStoredData(key string) (*StoreRequest, error) {
	var result *StoreRequest
	keyByte, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	err = sdb.DB.View(func(tx *bolt.Tx) error {
		v, err := sdb.getFromTx(tx, keyByte)
		if err != nil {
			return err
		}
		result = v
		return nil
	})
	return result, err
}

func (sdb *SimpleCalypsoDB) StoreData(req *StoreRequest) (key string, err error) {
	dataHash := sha256.Sum256(req.Data)
	if bytes.Compare(dataHash[:], req.DataHash) != 0 {
		return key, errors.New("Hashes do not match")
	}
	val, err := network.Marshal(req)
	if err != nil {
		return key, errors.New("Cannot marshal store request")
	}
	err = sdb.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(sdb.bucketName)
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
		return key, err
	}
	return hex.EncodeToString(dataHash[:]), nil
}

func NewSimpleCalypsoDB(db *bolt.DB, bn []byte) *SimpleCalypsoDB {
	return &SimpleCalypsoDB{
		DB:         db,
		bucketName: bn,
	}
}
