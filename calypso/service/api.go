package service

/*
The api.go defines the methods that can be called from the outside. Most
of the methods will take a roster so that the service knows which nodes
it should work with.

This part of the service runs on the client or the app.
*/

import (
	"github.com/dedis/cothority"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	//"github.com/dedis/onet/network"
)

// Client is a structure to communicate with the template
// service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new template.Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, ServiceName)}
}

func (c *Client) Write(r *onet.Roster, wr *WriteRequest) (*WriteReply, error) {
	dst := r.List[0]
	log.Lvl3("Sending message to", dst)
	reply := &WriteReply{}
	err := c.SendProtobuf(dst, wr, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) Read(r *onet.Roster, rr *ReadRequest) (*ReadReply, error) {
	dst := r.List[0]
	log.Lvl3("Sending message to", dst)
	reply := &ReadReply{}
	err := c.SendProtobuf(dst, rr, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}
