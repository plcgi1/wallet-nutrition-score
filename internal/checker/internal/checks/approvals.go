package checks

import (
	"context"
	"fmt"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"

	"github.com/sirupsen/logrus"
)

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
	c.log.Debug("Checking approvals for address: %s", address)

	// Получаем данные из GoPlus API
	resp, err := c.goplusProvider.GetTokenApprovals(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get token approvals: %w", err)
	}

	// Анализируем результаты
	var riskyApprovals []entity.ApprovalInfo
	var targetApprovals []provider.ApprovedSpender
	for _, tokenApproval := range resp.Result {
		var isRisky bool
		for _, approval := range tokenApproval.ApprovedList {
			if approval.ApprovedAmount == "Unlimited" {
				targetApprovals = append(targetApprovals, approval)
			}
		}
		if tokenApproval.MaliciousAddress > 0 {
			isRisky = true
		}
		// Определяем, является ли approval рискованным
		// switch {
		// case approval.MaliciousAddress:
		// 	isRisky = true
		// case approval.Exposure > 0:
		// 	isRisky = true
		// case approval.ApprovalType == "Unlimited":
		// 	isRisky = true
		// }

		if isRisky {
			riskyApprovals = append(riskyApprovals, entity.ApprovalInfo{
				TokenAddress:    tokenApproval.TokenAddress,
				TokenName:       tokenApproval.TokenName,
				SpenderAddress:  targetApproval.ApprovedContract,
				ExposureBalance: 0,
				IsUnlimited:     targetApproval.ApprovedAmount == "Unlimited",
				IsMalicious:     tokenApproval.MaliciousAddress > 0,
			})
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
