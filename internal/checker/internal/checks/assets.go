package checks

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"
	"wallet-nutrition-score/pkg/util"

	"github.com/sirupsen/logrus"
)

// AssetCompositionCheck - Проверка состава активов
type AssetCompositionCheck struct {
	goplusProvider *provider.GoPlusClient
	alchemy        *provider.AlchemyClient
	cfg            *config.Config
	log            *logrus.Logger
}

// NewAssetCompositionCheck - Создает новую проверку состава активов
func NewAssetCompositionCheck(goplusProvider *provider.GoPlusClient, alchemy *provider.AlchemyClient, cfg *config.Config, log *logrus.Logger) *AssetCompositionCheck {
	return &AssetCompositionCheck{
		goplusProvider: goplusProvider,
		alchemy:        alchemy,
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

	// Получаем список токенов на кошельке
	tokens, err := c.alchemy.GetERC20Tokens(ctx, address)
	if err != nil {
		c.log.Errorf("Failed to get ERC20 tokens for address %s: %v", address, err)
		return nil, err
	}

	c.log.Debugf("Found %d ERC20 tokens for address %s", len(tokens), address)

	// Получаем баланс ETH для кошелька
	ethBalance, err := c.alchemy.GetETHBalance(ctx, address)

	if err != nil {
		c.log.Errorf("Failed to get ETH balance for address %s: %v", address, err)
		return nil, err
	}

	c.log.Debugf("ETH balance for address %s: %.6f", address, ethBalance)

	// // Создаем список известных стабильных токенов
	// stablecoins := map[string]bool{
	// 	"0xdAC17F958D2ee523a2206206994597C13D831ec7": true, // USDT
	// 	"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48": true, // USDC
	// 	"0x6B175474E89094C44Da98b954EedeAC495271d0F": true, // DAI
	// 	"0x1456688345527bE1f37E9e627DA0837D6f08C925": true, // GUSD
	// 	"0x57Ab1ec28D129707052df4dF418D58a2D46d5f51": true, // sUSD
	// 	"0x0000000000085d4780B73119b644AE5ecd22b376": true, // TUSD
	// }

	// Преобразуем токены в нашу структуру TokenInfo
	var tokenInfos []entity.TokenInfo

	// Добавляем ETH как отдельный токен
	// Для MVP используем фиксированную цену ETH (в реальности нужно получать из API)
	ethUSDPrice := 2000.0
	if ethBalance > 0 {
		tokenInfos = append(tokenInfos, entity.TokenInfo{
			Address:    "0x0000000000000000000000000000000000000000",
			AddressURL: util.GetAdressURL("0x0000000000000000000000000000000000000000"),
			Name:       "Ethereum",
			Symbol:     "ETH",
			Balance:    ethBalance,
			USDValue:   ethBalance * ethUSDPrice,
			IsStable:   false,
		})
	}

	// Обрабатываем ERC20 токены
	for _, token := range tokens {
		// Пропускаем токены с нулевым балансом
		if token.Balance == "0" {
			continue
		}

		// Преобразуем баланс из строки в число с учетом десятичных знаков
		balanceInt, ok := new(big.Int).SetString(token.Balance, 10)
		if !ok {
			c.log.Warnf("Failed to parse token balance: %s", token.Balance)
			continue
		}

		// Получаем количество десятичных знаков
		decimals := 18
		if token.TokenDecimal != "" {
			if d, err := strconv.Atoi(token.TokenDecimal); err == nil {
				decimals = d
			}
		}

		// Преобразуем в читаемый формат
		balance := new(big.Float).SetInt(balanceInt)
		decimalsFactor := new(big.Float).SetFloat64(float64(10) * float64(decimals))
		balanceFloat, _ := new(big.Float).Quo(balance, decimalsFactor).Float64()

		// Для MVP используем фиксированную цену для стабильных токенов и нулевую для других
		var usdValue float64
		isStable := util.IsTrusted(token.ContractAddress)
		if isStable {
			usdValue = balanceFloat
		} else {
			// Для нестабильных токенов в MVP можно установить нулевую цену
			// В реальной реализации нужно получать цены из API (например, CoinGecko, CoinMarketCap)
			usdValue = 0
		}
		fmt.Printf("token %v \n", token.TokenID)
		tokenInfos = append(tokenInfos, entity.TokenInfo{
			Address:    token.ContractAddress,
			AddressURL: util.GetAdressURL(token.ContractAddress),
			Name:       token.TokenName,
			Symbol:     token.TokenSymbol,
			Balance:    balanceFloat,
			USDValue:   usdValue,
			IsStable:   isStable,
		})
	}

	// Анализируем состав активов
	var totalStable float64
	var totalVolatile float64
	var totalValue float64

	for _, token := range tokenInfos {
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

	c.log.Debugf("Asset composition check completed for address %s: %s", address, details)

	return &entity.CheckResult{
		CheckName:    c.Name(),
		RiskFound:    riskFound,
		RiskLevel:    entity.RiskLevelMedium,
		ScorePenalty: scorePenalty,
		Details:      details,
		RawData:      tokenInfos,
	}, nil
}
