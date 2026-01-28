package checks

import (
	"context"
	"fmt"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"

	"github.com/sirupsen/logrus"
)

// AssetCompositionCheck - Проверка состава активов
type AssetCompositionCheck struct {
	goplusProvider *provider.GoPlusClient
	etherscan      *provider.EtherscanClient
	cfg            *config.Config
	log            *logrus.Logger
}

// NewAssetCompositionCheck - Создает новую проверку состава активов
func NewAssetCompositionCheck(goplusProvider *provider.GoPlusClient, etherscan *provider.EtherscanClient, cfg *config.Config, log *logrus.Logger) *AssetCompositionCheck {
	return &AssetCompositionCheck{
		goplusProvider: goplusProvider,
		etherscan:      etherscan,
		cfg:            cfg,
		log:            log,
	}
}

// Name - Возвращает имя проверки
func (c *AssetCompositionCheck) Name() string {
	return "assets"
}

// Execute - Выполняет проверку
func (c *AssetCompositionCheck) Execute(ctx context.Context, address string) (*entity.CheckResult, error) {
	c.log.Debugf("Checking asset composition for address: %s", address)

	// В реальной реализации здесь нужно получить баланс токенов
	// и их USD стоимость для анализа состава активов
	// Для MVP пока возвращаем заглушку
	tokens := []entity.TokenInfo{
		{
			Address:  "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			Name:     "Tether",
			Symbol:   "USDT",
			Balance:  1000,
			USDValue: 1000,
			IsStable: true,
		},
		{
			Address:  "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
			Name:     "Wrapped Bitcoin",
			Symbol:   "WBTC",
			Balance:  0.1,
			USDValue: 6400,
			IsStable: false,
		},
	}

	// Анализируем состав активов
	var totalStable float64
	var totalVolatile float64
	var totalValue float64

	for _, token := range tokens {
		totalValue += token.USDValue
		if token.IsStable {
			totalStable += token.USDValue
		} else {
			totalVolatile += token.USDValue
		}
	}

	riskFound := false
	var scorePenalty float64
	var details string

	if totalValue > 0 {
		stableRatio := (totalStable / totalValue) * 100
		volatileRatio := (totalVolatile / totalValue) * 100

		if volatileRatio > 90 {
			riskFound = true
			scorePenalty = c.cfg.Scoring.Weights["asset_ratio"] * 100
			details = fmt.Sprintf("High volatile assets ratio: %.1f%%", volatileRatio)
		} else {
			details = fmt.Sprintf("Stable assets: %.1f%%, volatile assets: %.1f%%", stableRatio, volatileRatio)
		}
	} else {
		details = "No assets found"
	}

	return &entity.CheckResult{
		CheckName:    c.Name(),
		RiskFound:    riskFound,
		RiskLevel:    entity.RiskLevelMedium,
		ScorePenalty: scorePenalty,
		Details:      details,
		RawData:      tokens,
	}, nil
}
