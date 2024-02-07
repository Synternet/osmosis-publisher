package osmosis

import (
	"fmt"
	"log"
	"time"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

func (p *Publisher) subscribeBlocks() error {
	return p.rpc.Subscribe(fmt.Sprintf("tm.event='%s'", tmtypes.EventNewBlock), p.handleBlocks)
}

func (p *Publisher) handleBlocks(events <-chan ctypes.ResultEvent) error {
	sentinel := p.MakeSentinel(time.Minute)

	for {
		select {
		case <-p.Context.Done():
			log.Println("handleTransactions: c.Context Done")
			return nil
		case ev, ok := <-events:
			if !ok {
				log.Println("handleTransactions: events closed")
				return nil
			}

			if err := sentinel(); err != nil {
				return err
			}

			switch data := ev.Data.(type) {
			case tmtypes.EventDataNewBlock:
				now := time.Now()
				log.Printf("Block: START hash=%s height=%d len(queue)=%d", data.Block.Hash().String(), data.Block.Height, len(events))
				p.indexer.SetLatestBlockHeight(uint64(data.Block.Height), data.Block.Time)
				p.handleBlock(data.Block)
				p.handleMonitoredPools(data.Block.Height, data.Block.Time, data.Block.Hash().String())
				log.Printf("Block: FINISH hash=%s height=%d duration=%v len(queue)=%d", data.Block.Hash().String(), data.Block.Height, time.Since(now), len(events))
			default:
				p.evtOtherCounter.Add(1)
			}
		}
	}
}

func (p *Publisher) handleBlock(block *tmtypes.Block) {
	p.blockCounter.Add(1)
	outBlock := p.rpc.translateBlock(block)
	outBlock.Nonce = p.NewNonce()
	p.Publish(
		outBlock,
		"block",
	)
}
