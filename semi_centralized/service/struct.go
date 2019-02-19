package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/ceyhunalp/calypso_experiments/util"
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

type SemiCentralizedDB struct {
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

func getDecryptedData(req *DecryptRequest, storedData *StoreRequest, sk kyber.Scalar) (*DecryptReply, error) {
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
}

func (sdb *SemiCentralizedDB) getFromTx(tx *bolt.Tx, key []byte) (*StoreRequest, error) {
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

func (sdb *SemiCentralizedDB) GetStoredData(key string) (*StoreRequest, error) {
	var result *StoreRequest
	keyByte, err := hex.DecodeString(key)
	if err != nil {
		log.Errorf("GetStoredData error: %v", err)
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

func (sdb *SemiCentralizedDB) StoreData(req *StoreRequest) (key string, err error) {
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

func NewSemiCentralizedDB(db *bolt.DB, bn []byte) *SemiCentralizedDB {
	return &SemiCentralizedDB{
		DB:         db,
		bucketName: bn,
	}
}
