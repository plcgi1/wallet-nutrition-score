package checker

import (
	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/internal/checker/internal/checks"
	"alpha-hygiene-backend/internal/provider"

	"github.com/sirupsen/logrus"
)

// Factory - Фабрика для создания проверок
type Factory struct {
	cfg            *config.Config
	goplusProvider *provider.GoPlusClient
	etherscan      *provider.EtherscanClient
	alchemy        *provider.AlchemyClient
	log            *logrus.Entry
}

// NewFactory - Создает новую фабрику проверок
func NewFactory(cfg *config.Config, goplusProvider *provider.GoPlusClient, etherscan *provider.EtherscanClient, alchemy *provider.AlchemyClient, log *logrus.Entry) *Factory {
	return &Factory{
		cfg:            cfg,
		goplusProvider: goplusProvider,
		etherscan:      etherscan,
		alchemy:        alchemy,
		log:            log,
	}
}

// CreateCheck - Создает проверку по типу
func (f *Factory) CreateCheck(t CheckType) IHealthCheck {
	switch t {
	case CheckApprovals:
		return checks.NewApprovalsCheck(f.goplusProvider, f.etherscan, f.cfg, f.log)
	case CheckScamTokens:
		return checks.NewScamTokensCheck(f.goplusProvider, f.alchemy, f.cfg, f.log)
	case CheckAssets:
		return checks.NewAssetCompositionCheck(f.goplusProvider, f.alchemy, f.cfg, f.log)
	case CheckNFT:
		return checks.NewDeadNFTCheck(f.goplusProvider, f.alchemy, f.cfg, f.log)
	default:
		return nil
	}
}

// GetAllCheckTypes - Возвращает все доступные типы проверок
func GetAllCheckTypes() []CheckType {
	return []CheckType{
		CheckApprovals,
		CheckScamTokens,
		CheckAssets,
		CheckNFT,
	}
}
