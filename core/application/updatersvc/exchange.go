package updatersvc

import (
	"fmt"

	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/Plotrader/infractructure/eier"
	"github.com/H3Cki/Plotrader/infractructure/exchanges/binancefutures"
	"go.uber.org/zap"
)

var (
	exDirPath = "data/exchange_infos"
)

func parseExchange(logger *zap.SugaredLogger, ex inbound.ExchangeConfig) (outbound.Exchange, error) {
	switch ex.Name {
	case "binancefutures":
		ucfg := binancefutures.UserConfig{}
		if err := ex.MarshalConfig(&ucfg); err != nil {
			return nil, err
		}
		eier, err := eier.New[binancefutures.ExchangeInfo](exDirPath)
		if err != nil {
			return nil, err
		}
		cfg := binancefutures.Config{
			ExchangeInfoer: eier,
			UserConfig:     ucfg,
		}
		return binancefutures.New(logger, cfg), nil
	}
	return nil, fmt.Errorf("unknown exchange: %s", ex.Name)
}
