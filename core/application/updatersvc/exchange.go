package updatersvc

import (
	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/Plotrader/infractructure/exchanges/binancefutures"
	"github.com/H3Cki/Plotrader/infractructure/exchanges/noop"
	"go.uber.org/zap"
)

func parseExchange(logger *zap.SugaredLogger, ex inbound.ExchangeConfig) (outbound.Exchange, error) {
	switch ex.Name {
	case "binancefutures":
		cfg := binancefutures.Config{}
		if err := ex.MarshalConfig(&cfg); err != nil {
			return nil, err
		}
		return binancefutures.New(logger, cfg), nil
	case "noop":
		e := &noop.Exchange{}
		if err := ex.MarshalConfig(&e); err != nil {
			return nil, err
		}
		return e, nil
	}

	return nil, nil
}
