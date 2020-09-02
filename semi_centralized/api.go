package semicentralized

import (
	"encoding/hex"
	"time"

	"github.com/ceyhunalp/calypso_experiments/util"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/protobuf"
)

const ServiceName = "SemiCentralizedService"

type SCClient struct {
	bcClient *byzcoin.Client
	c        *onet.Client
}

func NewClient(bc *byzcoin.Client) *SCClient {
	return &SCClient{bcClient: bc, c: onet.NewClient(cothority.Suite, ServiceName)}
}

func SetupByzcoin(r *onet.Roster, blockInterval int) (cl *byzcoin.Client, admin darc.Signer, gDarc darc.Darc, err error) {
	admin = darc.NewSignerEd25519(nil, nil)
	gMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + byzcoin.ContractDarcID, "spawn:" + calypso.ContractSemiWriteID, "spawn:" + calypso.ContractReadID}, admin.Identity())
	if err != nil {
		log.Errorf("Setting up byzcoin dfailed error: %v", err)
		return
	}
	gMsg.BlockInterval = time.Duration(blockInterval) * time.Second
	log.Info("Block interval is:", gMsg.BlockInterval)
	gDarc = gMsg.GenesisDarc
	cl, _, err = byzcoin.NewLedger(gMsg, false)
	if err != nil {
		log.Errorf("Setting up byzcoin failed: %v", err)
		return
	}
	return
}

func (scCl *SCClient) SetupDarcs() (darc.Signer, darc.Signer, *darc.Darc, error) {
	var writer darc.Signer
	var reader darc.Signer
	writer = darc.NewSignerEd25519(nil, nil)
	reader = darc.NewSignerEd25519(nil, nil)
	writeDarc := darc.NewDarc(darc.InitRules([]darc.Identity{writer.Identity()}, []darc.Identity{writer.Identity()}), []byte("Writer"))
	err := writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractSemiWriteID), expression.InitOrExpr(writer.Identity().String()))
	if err != nil {
		return writer, reader, nil, err
	}
	err = writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractReadID), expression.InitOrExpr(reader.Identity().String()))
	if err != nil {
		return writer, reader, nil, err
	}
	return writer, reader, writeDarc, nil
}

func (scCl *SCClient) SpawnDarc(signer darc.Signer, spawnDarc darc.Darc, controlDarc darc.Darc, wait int) (*byzcoin.AddTxResponse, error) {
	darcBuf, err := spawnDarc.ToProto()
	if err != nil {
		log.Errorf("Spawning darc failed: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID: byzcoin.NewInstanceID(controlDarc.GetBaseID()),
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
	err = ctx.Instructions[0].SignBy(controlDarc.GetBaseID(), signer)
	if err != nil {
		log.Errorf("Spawning darc failed: %v", err)
		return nil, err
	}
	return scCl.bcClient.AddTransactionAndWait(ctx, wait)
}

func (scCl *SCClient) StoreData(r *onet.Roster, data []byte, dataHash []byte) (*StoreReply, error) {
	sr := &StoreRequest{
		Data:     data,
		DataHash: dataHash,
	}
	dest := r.List[0]
	log.Lvl3("Sending message to", dest)
	reply := &StoreReply{}
	err := scCl.c.SendProtobuf(dest, sr, reply)
	if err != nil {
		log.Errorf("Storing encrypted data failed: %v", err)
		return nil, err
	}
	return reply, nil
}

func (scCl *SCClient) AddWriteTransaction(wd *util.WriteData, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
	sWrite := &calypso.SemiWrite{
		DataHash:  wd.DataHash,
		K:         wd.K,
		C:         wd.C,
		Reader:    wd.Reader,
		EncReader: wd.EncReader,
	}
	writeBuf, err := protobuf.Encode(sWrite)
	if err != nil {
		log.Errorf("Adding write transaction failed: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{{
			InstanceID: byzcoin.NewInstanceID(darc.GetBaseID()),
			Nonce:      byzcoin.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: calypso.ContractSemiWriteID,
				Args: byzcoin.Arguments{{
					Name: "write", Value: writeBuf}},
			},
		}},
	}
	//Sign the transaction
	err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
	if err != nil {
		log.Errorf("Adding write transaction failed: %v", err)
		return nil, err
	}
	reply := &TransactionReply{}
	reply.InstanceID = ctx.Instructions[0].DeriveID("")
	reply.AddTxResponse, err = scCl.bcClient.AddTransactionAndWait(ctx, wait)
	if err != nil {
		log.Errorf("Adding write transaction failed: %v", err)
		return nil, err
	}
	return reply, err
}

func (scCl *SCClient) AddReadTransaction(proof *byzcoin.Proof, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
	read := &calypso.Read{
		Write: byzcoin.NewInstanceID(proof.InclusionProof.Key),
		Xc:    signer.Ed25519.Point,
	}
	readBuf, err := protobuf.Encode(read)
	if err != nil {
		log.Errorf("Adding read transaction failed: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{{
			InstanceID: byzcoin.NewInstanceID(proof.InclusionProof.Key),
			Nonce:      byzcoin.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: calypso.ContractReadID,
				Args:       byzcoin.Arguments{{Name: "read", Value: readBuf}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
	if err != nil {
		log.Errorf("Adding read transaction failed: %v", err)
		return nil, err
	}
	reply := &TransactionReply{}
	reply.InstanceID = ctx.Instructions[0].DeriveID("")
	//if wait == 0 {
	//reply.AddTxResponse, err = scCl.bcClient.AddTransaction(ctx)
	//} else {
	//reply.AddTxResponse, err = scCl.bcClient.AddTransactionAndWait(ctx, wait)
	//}
	reply.AddTxResponse, err = scCl.bcClient.AddTransactionAndWait(ctx, wait)
	if err != nil {
		log.Errorf("Adding read transaction failed: %v", err)
		return nil, err
	}
	return reply, nil
}

func (scCl *SCClient) GetProof(id byzcoin.InstanceID) (*byzcoin.GetProofResponse, error) {
	return scCl.bcClient.GetProof(id.Slice())
}

func (scCl *SCClient) Decrypt(r *onet.Roster, wrProof *byzcoin.Proof, rProof *byzcoin.Proof, key string, sk kyber.Scalar) (*DecryptReply, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		log.Errorf("Decrypt failed: %v", err)
		return nil, err
	}
	sig, err := schnorr.Sign(cothority.Suite, sk, keyBytes)
	if err != nil {
		log.Errorf("Decrypt failed: %v", err)
		return nil, err
	}
	dr := &DecryptRequest{
		Write: wrProof,
		Read:  rProof,
		SCID:  scCl.bcClient.ID,
		Key:   key,
		Sig:   sig,
	}
	dest := r.List[0]
	log.Lvl3("Sending message to", dest)
	reply := &DecryptReply{}
	err = scCl.c.SendProtobuf(dest, dr, reply)
	if err != nil {
		log.Errorf("Decrypt failed: %v", err)
		return nil, err
	}
	return reply, err
}
