package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"alpha-hygiene-backend/config"

	"github.com/sirupsen/logrus"
)

// EtherscanClient - Клиент для Etherscan API
type EtherscanClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
	log     *logrus.Entry
}

// NewEtherscanClient - Создает новый клиент Etherscan
func NewEtherscanClient(cfg *config.Config, log *logrus.Entry) *EtherscanClient {
	baseURL := cfg.Etherscan.URL
	if baseURL == "" {
		baseURL = "https://api.etherscan.io"
	}
	logger := log.WithFields(logrus.Fields{"component": "etherscan"})
	return &EtherscanClient{
		apiKey:  cfg.Etherscan.ApiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: logger,
	}
}

// TokenTransaction - Структура для транзакции с токеном
type TokenTransaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	From              string `json:"from"`
	ContractAddress   string `json:"contractAddress"`
	To                string `json:"to"`
	Value             string `json:"value"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	TransactionIndex  string `json:"transactionIndex"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           string `json:"gasUsed"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	Input             string `json:"input"`
	Confirmations     string `json:"confirmations"`
}

// TokenBalance - Структура для баланса токена
type TokenBalance struct {
	Account         string `json:"account"`
	TokenID         string `json:"tokenID"`
	ContractAddress string `json:"contractAddress"`
	TokenName       string `json:"tokenName"`
	TokenSymbol     string `json:"tokenSymbol"`
	TokenDecimal    string `json:"tokenDecimal"`
	Balance         string `json:"balance"`
}

// GetERC20Tokens - Получает список ERC20 токенов для адреса
func (c *EtherscanClient) GetERC20Tokens(ctx context.Context, address string) ([]*TokenBalance, error) {
	// NOTE: Этот метод использует неправильный endpoint!
	// Для получения токенов на кошельке нужно использовать Moralis API
	c.log.Warnf("Etherscan API does not support getting all ERC20 tokens for address. Use Moralis API instead.")
	return nil, fmt.Errorf("Etherscan API does not support getting all ERC20 tokens for address")
}

// GetETHBalance - Получает баланс ETH для адреса
func (c *EtherscanClient) GetETHBalance(ctx context.Context, address string) (float64, error) {
	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "balance")
	params.Set("address", address)
	params.Set("tag", "latest")
	params.Set("apikey", c.apiKey)

	urlStr := fmt.Sprintf("%s/api?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("Etherscan API request failed: %v", err)
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.log.Errorf("Failed to read response body: %v", err)
		return 0, err
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		c.log.Errorf("Failed to unmarshal response: %v", err)
		return 0, err
	}

	if result.Status != "1" {
		c.log.Errorf("Etherscan API error: %s", result.Message)
		return 0, fmt.Errorf("Etherscan API error: %s", result.Message)
	}

	wei, err := strconv.ParseUint(result.Result, 10, 64)
	if err != nil {
		c.log.Errorf("Failed to parse balance: %v", err)
		return 0, err
	}

	eth := float64(wei) / 1e18
	return eth, nil
}

// GetInternalTransactions - Получает внутренние транзакции для адреса
func (c *EtherscanClient) GetInternalTransactions(ctx context.Context, address string, startBlock, endBlock int) ([]map[string]interface{}, error) {
	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "txlistinternal")
	params.Set("address", address)
	params.Set("startblock", strconv.Itoa(startBlock))
	params.Set("endblock", strconv.Itoa(endBlock))
	params.Set("page", "1")
	params.Set("offset", "100")
	params.Set("sort", "asc")
	params.Set("apikey", c.apiKey)

	urlStr := fmt.Sprintf("%s/api?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("Etherscan API request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.log.Errorf("Failed to read response body: %v", err)
		return nil, err
	}

	var result struct {
		Status  string                   `json:"status"`
		Message string                   `json:"message"`
		Result  []map[string]interface{} `json:"result"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		c.log.Errorf("Failed to unmarshal response: %v", err)
		return nil, err
	}

	if result.Status != "1" {
		c.log.Errorf("Etherscan API error: %s", result.Message)
		return nil, fmt.Errorf("Etherscan API error: %s", result.Message)
	}

	return result.Result, nil
}
