package checks

import (
	"context"
	"fmt"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"

	"github.com/sirupsen/logrus"
)

// RugPullCheck - Проверка на rug pull контакты
type RugPullCheck struct {
	goPlusProvider  *provider.GoPlusClient
	etherscanClient *provider.EtherscanClient
	cfg             *config.Config
	log             *logrus.Logger
}

// NewRugPullCheck - Создает новую проверку на rug pull
func NewRugPullCheck(goPlusProvider *provider.GoPlusClient, etherscanClient *provider.EtherscanClient, cfg *config.Config, log *logrus.Logger) *RugPullCheck {
	return &RugPullCheck{
		goPlusProvider:  goPlusProvider,
		etherscanClient: etherscanClient,
		cfg:             cfg,
		log:             log,
	}
}

// Name - Возвращает имя проверки
func (c *RugPullCheck) Name() string {
	return "rug_pull"
}

// Execute - Выполняет проверку
func (c *RugPullCheck) Execute(ctx context.Context, address string) (*entity.CheckResult, error) {
	c.log.Debugf("Checking for rug pull interactions for address: %s", address)

	// Получаем список токенов на кошельке и историю транзакций с ними
	tokens, err := c.etherscanClient.GetERC20Tokens(ctx, address)
	if err != nil {
		c.log.Errorf("Failed to get ERC20 tokens for address %s: %v", address, err)
		return nil, err
	}

	// Извлекаем адреса токенов для проверки
	var tokenAddresses []string
	for _, token := range tokens {
		tokenAddresses = append(tokenAddresses, token.ContractAddress)
	}

	c.log.Debugf("Found %d tokens to check for rug pull risk", len(tokenAddresses))

	// Проверяем токены на риск rug pull через GoPlus API
	var rugPullInteractions []string
	if len(tokenAddresses) > 0 {
		securityResult, err := c.goPlusProvider.GetTokenSecurity(ctx, tokenAddresses)
		if err != nil {
			c.log.Errorf("Failed to check token security for address %s: %v", address, err)
			return nil, err
		}

		// Анализируем результаты и собираем токены с риском rug pull
		for addr, info := range securityResult.Result {
			if info.IsHoneypot || info.CannotSellAll {
				rugPullInteractions = append(rugPullInteractions, addr)
			}
		}
	}

	riskFound := len(rugPullInteractions) > 0
	var scorePenalty float64
	var details string

	if riskFound {
		scorePenalty = c.cfg.Scoring.Weights["rug_pulls"] * 100
		details = fmt.Sprintf("Found %d tokens with rug pull risk", len(rugPullInteractions))
	} else {
		details = "No rug pull interactions found"
	}

	return &entity.CheckResult{
		CheckName:    c.Name(),
		RiskFound:    riskFound,
		RiskLevel:    entity.RiskLevelCritical,
		ScorePenalty: scorePenalty,
		Details:      details,
		RawData:      rugPullInteractions,
	}, nil
}
