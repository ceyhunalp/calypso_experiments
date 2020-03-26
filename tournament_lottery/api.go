package tournament

import (
	"crypto/sha256"
	"go.dedis.ch/cothority/byzcoin"
	"go.dedis.ch/cothority/darc"
	"go.dedis.ch/kyber/util/random"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/log"
	"go.dedis.ch/protobuf"
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

type DataStore struct {
	Data [32]byte
}

//type Commit struct {
//SecretHash [32]byte
//}

func SafeXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

func OrganizeList(participantList []int, winnerList []int) {
	wlIdx := 0
	plIdx := 0
	foundCnt := 0
	wlSz := len(winnerList)
	plSz := len(participantList)
	for plIdx < plSz && wlIdx < wlSz {
		val := participantList[plIdx]
		if val == 1 {
			if foundCnt < winnerList[wlIdx] {
				participantList[plIdx] = 0
			} else if foundCnt == winnerList[wlIdx] {
				wlIdx++
			}
			foundCnt++
		}
		plIdx++
	}
}

func (byzd *ByzcoinData) GetProof(id byzcoin.InstanceID) (*byzcoin.GetProofResponse, error) {
	pr, err := byzd.Cl.GetProof(id.Slice())
	if err != nil {
		log.Errorf("GetProof error: %v", err)
	}
	return pr, err
}

func (byzd *ByzcoinData) AddSecretTransaction(ld *LotteryData, wait int) (*TransactionReply, error) {
	//commit := &Commit{
	//SecretHash: ld.Digest,
	//}
	secret := &DataStore{
		Data: ld.Secret,
	}
	secretBuf, err := protobuf.Encode(secret)
	if err != nil {
		log.Errorf("AddSecretTransaction error: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{{
			InstanceID: byzcoin.NewInstanceID(byzd.GDarc.GetBaseID()),
			Nonce:      byzcoin.GenNonce(),
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: ContractLotteryStoreID,
				Args: byzcoin.Arguments{{
					Name: "store", Value: secretBuf}},
			},
		}},
	}
	//Sign the transaction
	err = byzcoin.SignInstruction(&ctx.Instructions[0], byzd.GDarc.GetBaseID(), byzd.Signer)
	//err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
	if err != nil {
		log.Errorf("AddSecretTransaction error: %v", err)
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
		log.Errorf("AddSecretTransaction error: %v", err)
		return nil, err
	}
	return reply, err
}

func (byzd *ByzcoinData) AddCommitTransaction(ld *LotteryData, wait int) (*TransactionReply, error) {
	//commit := &Commit{
	//SecretHash: ld.Digest,
	//}
	commit := &DataStore{
		Data: ld.Digest,
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
				ContractID: ContractLotteryStoreID,
				//ContractID: ContractCommitID,
				Args: byzcoin.Arguments{{
					Name: "store", Value: commitBuf}},
				//Name: "commit", Value: commitBuf}},
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
	byzd.GMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + ContractLotteryStoreID}, byzd.Signer.Identity())
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	byzd.GMsg.BlockInterval = 5 * time.Second
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	return byzd, nil
}
