package lottery

import (
	"crypto/sha256"
	"time"

	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
)

type LotteryData struct {
	Secret [32]byte
	Digest [32]byte
}

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

func (byzd *ByzcoinData) SpawnDarc(spawnDarc darc.Darc, wait int) (*byzcoin.AddTxResponse, error) {
	darcBuf, err := spawnDarc.ToProto()
	if err != nil {
		log.Errorf("SpawnDarc error: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID: byzcoin.NewInstanceID(byzd.GDarc.GetBaseID()),
			Nonce:      byzcoin.GenNonce(),
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: byzcoin.ContractDarcID,
				Args: []byzcoin.Argument{{
					Name:  "darc",
					Value: darcBuf,
				}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(byzd.GDarc.GetBaseID(), byzd.Signer)
	if err != nil {
		log.Errorf("SpawnDarc error: %v", err)
		return nil, err
	}
	return byzd.Cl.AddTransactionAndWait(ctx, wait)
}

func SetupDarcs(numParticipant int) ([]darc.Signer, darc.Signer, []*darc.Darc, error) {
	//var writer darc.Signer
	var reader darc.Signer
	writerList := make([]darc.Signer, numParticipant)
	writeDarcList := make([]*darc.Darc, numParticipant)
	reader = darc.NewSignerEd25519(nil, nil)

	for i := 0; i < numParticipant; i++ {
		writerList[i] = darc.NewSignerEd25519(nil, nil)
	}

	for i := 0; i < numParticipant; i++ {
		writeDarcList[i] = darc.NewDarc(darc.InitRules([]darc.Identity{writerList[i].Identity()}, []darc.Identity{writerList[i].Identity()}), []byte("Writer"))
		err := writeDarcList[i].Rules.AddRule(darc.Action("spawn:"+calypso.ContractWriteID), expression.InitOrExpr(writerList[i].Identity().String()))
		if err != nil {
			log.Errorf("SetupDarcs error: %v", err)
			return nil, reader, nil, err
		}
		err = writeDarcList[i].Rules.AddRule(darc.Action("spawn:"+calypso.ContractReadID), expression.InitOrExpr(reader.Identity().String()))
		if err != nil {
			log.Errorf("SetupDarcs error: %v", err)
			return nil, reader, nil, err
		}
	}
	return writerList, reader, writeDarcList, nil
}

func SetupByzcoin(r *onet.Roster) (*ByzcoinData, error) {
	var err error
	byzd := &ByzcoinData{}
	byzd.Signer = darc.NewSignerEd25519(nil, nil)
	byzd.GMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + byzcoin.ContractDarcID}, byzd.Signer.Identity())
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	//byzd.GMsg.BlockInterval = 15 * time.Second
	//byzd.GMsg.BlockInterval = 10 * time.Second
	byzd.GMsg.BlockInterval = 5 * time.Second
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	return byzd, nil
}
