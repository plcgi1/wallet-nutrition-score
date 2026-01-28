package util

import "fmt"

func GetAdressURL(address string) string {
	result := fmt.Sprintf("https://etherscan.io/address/%s", address)
	return result
}
