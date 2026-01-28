package util

import "strings"

// trustedContracts хранит адреса в нижнем регистре
// Использование map[string]struct{} занимает 0 байт памяти на значение
var trustedContracts = map[string]struct{}{
	// --- Stablecoins ---
	"0xdac17f958d2ee523a2206206994597c13d831ec7": {}, // USDT
	"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": {}, // USDC
	"0x6b175474e89094c44da98b954eedeac495271d0f": {}, // DAI
	"0x4fabb145d64652a948d72533023f6e7a623c7c53": {}, // BUSD (Ethereum)

	"0x1456688345527bE1f37E9e627DA0837D6f08C925": {}, // GUSD
	"0x57Ab1ec28D129707052df4dF418D58a2D46d5f51": {}, // sUSD
	"0x0000000000085d4780B73119b644AE5ecd22b376": {}, // TUSD

	// --- Infrastructure ---
	"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {}, // WETH

	// --- DEX Routers (Spenders) ---
	"0x7a250d5630b4cf539739df2c5dacb4c659f2488d": {}, // Uniswap V2
	"0xe592427a0aece92de3edee1f18e0157c05861564": {}, // Uniswap V3
	"0x3fc91a3afd70395cd496c647d5a6cc9d4b2b7fad": {}, // Uniswap Universal Router
	"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45": {}, // Uniswap V3 Router 2
	"0x1111111254fb6c44bac0bed2854e76f90643097d": {}, // 1inch V5
	"0xdef1c0ded9bec7f1a1670819833240f027b25eff": {}, // 0x Exchange Proxy

	// --- NFT Marketplaces ---
	"0x00000000000000adc04c56bf30ac9d3c0aaf14dc": {}, // Seaport 1.5 (OpenSea)
	"0x00000000006c3852cbef3e08e8df289169ede581": {}, // Seaport 1.1
}

// IsTrusted проверяет, входит ли адрес в белый список
func IsTrusted(address string) bool {
	// Обязательно приводим к нижнему регистру перед поиском
	lowerAddr := strings.ToLower(address)
	_, exists := trustedContracts[lowerAddr]
	return exists
}
