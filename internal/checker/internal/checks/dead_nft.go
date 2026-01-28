package checks

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"
)

// DeadNFTCheck - Проверка на мертвые NFT
type DeadNFTCheck struct {
	provider *provider.GoPlusClient
	cfg      *config.Config
	log      *logrus.Logger
}

// NewDeadNFTCheck - Создает новую проверку на мертвые NFT
func NewDeadNFTCheck(provider *provider.GoPlusClient, cfg *config.Config, log *logrus.Logger) *DeadNFTCheck {
	return &DeadNFTCheck{
		provider: provider,
		cfg:      cfg,
		log:      log,
	}
}

// Name - Возвращает имя проверки
func (c *DeadNFTCheck) Name() string {
	return "dead_nft"
}

// Execute - Выполняет проверку
func (c *DeadNFTCheck) Execute(ctx context.Context, address string) (*entity.CheckResult, error) {
	c.log.Debugf("Checking for dead NFTs for address: %s", address)

	// В реальной реализации здесь нужно получить список NFT на кошельке
	// и проверить их через GoPlus NFT Security API
	// Для MVP пока возвращаем заглушку
	var deadNFTs []string

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
