package checker

import (
	"context"

	"alpha-hygiene-backend/internal/entity"
)

// IHealthCheck - Контракт для любой проверки
type IHealthCheck interface {
	// Name возвращает уникальный идентификатор проверки
	Name() string
	// Execute выполняет проверку и возвращает результат
	Execute(ctx context.Context, address string) (*entity.CheckResult, error)
}

// CheckType - Типы проверок
type CheckType string

const (
	CheckApprovals  CheckType = "approvals"
	CheckRugPull    CheckType = "rug_pull"
	CheckAssets     CheckType = "assets"
	CheckScamTokens CheckType = "scam_tokens"
	CheckNFT        CheckType = "dead_nft"
)
