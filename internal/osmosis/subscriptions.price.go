package osmosis

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Synternet/data-layer-sdk/pkg/service"
	"github.com/Synternet/price-publisher/pkg/cmc"
)

func (p *Publisher) subscribePriceFeed() error {
	priceFeed, err := p.SubscribeTo(p.handlePriceFeed, p.PriceSubject())
	if err != nil {
		return err
	}
	p.priceFeed = priceFeed
	return nil
}

func (p *Publisher) handlePriceFeed(msg service.Message) {
	var quote cmc.QuoteInfo
	err := json.Unmarshal(msg.Data(), &quote)
	if err != nil {
		p.Logger.Error("unmarshalling PRICE message", "err", err)
		return
	}

	p.pricesCounter.Add(1)

	parts := strings.Split(msg.Subject(), ".")

	err = p.indexer.SetLatestPrice(parts[len(parts)-1], "USD", quote.Price, time.Unix(quote.LastUpdated, 0))
	if err != nil {
		p.Logger.Error("indexing price: ", "err", err)
		return
	}

	p.Logger.Debug("PRICE", "subject", msg.Subject(), "quote", quote)
}
