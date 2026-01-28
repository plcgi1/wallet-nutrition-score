package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"wallet-nutrition-score/config"
	"wallet-nutrition-score/pkg/util"

	"github.com/sirupsen/logrus"
)

// AlchemyClient - Клиент для Alchemy API
type AlchemyClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
	log     *logrus.Entry
}

// NewAlchemyClient - Создает новый клиент для Alchemy API
func NewAlchemyClient(cfg *config.Config, log *logrus.Entry) *AlchemyClient {
	baseURL := cfg.Alchemy.URL
	if baseURL == "" {
		baseURL = "https://eth-mainnet.g.alchemy.com/v2"
	}
	logger := log.WithFields(logrus.Fields{"component": "alchemy"})
	return &AlchemyClient{
		apiKey:  cfg.Alchemy.ApiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: logger,
	}
}

// AlchemyTokenBalance - Структура для баланса токена из Alchemy API
type AlchemyTokenBalance struct {
	ContractAddress string `json:"contractAddress"`
	TokenName       string `json:"name"`
	TokenSymbol     string `json:"symbol"`
	Decimals        string `json:"decimals"`
	Balance         string `json:"balance"`
}

// GetETHBalance - Получает баланс ETH для адреса (в ETH)
func (c *AlchemyClient) GetETHBalance(ctx context.Context, address string) (float64, error) {
	urlStr := fmt.Sprintf("%s/%s", c.baseURL, c.apiKey)

	reqBody := map[string]interface{}{
		"id":      1,
		"jsonrpc": "2.0",
		"method":  "eth_getBalance",
		"params":  []interface{}{address, "latest"},
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		c.log.Errorf("Failed to marshal request: %v", err)
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewBuffer(reqData))
	if err != nil {
		c.log.Errorf("Failed to create request: %v", err)
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("Alchemy API request failed: %v", err)
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Errorf("Failed to read response body: %v", err)
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("Alchemy API error: %s", string(body))
		return 0, fmt.Errorf("Alchemy API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		c.log.Errorf("Failed to unmarshal response: %v", err)
		return 0, err
	}

	// Преобразуем из hex в wei (big.Int), затем в ETH (float64)
	balanceWei, ok := new(big.Int).SetString(response.Result[2:], 16)
	if !ok {
		c.log.Errorf("Failed to parse balance: %s", response.Result)
		return 0, fmt.Errorf("failed to parse balance")
	}

	// 1 ETH = 1e18 wei
	ethBalance := new(big.Float).SetInt(balanceWei)
	ethDivisor := new(big.Float).SetFloat64(1e18)
	ethBalance.Quo(ethBalance, ethDivisor)

	var ethFloat float64
	ethFloat, _ = ethBalance.Float64()

	return ethFloat, nil
}

type AlchemyERC20Response struct {
	Result struct {
		TokenBalances []struct {
			ContractAddress string `json:"contractAddress"`
			TokenBalance    string `json:"tokenBalance"`
		} `json:"tokenBalances"`
	} `json:"result"`
}

// GetERC20Tokens - Получает список ERC20 токенов для адреса
func (c *AlchemyClient) GetERC20Tokens(ctx context.Context, address string) ([]*TokenBalance, error) {
	urlStr := fmt.Sprintf("%s/%s", c.baseURL, c.apiKey)

	reqBody := map[string]interface{}{
		"id":      1,
		"jsonrpc": "2.0",
		"method":  "alchemy_getTokenBalances",
		"params":  []interface{}{address},
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		c.log.Errorf("Failed to marshal request: %v", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewBuffer(reqData))
	if err != nil {
		c.log.Errorf("Failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("Alchemy API request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Errorf("Failed to read response body: %v", err)
		return nil, err
	}

	// c.log.Debugf("Alchemy API response for address %s: %s", address, string(body))

	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("Alchemy API error: %s", string(body))
		return nil, fmt.Errorf("Alchemy API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response AlchemyERC20Response

	err = json.Unmarshal(body, &response)
	if err != nil {
		c.log.Errorf("Failed to unmarshal response: %v", err)
		return nil, err
	}

	c.log.Debugf("Found %d ERC20 tokens for address %s", len(response.Result.TokenBalances), address)

	// Преобразуем в нашу структуру TokenBalance
	var result []*TokenBalance
	for _, tb := range response.Result.TokenBalances {
		// Для получения дополнительной информации (name, symbol, decimals) нужно сделать отдельный запрос
		// Но для MVP мы можем использовать пустые значения

		// Конвертируем баланс из hex строки в decimal строку
		var balanceStr string
		if tb.TokenBalance == "0x" {
			balanceStr = "0"
		} else {
			balanceWei, ok := new(big.Int).SetString(tb.TokenBalance[2:], 16)
			if ok {
				balanceStr = balanceWei.String()
			} else {
				c.log.Warnf("Failed to parse token balance for contract %s: %s", tb.ContractAddress, tb.TokenBalance)
				balanceStr = "0"
			}
		}

		result = append(result, &TokenBalance{
			Account:         address,
			ContractAddress: tb.ContractAddress,
			TokenName:       "",
			TokenSymbol:     "",
			TokenDecimal:    "18",
			Balance:         balanceStr,
		})
	}

	return result, nil
}

// AlchemyNFT - Структура для NFT из Alchemy API
type AlchemyNFT struct {
	ContractAddress string `json:"contractAddress"`
	TokenID         string `json:"tokenId"`
	TokenType       string `json:"tokenType"`
}

type AlchemyNFTApiResponse struct {
	OwnedNfts []struct {
		Contract struct {
			Address string `json:"address"`
		} `json:"contract"`
		Id struct {
			TokenID       string `json:"tokenId"`
			TokenMetadata struct {
				TokenType string `json:"tokenType"`
			} `json:"tokenMetadata"`
		} `json:"id"`
		TokenType string `json:"tokenType"`
	} `json:"ownedNfts"`
	TotalCount int `json:"totalCount"`
}

// GetNFTs - Получает список NFT для адреса
func (c *AlchemyClient) GetNFTs(ctx context.Context, address string) ([]*AlchemyNFT, error) {
	urlStr := fmt.Sprintf("%s/%s/getNFTsForOwner?owner=%s&omitMetadata=true", c.baseURL, c.apiKey, address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		c.log.Errorf("Failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("Alchemy API request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		c.log.Errorf("Failed to read response body: %v", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("Alchemy API error: %s", string(body))
		return nil, fmt.Errorf("Alchemy API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response AlchemyNFTApiResponse

	if err := json.Unmarshal(body, &response); err != nil {
		c.log.Errorf("Failed to unmarshal response: %v", err)
		return nil, err
	}

	var nfts []*AlchemyNFT
	seen := make(map[string]bool)
	for _, nft := range response.OwnedNfts {
		if util.IsTrusted(nft.Contract.Address) {
			continue
		}
		// Проверяем, что у NFT есть контракт и токен ID
		if nft.Contract.Address == "" || nft.Id.TokenID == "" {
			c.log.Warnf("Skipping invalid NFT: contract=%s, tokenId=%s", nft.Contract.Address, nft.Id.TokenID)
			continue
		}

		// Генерируем уникальный ключ для проверки дубликатов
		key := fmt.Sprintf("%s:%s", nft.Contract.Address, nft.Id.TokenID)
		if seen[key] {
			c.log.Warnf("Skipping duplicate NFT: %s", key)
			continue
		}

		seen[key] = true
		nfts = append(nfts, &AlchemyNFT{
			ContractAddress: nft.Contract.Address,
			TokenID:         nft.Id.TokenID,
			TokenType:       nft.Id.TokenMetadata.TokenType,
		})
	}

	c.log.Debugf("Found %d valid NFTs for address %s", len(nfts), address)
	return nfts, nil
}

// Close - Закрывает клиент (не требуется для Alchemy API)
func (c *AlchemyClient) Close() error {
	return nil
}
