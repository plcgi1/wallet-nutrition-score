package checks

import (
	"context"
	"fmt"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"

	"github.com/sirupsen/logrus"
)

// ScamTokensCheck - Проверка на скам-токены
type ScamTokensCheck struct {
	goPlusProvider  *provider.GoPlusClient
	etherscanClient *provider.EtherscanClient
	cfg             *config.Config
	log             *logrus.Logger
}

// NewScamTokensCheck - Создает новую проверку на скам-токены
func NewScamTokensCheck(goPlusProvider *provider.GoPlusClient, etherscanClient *provider.EtherscanClient, cfg *config.Config, log *logrus.Logger) *ScamTokensCheck {
	return &ScamTokensCheck{
		goPlusProvider:  goPlusProvider,
		etherscanClient: etherscanClient,
		cfg:             cfg,
		log:             log,
	}
}

// Name - Возвращает имя проверки
func (c *ScamTokensCheck) Name() string {
	return "scam_tokens"
}

// Execute - Выполняет проверку
func (c *ScamTokensCheck) Execute(ctx context.Context, address string) (*entity.CheckResult, error) {
	c.log.Debugf("Checking for scam tokens for address: %s", address)

	// Получаем список токенов на кошельке
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

	c.log.Debugf("Found %d tokens to check for scams", len(tokenAddresses))

	// Проверяем токены через GoPlus API
	var scamTokens []string
	if len(tokenAddresses) > 0 {
		securityResult, err := c.goPlusProvider.GetTokenSecurity(ctx, tokenAddresses)
		if err != nil {
			c.log.Errorf("Failed to check token security for address %s: %v", address, err)
			return nil, err
		}

		// Анализируем результаты и собираем скам-токены
		for addr, info := range securityResult.Result {
			if info.IsBlacklisted || info.IsFakeToken || info.IsHoneypot || info.CannotSellAll {
				scamTokens = append(scamTokens, addr)
			}
		}
	}

	riskFound := len(scamTokens) > 0
	var scorePenalty float64
	var details string

	if riskFound {
		scorePenalty = c.cfg.Scoring.Weights["scam_tokens"] * 100
		details = fmt.Sprintf("Found %d scam tokens", len(scamTokens))
	} else {
		details = "No scam tokens found"
	}

	return &entity.CheckResult{
		CheckName:    c.Name(),
		RiskFound:    riskFound,
		RiskLevel:    entity.RiskLevelHigh,
		ScorePenalty: scorePenalty,
		Details:      details,
		RawData:      scamTokens,
	}, nil
}
