package zerolottery

import (
	"crypto/sha256"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/protobuf"
	"time"
)

type ByzcoinData struct {
	Signer darc.Signer
	Roster *onet.Roster
	Cl     *byzcoin.Client
	GMsg   *byzcoin.CreateGenesisBlock
	GDarc  *darc.Darc
	Csr    *byzcoin.CreateGenesisBlockResponse
}

type TransactionReply struct {
	*byzcoin.AddTxResponse
	byzcoin.InstanceID
}

type LotteryData struct {
	Secret [32]byte
	Digest [32]byte
}

type Commit struct {
	SecretHash [32]byte
}

func (byzd *ByzcoinData) GetProof(id byzcoin.InstanceID) (*byzcoin.GetProofResponse, error) {
	return byzd.Cl.GetProof(id.Slice())
}

func (byzd *ByzcoinData) AddCommitTransaction(ld *LotteryData, wait int) (*TransactionReply, error) {
	commit := &Commit{
		SecretHash: ld.Digest,
	}
	commitBuf, err := protobuf.Encode(commit)
	if err != nil {
		log.Errorf("AddCommitTransaction error: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{{
			InstanceID: byzcoin.NewInstanceID(byzd.GDarc.GetBaseID()),
			Nonce:      byzcoin.GenNonce(),
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: ContractCommitID,
				Args: byzcoin.Arguments{{
					Name: "commit", Value: commitBuf}},
			},
		}},
	}
	//Sign the transaction
	err = byzcoin.SignInstruction(&ctx.Instructions[0], byzd.GDarc.GetBaseID(), byzd.Signer)
	//err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
	if err != nil {
		log.Errorf("AddCommitTransaction error: %v", err)
		return nil, err
	}
	reply := &TransactionReply{}
	reply.InstanceID = ctx.Instructions[0].DeriveID("")
	//Delegate the work to the byzcoin client
	if wait == 0 {
		reply.AddTxResponse, err = byzd.Cl.AddTransaction(ctx)
	} else {
		reply.AddTxResponse, err = byzd.Cl.AddTransactionAndWait(ctx, wait)
	}
	if err != nil {
		log.Errorf("AddCommitTransaction error: %v", err)
		return nil, err
	}
	return reply, err
}

func CreateLotteryData() *LotteryData {
	var secret [32]byte
	random.Bytes(secret[:], random.New())
	digest := sha256.Sum256(secret[:])
	ld := &LotteryData{
		Secret: secret,
		Digest: digest,
	}
	return ld
}

func SetupByzcoin(r *onet.Roster) (*ByzcoinData, error) {
	var err error
	byzd := &ByzcoinData{}
	byzd.Signer = darc.NewSignerEd25519(nil, nil)
	byzd.GMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + ContractCommitID}, byzd.Signer.Identity())
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	byzd.GMsg.BlockInterval = 500 * time.Millisecond
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	return byzd, nil
}
