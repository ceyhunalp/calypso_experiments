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

/*
 *func (c *Client) Write(r *onet.Roster) (*WriteReply, error) {
 *        dst := r.RandomServerIdentity()
 *        log.Lvl3("Sending message to", dst)
 *        reply := &WriteReply{}
 *        err := c.SendProtobuf(dst, &WriteReq{r}, reply)
 *        if err != nil {
 *                return nil, err
 *        }
 *        return reply, nil
 *}
 */

// Clock chooses one server from the Roster at random. It
// sends a Clock to it, which is then processed on the server side
// via the code in the service package.
//
// Clock will return the time in seconds it took to run the protocol.
/*
 *func (c *Client) Clock(r *onet.Roster) (*ClockReply, error) {
 *        dst := r.RandomServerIdentity()
 *        log.Lvl4("Sending message to", dst)
 *        reply := &ClockReply{}
 *        err := c.SendProtobuf(dst, &Clock{r}, reply)
 *        if err != nil {
 *                return nil, err
 *        }
 *        return reply, nil
 *}
 */

// Count will return the number of times `Clock` has been called on this
// service-node.
/*
 *func (c *Client) Count(si *network.ServerIdentity) (int, error) {
 *        reply := &CountReply{}
 *        err := c.SendProtobuf(si, &Count{}, reply)
 *        if err != nil {
 *                return -1, err
 *        }
 *        return reply.Count, nil
 *}
 */
