package semicentralized

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	bolt "github.com/coreos/bbolt"
	"github.com/dedis/cothority"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
)

func NewSemiCentralizedDB(db *bolt.DB, bn []byte) *SemiCentralizedDB {
	return &SemiCentralizedDB{
		DB:         db,
		bucketName: bn,
	}
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
