package followsvc

import (
	"fmt"

	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/Plotrader/infractructure/exchanges/binancefutures"
	"github.com/H3Cki/Plotrader/infractructure/floader"
	"go.uber.org/zap"
)

var (
	exDirPath = "data/exchange_infos"
)

func parseExchange(logger *zap.SugaredLogger, ex inbound.Exchange) (outbound.Exchange, error) {
	switch ex.Name {
	case "BINANCE_FUTURES":
		ucfg := binancefutures.UserConfig{}
		if err := ex.UnmarshalConfig(&ucfg); err != nil {
			return nil, err
		}
		eier, err := floader.NewPrefixed[binancefutures.ExchangeInfo](exDirPath)
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
