package checks

import (
	"context"
	"fmt"
	"math/big"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"
	"wallet-nutrition-score/pkg/util"

	"github.com/sirupsen/logrus"
)

//
// 1. Где взять адреса с Dangerous Approvals (Опасные разрешения)
// 	Самый верный способ найти кошельки с плохими аппрувами — посмотреть на жертв дрейнеров (Wallet Drainers).
// Это люди, которые подписали Approve мошеннику, и их обокрали.
// Но часто у них остаются другие неотмененные аппрувы.
// 	Алгоритм поиска:
// 		Зайдите на Etherscan.
// 		В поиске введите тег: Fake_Phishing или найдите известные адреса дрейнеров (например, Pink Drainer или Inferno Drainer).
// 		Откройте адрес мошенника.
// 		Перейдите во вкладку Transactions (или ERC-20 Token Txns).
// 		Посмотрите на колонку From. Адреса, которые отправляли токены на адрес мошенника — это жертвы.
// 		Возьмите любой адрес из колонки From. Скорее всего, у этого кошелька куча «висящих» аппрувов на фишинговые контракты.
// Пример для теста (жертвы хаков):
// 		Посмотрите входящие транзакции на адреса, помеченные как Exploit или Heist.
// Поищите "Orbit Bridge Exploiter" или "Multichain Exploiter" на Etherscan. Входящие переводы шли от пострадавших пользователей.

// ApprovalsCheck - Проверка токен approvals
type ApprovalsCheck struct {
	goplusProvider *provider.GoPlusClient
	etherscan      *provider.EtherscanClient
	cfg            *config.Config
	log            *logrus.Logger
}

// NewApprovalsCheck - Создает новую проверку approvals
func NewApprovalsCheck(goplusProvider *provider.GoPlusClient, etherscan *provider.EtherscanClient, cfg *config.Config, log *logrus.Logger) *ApprovalsCheck {
	logger := log.WithFields(logrus.Fields{"component": "approvals"}).Logger
	return &ApprovalsCheck{
		goplusProvider: goplusProvider,
		etherscan:      etherscan,
		cfg:            cfg,
		log:            logger,
	}
}

// Name - Возвращает имя проверки
func (c *ApprovalsCheck) Name() string {
	return "approvals"
}

// Execute - Выполняет проверку
func (c *ApprovalsCheck) Execute(ctx context.Context, address string) (*entity.CheckResult, error) {
	c.log.Debugf("Checking approvals for address: %s", address)

	// Получаем данные из GoPlus API
	resp, err := c.goplusProvider.GetTokenApprovals(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get token approvals: %w", err)
	}

	// Анализируем результаты
	var riskyApprovals []entity.ApprovalInfo
	for _, tokenApproval := range resp.Result {
		for _, approval := range tokenApproval.ApprovedList {
			var isRisky bool

			// Определяем, является ли approval рискованным
			switch {
			case tokenApproval.MaliciousAddress > 0:
				isRisky = true
			case approval.ApprovedAmount == "Unlimited":
				isRisky = true
			case approval.AddressInfo.DoubtList > 0:
				isRisky = true
			case len(approval.AddressInfo.MaliciousBehavior) > 0:
				isRisky = true
			}

			if isRisky {
				// Calculate exposure balance
				exposureBalance := calculateExposureBalance(
					approval.ApprovedAmount,
					tokenApproval.Balance,
					tokenApproval.Decimals,
				)

				riskyApprovals = append(riskyApprovals, entity.ApprovalInfo{
					TokenAddress:    tokenApproval.TokenAddress,
					TokenURL:        util.GetAdressURL(tokenApproval.TokenAddress),
					TokenName:       tokenApproval.TokenName,
					SpenderAddress:  approval.ApprovedContract,
					SpenderURL:      util.GetAdressURL(approval.ApprovedContract),
					ExposureBalance: exposureBalance,
					IsUnlimited:     approval.ApprovedAmount == "Unlimited",
					IsMalicious:     tokenApproval.MaliciousAddress > 0 || len(approval.AddressInfo.MaliciousBehavior) > 0,
				})
			}
		}
	}

	// Рассчитываем штраф
	var scorePenalty float64
	var details string
	riskFound := len(riskyApprovals) > 0

	if riskFound {
		scorePenalty = c.cfg.Scoring.Weights["approvals"] * 100
		details = fmt.Sprintf("Found %d risky approvals", len(riskyApprovals))
	} else {
		details = "No risky approvals found"
	}

	return &entity.CheckResult{
		CheckName:    c.Name(),
		RiskFound:    riskFound,
		RiskLevel:    c.determineMaxRiskLevel(riskyApprovals),
		ScorePenalty: scorePenalty,
		Details:      details,
		RawData:      riskyApprovals,
	}, nil
}

// determineMaxRiskLevel - Определяет максимальный уровень риска
func (c *ApprovalsCheck) determineMaxRiskLevel(approvals []entity.ApprovalInfo) entity.RiskLevel {
	maxLevel := entity.RiskLevelLow

	for _, approval := range approvals {
		var level entity.RiskLevel
		switch {
		case approval.IsMalicious:
			level = entity.RiskLevelCritical
		case approval.ExposureBalance > 0:
			level = entity.RiskLevelCritical
		case approval.IsUnlimited:
			level = entity.RiskLevelHigh
		}

		if c.isHigherRisk(level, maxLevel) {
			maxLevel = level
		}
	}

	return maxLevel
}

// isHigherRisk - Сравнивает уровни риска
func (c *ApprovalsCheck) isHigherRisk(a, b entity.RiskLevel) bool {
	levels := map[entity.RiskLevel]int{
		entity.RiskLevelLow:      1,
		entity.RiskLevelMedium:   2,
		entity.RiskLevelHigh:     3,
		entity.RiskLevelCritical: 4,
	}

	return levels[a] > levels[b]
}

// calculateExposureBalance - Calculates exposure balance
func calculateExposureBalance(approvedAmount, tokenBalance string, decimals int) float64 {
	if approvedAmount == "Unlimited" {
		// If unlimited, exposure is total token balance
		balance, err := parseTokenAmount(tokenBalance, decimals)
		if err != nil {
			return 0
		}
		return balance
	}

	// If limited, exposure is min(approvedAmount, tokenBalance)
	approved, err1 := parseTokenAmount(approvedAmount, decimals)
	balance, err2 := parseTokenAmount(tokenBalance, decimals)

	if err1 != nil || err2 != nil {
		return 0
	}

	if approved < balance {
		return approved
	}

	return balance
}

// parseTokenAmount - Parse token amount from string to float64 considering decimals
func parseTokenAmount(amountStr string, decimals int) (float64, error) {
	amountBig, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return 0, fmt.Errorf("failed to parse amount: %s", amountStr)
	}

	// Convert to float64 with decimals
	decimalFactor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountFloat := new(big.Float).SetInt(amountBig)
	amountFloat.Quo(amountFloat, decimalFactor)

	result, _ := amountFloat.Float64()
	return result, nil
}
