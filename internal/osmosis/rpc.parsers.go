package osmosis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/syntropynet/osmosis-publisher/pkg/types"

	ibctypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	types1 "github.com/cosmos/cosmos-sdk/codec/types"
	cosmotypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmtypes "github.com/cometbft/cometbft/types"

	gammtypes "github.com/osmosis-labs/osmosis/v22/x/gamm/types"
	pmtypes "github.com/osmosis-labs/osmosis/v22/x/poolmanager/types"
)

type TxProtoGetter interface {
	GetProtoTx() *tx.Tx
}

type IBCDenomTrace map[string]ibctypes.DenomTrace

func (c IBCDenomTrace) Add(denom string) {
	if !strings.HasPrefix(strings.ToLower(denom), "ibc/") {
		return
	}
	c[denom] = ibctypes.DenomTrace{}
}

func translateCoins(coins cosmotypes.Coins) []*cosmotypes.Coin {
	res := make([]*cosmotypes.Coin, len(coins))
	for i, c := range coins {
		// NOTE: for variable scope is per-for, not per iteration, thus we need to do a copy here.
		c := c
		res[i] = &c
	}
	return res
}

func (c *rpc) decodeTransaction(txRaw []byte) (cosmotypes.Tx, error) {
	decoder := c.enccfg.TxConfig.TxDecoder()
	return decoder(txRaw)
}

func (c *rpc) getDenomsFromTransactions(tx cosmotypes.Tx) (IBCDenomTrace, error) {
	ibcTrace := make(IBCDenomTrace)
	for _, msg := range tx.GetMsgs() {
		switch m := msg.(type) {
		case *banktypes.MsgMultiSend:
			for _, input := range m.Inputs {
				ibcTrace.Add(input.Coins[0].Denom)
			}
			for _, output := range m.Outputs {
				ibcTrace.Add(output.Coins[0].Denom)
			}
		case *banktypes.MsgSend:
			ibcTrace.Add(m.Amount[0].Denom)
		case *ibctypes.MsgTransfer:
			ibcTrace.Add(m.Token.Denom)
		case *gammtypes.MsgSwapExactAmountIn:
			ibcTrace.Add(m.TokenIn.Denom)
			for _, route := range m.Routes {
				ibcTrace.Add(route.GetTokenOutDenom())
			}
		case *gammtypes.MsgSwapExactAmountOut:
			ibcTrace.Add(m.TokenOut.Denom)
			for _, route := range m.Routes {
				ibcTrace.Add(route.GetTokenInDenom())
			}
		case *gammtypes.MsgJoinPool:
			for _, tokenInMax := range m.TokenInMaxs {
				ibcTrace.Add(tokenInMax.Denom)
			}
		case *gammtypes.MsgExitPool:
			for _, tokenOutMin := range m.TokenOutMins {
				ibcTrace.Add(tokenOutMin.Denom)
			}
		}
	}

	err := c.getDenoms(ibcTrace)
	return ibcTrace, err
}

func (c *rpc) translateTransaction(
	txRaw []byte, txid, nonce string, txResult *abci.TxResult, code *uint32,
) *types.Transaction {
	transaction := &types.Transaction{
		Nonce:    nonce,
		TxID:     txid,
		Raw:      hex.EncodeToString(txRaw),
		TxResult: txResult,
	}
	if code != nil {
		transaction.Code = *code
	}

	decodedTx, err := c.decodeTransaction(txRaw)
	if err != nil {
		log.Println("Decode Transaction failed:", err.Error())
		return transaction
	}

	ibcMap, err := c.getDenomsFromTransactions(decodedTx)
	if err != nil {
		log.Println("Extracting denoms failed:", err.Error())
	} else {
		transaction.Metadata = ibcMap
	}

	getter, ok := decodedTx.(TxProtoGetter)
	if !ok {
		return transaction
	}

	tx := getter.GetProtoTx()
	b, err := c.enccfg.Marshaler.MarshalJSON(tx)
	if err != nil {
		log.Println("marshaling intermediate JSON failed: ", err.Error())
	}
	err = json.Unmarshal(b, &transaction.Tx)
	if err != nil {
		log.Println("unmarshaling intermediate JSON failed: ", err.Error())
	}
	transaction.Raw = ""

	return transaction
}

func (c *rpc) translateBlock(block *tmtypes.Block) *types.Block {
	blockProto, err := block.ToProto()
	if err != nil {
		panic(err)
	}

	return &types.Block{
		// Nonce: c.NewNonce(),
		Block: blockProto,
	}
}

func (c *rpc) translatePools(ps []*types1.Any) ([]*pmtypes.PoolI, error) {
	pools := make([]*pmtypes.PoolI, len(ps))
	for i, p := range ps {
		var pool pmtypes.PoolI
		err := c.enccfg.InterfaceRegistry.UnpackAny(p, &pool)
		if err != nil {
			return nil, fmt.Errorf("failed unmarshalling pool: %w", err)
		}
		pools[i] = &pool
	}
	return pools, nil
}

func extractTxMessageNames(tx *types.Transaction) []string {
	bodyM, ok := tx.Tx.(map[string]any)
	if !ok {
		return nil
	}
	body, ok := bodyM["body"]
	if !ok {
		return nil
	}
	msgsM, ok := body.(map[string]any)
	if !ok {
		return nil
	}
	msgsA, ok := msgsM["messages"]
	if !ok {
		return nil
	}
	msgs, ok := msgsA.([]any)
	if !ok {
		return nil
	}

	messageNames := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		t, ok := m["@type"]
		if !ok {
			continue
		}
		typeUrl := t.(string)
		messageNames = append(messageNames, typeUrl)
	}
	return messageNames
}
