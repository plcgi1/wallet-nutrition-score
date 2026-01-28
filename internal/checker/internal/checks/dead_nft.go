package checks

import (
	"context"
	"fmt"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"

	"github.com/sirupsen/logrus"
)

// DeadNFTCheck - Проверка на мертвые NFT
type DeadNFTCheck struct {
	goplusProvider *provider.GoPlusClient
	alchemy        *provider.AlchemyClient
	cfg            *config.Config
	log            *logrus.Entry
}

// NewDeadNFTCheck - Создает новую проверку на мертвые NFT
func NewDeadNFTCheck(goplusProvider *provider.GoPlusClient, alchemy *provider.AlchemyClient, cfg *config.Config, log *logrus.Entry) *DeadNFTCheck {
	logger := log.WithFields(logrus.Fields{"component": "dead_nft"})
	return &DeadNFTCheck{
		goplusProvider: goplusProvider,
		alchemy:        alchemy,
		cfg:            cfg,
		log:            logger,
	}
}

// Name - Возвращает имя проверки
func (c *DeadNFTCheck) Name() string {
	return "dead_nft"
}

// Execute - Выполняет проверку
func (c *DeadNFTCheck) Execute(ctx context.Context, address string) (*entity.CheckResult, error) {
	c.log.Debugf("Checking for dead NFTs for address: %s", address)

	// Получаем список NFT на кошельке
	nfts, err := c.alchemy.GetNFTs(ctx, address)
	if err != nil {
		c.log.Errorf("Failed to get NFTs for address %s: %v", address, err)
		return nil, err
	}

	c.log.Debugf("Found %d NFTs for address %s", len(nfts), address)

	// Для MVP определим "мертвые NFT" как NFT с контрактом, которые не являются популярными коллекциями
	// В реальной реализации нужно проверять через GoPlus NFT Security API или другие источники
	var deadNFTs []string
	seen := make(map[string]bool)

	for _, nft := range nfts {
		c.log.Debugf("Processing NFT: contract=%s, tokenId=%s, type=%s", nft.ContractAddress, nft.TokenID, nft.TokenType)
		// Здесь должна быть логика проверки на "мертвые" NFT
		// Например, проверка на:
		// - Наличие торгов на рынках (OpenSea, Blur)
		// - Количество держателей
		// - Активность контракта
		// - Является ли коллекция скемой или фродом

		// Проверяем, что NFT имеет все необходимые поля
		if nft.ContractAddress == "" || nft.TokenID == "" {
			c.log.Warnf("Skipping invalid NFT: contract=%s, tokenId=%s", nft.ContractAddress, nft.TokenID)
			continue
		}

		// Генерируем уникальный ключ для проверки дубликатов
		key := fmt.Sprintf("%s:%s", nft.ContractAddress, nft.TokenID)
		if seen[key] {
			c.log.Warnf("Skipping duplicate NFT: %s", key)
			continue
		}

		seen[key] = true
		deadNFTs = append(deadNFTs, key)
	}

	riskFound := len(deadNFTs) > 0
	var scorePenalty float64
	var details string

	if riskFound {
		scorePenalty = c.cfg.Scoring.Weights["dead_nft"] * 100
		details = fmt.Sprintf("Found %d dead NFTs", len(deadNFTs))
	} else {
		details = "No dead NFTs found"
	}

	return &entity.CheckResult{
		CheckName:    c.Name(),
		RiskFound:    riskFound,
		RiskLevel:    entity.RiskLevelLow,
		ScorePenalty: scorePenalty,
		Details:      details,
		RawData:      deadNFTs,
	}, nil
}
