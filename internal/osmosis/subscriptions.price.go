package osmosis

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/syntropynet/data-layer-sdk/pkg/service"
	"github.com/syntropynet/price-publisher/pkg/cmc"
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
		p.Logger.Error("Bogus PRICE message", err)
		return
	}

	parts := strings.Split(msg.Subject(), ".")

	err = p.indexer.SetLatestPrice(parts[len(parts)-1], "USD", quote.Price, time.Unix(quote.LastUpdated, 0))
	if err != nil {
		p.Logger.Error("Failed indexing price: ", err)
		return
	}

	p.Logger.Debug("PRICE", "subject", msg.Subject(), "quote", quote)
}
