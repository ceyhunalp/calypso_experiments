package util

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/dedis/cothority"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/util/encoding"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/app"
	"github.com/dedis/onet/log"
	"io"
	"os"
)

const nonceLen = 12

type WriteData struct {
	Data      []byte
	DataHash  []byte
	K         kyber.Point
	C         kyber.Point
	Reader    kyber.Point
	EncReader []byte
	StoredKey string
}

func CompareKeys(readerPt kyber.Point, decReader []byte) (int, error) {
	readerPtBytes, err := readerPt.MarshalBinary()
	if err != nil {
		return -1, err
	}
	same := bytes.Compare(readerPtBytes, decReader)
	return same, nil
}

func aeadSeal(symKey, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(symKey)
	if err != nil {
		return nil, err
	}
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, nonceLen)
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	encData := aesgcm.Seal(nil, nonce, data, nil)
	encData = append(encData, nonce...)
	return encData, nil
}

func AeadOpen(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	log.ErrFatal(err)
	if len(ciphertext) < 12 {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[len(ciphertext)-nonceLen:]
	out, err := aesgcm.Open(nil, nonce, ciphertext[0:len(ciphertext)-nonceLen], nil)
	return out, err
}

func RecoverData(encData []byte, sk kyber.Scalar, k kyber.Point, c kyber.Point) ([]byte, error) {
	recvKey, err := ElGamalDecrypt(sk, k, c)
	if err != nil {
		return nil, err
	}
	return AeadOpen(recvKey, encData)
}

func ElGamalDecrypt(sk kyber.Scalar, K kyber.Point, C kyber.Point) ([]byte, error) {
	S := cothority.Suite.Point().Mul(sk, K)
	M := cothority.Suite.Point().Sub(C, S)
	return M.Data()
}

func ElGamalEncrypt(pk kyber.Point, msg []byte) (K, C kyber.Point, remainder []byte) {
	// Embed the message (or as much of it as will fit) into a curve point.
	M := cothority.Suite.Point().Embed(msg, random.New())
	max := cothority.Suite.Point().EmbedLen()
	if max > len(msg) {
		max = len(msg)
	}
	remainder = msg[max:]
	// ElGamal-encrypt the point to produce ciphertext (K,C).
	k := cothority.Suite.Scalar().Pick(random.New()) // ephemeral private key
	K = cothority.Suite.Point().Mul(k, nil)          // ephemeral DH public key
	S := cothority.Suite.Point().Mul(k, pk)          // ephemeral DH shared secret
	C = S.Add(S, M)                                  // message blinded with secret
	return
}

func symEncrypt(msg []byte, key []byte) ([]byte, error) {
	encData, err := aeadSeal(key[:], msg)
	if err != nil {
		return nil, err
	}
	return encData, nil
}

func CreateWriteData(data []byte, reader kyber.Point, serverKey kyber.Point) (*WriteData, error) {
	var symKey [16]byte
	random.Bytes(symKey[:], random.New())
	encData, err := symEncrypt(data, symKey[:])
	if err != nil {
		log.Errorf("In CreateWriteData: %v", err)
		return nil, err
	}
	readerBytes, err := reader.MarshalBinary()
	if err != nil {
		log.Errorf("In CreateWriteData: %v", err)
		return nil, err
	}
	encReader, err := symEncrypt(readerBytes, symKey[:])
	if err != nil {
		log.Errorf("In CreateWriteData: %v", err)
		return nil, err
	}
	k, c, _ := ElGamalEncrypt(serverKey, symKey[:])
	dh := sha256.Sum256(encData)
	wd := &WriteData{
		Data:      encData,
		DataHash:  dh[:],
		K:         k,
		C:         c,
		Reader:    reader,
		EncReader: encReader,
	}
	return wd, nil
}

func GetServerKey(fname *string) (kyber.Point, error) {
	var keys []kyber.Point
	fh, err := os.Open(*fname)
	defer fh.Close()
	if err != nil {
		return nil, err
	}

	fs := bufio.NewScanner(fh)
	for fs.Scan() {
		tmp, err := encoding.StringHexToPoint(cothority.Suite, fs.Text())
		if err != nil {
			return nil, err
		}
		keys = append(keys, tmp)
	}
	return keys[0], nil
}

func ReadRoster(path *string) (*onet.Roster, error) {
	file, err := os.Open(*path)
	if err != nil {
		return nil, err
	}

	group, err := app.ReadGroupDescToml(file)
	if err != nil {
		return nil, err
	}

	if len(group.Roster.List) == 0 {
		fmt.Println("Empty roster")
		return nil, err
	}
	return group.Roster, nil
}
