package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	bolt "github.com/coreos/bbolt"
	"github.com/dedis/kyber"
	//"github.com/dedis/kyber/sign/schnorr"
	//"github.com/dedis/kyber/util/encoding"
	"github.com/dedis/onet/network"
	//"strings"
)

//func verifyReader(gr kyber.Group, sd *StoredWrite, rr *ReadRequest) error {
//rPk, err := encoding.StringHexToPoint(gr, sd.Reader)
//origWid := hex.EncodeToString(sd.DataHash)
//ok := strings.Compare(origWid, rr.WriteID)
//if ok != 0 {
//return errors.New("WriteIDs do not match")
//}
//widBytes, err := hex.DecodeString(rr.WriteID)
//ok := schnorr.Verify(gr, rPk, widBytes, sig)
//return nil
//}

func storeWrite() {

}

//func createStoredData(req *WriteRequest, gr kyber.Group) (*StoredWrite, error) {
//sw := &StoredWrite{}
//kStr, err := encoding.PointToStringHex(gr, req.K)
//if err != nil {
//return sw, errors.New("Cannot convert K to string")
//}
//cStr, err := encoding.PointToStringHex(gr, req.C)
//if err != nil {
//return sw, errors.New("Cannot convert C to string")
//}
//readerStr, err := encoding.PointToStringHex(gr, req.Reader)
//if err != nil {
//return sw, errors.New("Cannot convert Reader to string")
//}
//sw.EncData = req.EncData
//sw.K = kStr
//sw.C = cStr
//sw.Reader = readerStr
//return sw, nil
//}

func (cdb *CalypsoDB) StoreWrite(req *WriteRequest) (string, error) {
	dataDigest := sha256.Sum256(req.EncData)
	if bytes.Compare(dataDigest[:], req.DataHash) != 0 {
		return "", errors.New("Hashes do not match")
	}
	//digestStr := hex.EncodeToString(dataDigest[:])
	val, err := network.Marshal(req)
	if err != nil {
		return "", errors.New("Cannot marshal write request")
	}
	err = cdb.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(cdb.bucketName)
		//b := tx.Bucket([]byte("writetransactions"))
		v := b.Get(dataDigest[:])
		if v != nil {
			return errors.New("Key already exists")
		}
		err := b.Put(dataDigest[:], val)
		if err != nil {
			return errors.New("Cannot store the value")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(dataDigest[:]), nil
}

func NewCalypsoDB(db *bolt.DB, bn []byte) *CalypsoDB {
	return &CalypsoDB{
		DB:         db,
		bucketName: bn,
	}
}

type CalypsoDB struct {
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

type ReadReply struct{}

//type StoredWrite struct {
//EncData []byte
//K       string
//C       string
//Reader  string
//}
