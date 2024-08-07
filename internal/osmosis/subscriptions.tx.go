package osmosis

import (
	"encoding/hex"
	"fmt"
	"time"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

func (p *Publisher) subscribeTransactions() error {
	return p.rpc.Subscribe(fmt.Sprintf("tm.event='%s'", tmtypes.EventTx), p.handleTransactions)
}

func (p *Publisher) handleTransactions(events <-chan ctypes.ResultEvent) error {
	sentinel := p.MakeSentinel(time.Minute * 10)

	for {
		select {
		case <-p.Context.Done():
			p.Logger.Info("handleTransactions: c.Context Done")
			return nil
		case ev, ok := <-events:
			if !ok {
				p.Logger.Info("handleTransactions: events closed")
				return nil
			}

			if err := sentinel(); err != nil {
				return err
			}

			switch data := ev.Data.(type) {
			case tmtypes.EventDataTx:
				p.handleTransaction(data, len(events))
			default:
				p.evtOtherCounter.Add(1)
			}
		}
	}
}

func (p *Publisher) handleTransaction(data tmtypes.EventDataTx, queueSize int) {
	p.txCounter.Add(1)
	p.transactionsCounter.Add(1)
	txData := data.GetTx()
	hash := hex.EncodeToString(tmtypes.Tx(txData).Hash())
	tx := p.rpc.translateTransaction(txData, hash, p.NewNonce(), &data.TxResult, &data.TxResult.Result.Code)
	p.Publish(
		tx,
		"tx",
	)

	p.messagesCounter.Add(1)
	p.Logger.Debug("Transaction", "txID", tx.TxID, "names", extractTxMessageNames(tx), "queue_size", queueSize)
}
