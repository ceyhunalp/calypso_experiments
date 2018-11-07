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

func (c *Client) StoreData(r *onet.Roster, sr *StoreRequest) (*StoreReply, error) {
	dest := r.List[0]
	log.Lvl3("Sending message to", dest)
	reply := &StoreReply{}
	err := c.SendProtobuf(dest, sr, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) Decrypt(r *onet.Roster, dr *DecryptRequest) (*DecryptReply, error) {
	dest := r.List[0]
	log.Lvl3("Sending message to", dest)
	reply := &DecryptReply{}
	err := c.SendProtobuf(dest, dr, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}
