package checks

import (
	"context"
	"fmt"
	"strconv"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"

	"github.com/sirupsen/logrus"
)

// parseCreatorPercent - Пытается преобразовать строковой процент создателя в float64
func parseCreatorPercent(percentStr string) (float64, error) {
	percentStr = trimString(percentStr)
	if percentStr == "" {
		return 0.0, fmt.Errorf("empty creator percent")
	}
	return strconv.ParseFloat(percentStr, 64)
}

// parseTax - Пытается преобразовать строковой налог в float64
func parseTax(taxStr string) (float64, error) {
	taxStr = trimString(taxStr)
	if taxStr == "" {
		return 0.0, fmt.Errorf("empty tax")
	}
	return strconv.ParseFloat(taxStr, 64)
}

// trimString - Удаляет возможные лишние символы из строки
func trimString(s string) string {
	if s == "" {
		return ""
	}
	// Удаляем возможные символы процента или другие нецифровые символы
	trimmed := []byte{}
	for _, b := range []byte(s) {
		if (b >= '0' && b <= '9') || b == '.' {
			trimmed = append(trimmed, b)
		}
	}
	return string(trimmed)
}

// адреса которые уже в черных списках
// https://chainabuse.com/reports?query=honeypot
// 0xaff8ed5415b68ab81786200e3bfd74d7c37df31e DOPP
// 0x060360e3f44c2e8ad7b48c3e3a5533075939aee6 XMTP
// 0x83e0bac227956aab8834324e31e9420505760db9 ZYNK
// 0xd7a7c5bd399df4aeadd2cf9ac19b0276afd7e7a6 ALTS
// 0x6a100890d851867a97dbd560874ad0f6f5253eb1 BRZ
// 0xb279577df4e083228c2f797fdade203fc7848feb HAR
// Honeypot tokens , created by same addresses 0xD8c4cA0809Be7A908779405C02EEA87AfB98c836 contract creator.
// 0x5ecCBfFb378463782069080F7ACD6d2E7199d903 , possible wallet 0xA961A268cE78DBB24Ac55239609535Fab4B0bef2 for receiving funds.

// https://chainabuse.com/reports?query=fake+airdrop
// 0xdaecFaE58c531cF68cf8651A2fdA75AFF1C56dD4
// 0x794a299251d9b4129048567d291128d17e7fcae3
// DRAIN TX: 0xdc46ccd32bb3a2cf3b8f4249839fad727063f7058111e1ae69d76f6762976e40
// MALICIOUS CONTRACT: 0x4ee879f39cce3c4ca80e2ee90f9df5afeeaeb220
// Contract Explorer: https://etherscan.io/address/0x4ee879f39cce3c4ca80e2ee90f9df5afeeaeb220

// ScamTokensCheck - Проверка на скам-токены
type ScamTokensCheck struct {
	goPlusProvider *provider.GoPlusClient
	alchemyClient  *provider.AlchemyClient
	cfg            *config.Config
	log            *logrus.Entry
}

// NewScamTokensCheck - Создает новую проверку на скам-токены
func NewScamTokensCheck(goPlusProvider *provider.GoPlusClient, alchemyClient *provider.AlchemyClient, cfg *config.Config, log *logrus.Entry) *ScamTokensCheck {
	logger := log.WithFields(logrus.Fields{"component": "scam_tokens"})
	return &ScamTokensCheck{
		goPlusProvider: goPlusProvider,
		alchemyClient:  alchemyClient,
		cfg:            cfg,
		log:            logger,
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
	tokens, err := c.alchemyClient.GetERC20Tokens(ctx, address)
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
			// Проверяем на потенциальные скам-токены по нескольким критериям:

			// 1. Токен нельзя купить или продать
			if info.CannotBuy == "1" {
				scamTokens = append(scamTokens, addr)
				continue
			}

			// 2. Токен не добавлен в ни один DEX
			if info.IsInDex == "0" {
				scamTokens = append(scamTokens, addr)
				continue
			}

			// 3. Высокий процент владения создателем (более 50%) - потенциальный Rug Pull
			creatorPercent, err := parseCreatorPercent(info.CreatorPercent)
			if err == nil && creatorPercent > 50.0 {
				scamTokens = append(scamTokens, addr)
				continue
			}

			// 4. Высокие налоги на покупку или продажу (более 10%)
			buyTax, err := parseTax(info.BuyTax)
			if err == nil && buyTax > 10.0 {
				scamTokens = append(scamTokens, addr)
				continue
			}

			sellTax, err := parseTax(info.SellTax)
			if err == nil && sellTax > 10.0 {
				scamTokens = append(scamTokens, addr)
				continue
			}

			// 5. Создаатель имеет 100% токенов
			if info.CreatorPercent == "100.000000" || info.CreatorBalance == info.TotalSupply {
				scamTokens = append(scamTokens, addr)
				continue
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
