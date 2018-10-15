package util

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/dedis/kyber"
	"github.com/dedis/kyber/util/encoding"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/app"
	"github.com/dedis/onet/log"
)

const nonceLen = 12

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

func aeadOpen(key, ciphertext []byte) ([]byte, error) {
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

func RecoverData(encData []byte, gr kyber.Group, sk kyber.Scalar, k kyber.Point, c kyber.Point) ([]byte, error) {
	recvKey, err := ElGamalDecrypt(gr, sk, k, c)
	if err != nil {
		return nil, err
	}
	return aeadOpen(recvKey, encData)
}

func ElGamalDecrypt(group kyber.Group, sk kyber.Scalar, K kyber.Point, C kyber.Point) ([]byte, error) {
	S := group.Point().Mul(sk, K)
	M := group.Point().Sub(C, S)
	return M.Data()
}

func ElGamalEncrypt(group kyber.Group, pk kyber.Point, msg []byte) (K, C kyber.Point, remainder []byte) {

	// Embed the message (or as much of it as will fit) into a curve point.
	M := group.Point().Embed(msg, random.New())
	max := group.Point().EmbedLen()
	if max > len(msg) {
		max = len(msg)
	}
	remainder = msg[max:]
	// ElGamal-encrypt the point to produce ciphertext (K,C).
	k := group.Scalar().Pick(random.New()) // ephemeral private key
	K = group.Point().Mul(k, nil)          // ephemeral DH public key
	S := group.Point().Mul(k, pk)          // ephemeral DH shared secret
	C = S.Add(S, M)                        // message blinded with secret
	return
}

func SymEncrypt(msg []byte, key []byte) ([]byte, error) {
	encData, err := aeadSeal(key[:], msg)
	if err != nil {
		return nil, err
	}
	return encData, nil
}

func GetServerKey(fname *string, group kyber.Group) (kyber.Point, error) {
	//func GetServerKey(fname *string, group kyber.Group) ([]kyber.Point, error) {
	var keys []kyber.Point
	fh, err := os.Open(*fname)
	defer fh.Close()
	if err != nil {
		return nil, err
	}

	fs := bufio.NewScanner(fh)
	for fs.Scan() {
		tmp, err := encoding.StringHexToPoint(group, fs.Text())
		if err != nil {
			return nil, err
		}
		keys = append(keys, tmp)
	}
	return keys[0], nil
}

func ReadRoster(path string) (*onet.Roster, error) {
	file, err := os.Open(path)
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
