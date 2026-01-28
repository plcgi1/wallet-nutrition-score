package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wallet-nutrition-score/config"
	_ "wallet-nutrition-score/docs"
	"wallet-nutrition-score/internal/aggregator"
	"wallet-nutrition-score/internal/cache"
	"wallet-nutrition-score/internal/checker"
	"wallet-nutrition-score/internal/entity"
	"wallet-nutrition-score/internal/provider"
	"wallet-nutrition-score/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Wallet Nutrition Score API
// @version 1.0
// @description API for checking wallet security and calculating nutrition score
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
func main() {
	// Инициализация конфигурации
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Инициализация логирования
	log, err := logger.New(cfg.App.LogLevel)
	if err != nil {
		panic(err)
	}

	log.Info("Application starting up")

	// Инициализация провайдеров
	goplusClient := provider.NewGoPlusClient(cfg, log)
	etherscanClient := provider.NewEtherscanClient(cfg, log)

	// Инициализация Redis кэша
	var redisCache cache.Cache
	redisCache, err = cache.NewRedisCache(cfg, log)
	if err != nil {
		log.Warnf("Failed to initialize Redis cache: %v. Cache will not be available.", err)
	}

	// Инициализация фабрики проверок
	checkerFactory := checker.NewFactory(cfg, goplusClient, etherscanClient, log)

	// Инициализация агрегатора
	aggregatorService := aggregator.NewService(cfg, checkerFactory, redisCache, log)

	// Настройка Gin
	if cfg.App.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Swagger endpoint
	// url := ginSwagger.URL("http://localhost:8080/swagger/doc.json")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Обработчики
	r.GET("/health", healthCheckHandler(log))
	r.POST("/api/check", checkWalletHandler(aggregatorService, log))

	// Запуск сервера
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: r,
	}

	// Graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		log.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Server shutdown failed: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Infof("Server starting on port %d", cfg.App.Port)
	log.Infof("Swagger documentation available at http://localhost:%d/swagger/index.html", cfg.App.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server startup failed: %v", err)
	}

	<-idleConnsClosed
	log.Info("Server stopped")
}

// healthCheckHandler - Проверка статуса сервиса
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthCheckHandler(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	}
}

// CheckWalletRequest - Запрос на проверку кошелька
type CheckWalletRequest struct {
	Address string `json:"address" validate:"required,eth_addr"`
}

// CheckWalletResponse - Ответ с результатом проверки кошелька
type CheckWalletResponse struct {
	*entity.WalletReport `json:",inline"`
}

// checkWalletHandler - Обработчик проверки кошелька
// @Summary Check wallet security
// @Description Check wallet security and get nutrition score
// @Tags wallet
// @Accept  json
// @Produce  json
// @Param request body CheckWalletRequest true "Wallet address to check"
// @Success 200 {object} CheckWalletResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/check [post]
func checkWalletHandler(service *aggregator.Service, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CheckWalletRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			log.Errorf("Failed to parse request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid request format",
			})
			return
		}

		// Валидация запроса
		if err := validateRequest(&req); err != nil {
			log.Errorf("Validation failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		ctx := c.Request.Context()
		report, err := service.CheckWallet(ctx, req.Address)
		if err != nil {
			log.Errorf("Check wallet failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check wallet",
			})
			return
		}

		c.JSON(http.StatusOK, CheckWalletResponse{
			WalletReport: report,
		})
	}
}

// validateRequest - Валидация запроса с помощью validator
func validateRequest(req *CheckWalletRequest) error {
	validate := validator.New()

	// Кастомный валидатор для Ethereum адресов
	validate.RegisterValidation("eth_addr", func(fl validator.FieldLevel) bool {
		address := fl.Field().String()
		if len(address) != 42 {
			return false
		}
		if address[:2] != "0x" {
			return false
		}
		// Проверка на наличие только hex символов
		for _, c := range address[2:] {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	})

	return validate.Struct(req)
}
