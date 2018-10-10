package service

import (
	"github.com/dedis/kyber"
)

type WriteRequest struct {
	EncData []byte
	K       kyber.Point
	C       kyber.Point
	Reader  kyber.Point
}

type WriteReply struct {
	WriteID []byte
}

type ReadRequest struct{}

type ReadReply struct{}
