package semicentralized

import (
	bolt "github.com/coreos/bbolt"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/skipchain"
	"github.com/dedis/kyber"
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

type TransactionReply struct {
	*byzcoin.AddTxResponse
	byzcoin.InstanceID
}
