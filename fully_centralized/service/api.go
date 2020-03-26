package service

/*
The api.go defines the methods that can be called from the outside. Most
of the methods will take a roster so that the service knows which nodes
it should work with.

This part of the service runs on the client or the app.
*/

import (
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
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
	dest := r.List[0]
	log.Lvl3("Sending message to", dest)
	reply := &WriteReply{}
	err := c.SendProtobuf(dest, wr, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) Read(r *onet.Roster, rr *ReadRequest) (*ReadReply, error) {
	dest := r.List[0]
	log.Lvl3("Sending message to", dest)
	reply := &ReadReply{}
	err := c.SendProtobuf(dest, rr, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}
