package lottery

import (
	"crypto/sha256"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"time"
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

//func (byzd *ByzcoinData) DecryptRequest(r *onet.Roster, wrProof *byzcoin.Proof, rProof *byzcoin.Proof, key string, sk kyber.Scalar) (*simpServ.DecryptReply, error) {
//cl := simpServ.NewClient()
//defer cl.Close()
//keyBytes, err := hex.DecodeString(key)
//if err != nil {
//log.Errorf("DecryptRequest error: %v", err)
//return nil, err
//}
//sig, err := schnorr.Sign(cothority.Suite, sk, keyBytes)
//if err != nil {
//log.Errorf("DecryptRequest error: %v", err)
//return nil, err
//}
//dr := &simpServ.DecryptRequest{
//Write: wrProof,
//Read:  rProof,
//SCID:  byzd.Cl.ID,
//Key:   key,
//Sig:   sig,
//}
//return cl.Decrypt(r, dr)
//}

//func (byzd *ByzcoinData) GetProof(id byzcoin.InstanceID) (*byzcoin.GetProofResponse, error) {
//pr, err := byzd.Cl.GetProof(id.Slice())
//if err != nil {
//log.Errorf("GetProof error: %v", err)
//}
//return pr, err
//}

//func (byzd *ByzcoinData) AddReadTransaction(proof *byzcoin.Proof, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
//read := &calypso.Read{
//Write: byzcoin.NewInstanceID(proof.InclusionProof.Key),
//Xc:    signer.Ed25519.Point,
//}
//readBuf, err := protobuf.Encode(read)
//if err != nil {
//log.Errorf("AddReadTransaction error: %v", err)
//return nil, err
//}
//ctx := byzcoin.ClientTransaction{
//Instructions: byzcoin.Instructions{{
//InstanceID: byzcoin.NewInstanceID(proof.InclusionProof.Key),
//Nonce:      byzcoin.Nonce{},
//Index:      0,
//Length:     1,
//Spawn: &byzcoin.Spawn{
//ContractID: calypso.ContractReadID,
//Args:       byzcoin.Arguments{{Name: "read", Value: readBuf}},
//},
//}},
//}
//err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
//if err != nil {
//log.Errorf("AddReadTransaction error: %v", err)
//return nil, err
//}
//reply := &TransactionReply{}
//reply.InstanceID = ctx.Instructions[0].DeriveID("")
//if wait == 0 {
//reply.AddTxResponse, err = byzd.Cl.AddTransaction(ctx)
//} else {
//reply.AddTxResponse, err = byzd.Cl.AddTransactionAndWait(ctx, wait)
//}
//if err != nil {
//log.Errorf("AddReadTransaction error: %v", err)
//return nil, err
//}
//return reply, nil
//}

//func (byzd *ByzcoinData) AddWriteTransaction(wd *util.WriteData, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
//sWrite := &calypso.SimpleWrite{
//DataHash:  wd.DataHash,
//K:         wd.K,
//C:         wd.C,
//Reader:    wd.Reader,
//EncReader: wd.EncReader,
//}
//writeBuf, err := protobuf.Encode(sWrite)
//if err != nil {
//log.Errorf("AddWriteTransaction error: %v", err)
//return nil, err
//}
//ctx := byzcoin.ClientTransaction{
//Instructions: byzcoin.Instructions{{
//InstanceID: byzcoin.NewInstanceID(darc.GetBaseID()),
//Nonce:      byzcoin.Nonce{},
//Index:      0,
//Length:     1,
//Spawn: &byzcoin.Spawn{
//ContractID: calypso.ContractSimpleWriteID,
//Args: byzcoin.Arguments{{
//Name: "write", Value: writeBuf}},
//},
//}},
//}
////Sign the transaction
//err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
//if err != nil {
//log.Errorf("AddWriteTransaction error: %v", err)
//return nil, err
//}
//reply := &TransactionReply{}
//reply.InstanceID = ctx.Instructions[0].DeriveID("")
////Delegate the work to the byzcoin client
////reply.AddTxResponse, err = byzd.Cl.AddTransaction(ctx)
//if wait == 0 {
//reply.AddTxResponse, err = byzd.Cl.AddTransaction(ctx)
//} else {
//reply.AddTxResponse, err = byzd.Cl.AddTransactionAndWait(ctx, wait)
//}
//if err != nil {
//log.Errorf("AddWriteTransaction error: %v", err)
//return nil, err
//}
//return reply, err
//}

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
	byzd.GMsg.BlockInterval = 15 * time.Second
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	return byzd, nil
}
