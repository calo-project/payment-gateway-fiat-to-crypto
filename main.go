package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func convertIDRToUSD(amountIDR float64) (float64, error) {
	wiseAPIKey := os.Getenv("WISE_API_KEY")
	url := "https://api.transferwise.com/v1/quotes"

	requestBody, _ := json.Marshal(map[string]interface{}{
		"sourceCurrency": "IDR",
		"targetCurrency": "USD",
		"sourceAmount":   amountIDR,
	})

	req, err := http.NewRequest("POST", url, ioutil.NopCloser(bytes.NewBuffer(requestBody)))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+wiseAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	targetAmount, ok := result["targetAmount"].(float64)
	if !ok {
		return 0, fmt.Errorf("failed to get targetAmount from Wise API response")
	}

	return targetAmount, nil
}

func buyCrypto(usdAmount float64) (string, error) {
	binanceAPIKey := os.Getenv("BINANCE_API_KEY")
	url := "https://api.binance.com/api/v3/order"

	// Fetch the real-time price of Bitcoin
	resp, err := http.Get("https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var priceData map[string]interface{}
	json.Unmarshal(body, &priceData)

	bitcoinPrice, err := strconv.ParseFloat(priceData["price"].(string), 64)
	if err != nil {
		return "", err
	}

	quantity := usdAmount / bitcoinPrice

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-MBX-APIKEY", binanceAPIKey)
	q := req.URL.Query()
	q.Add("symbol", "BTCUSDT")
	q.Add("side", "BUY")
	q.Add("type", "MARKET")
	q.Add("quantity", fmt.Sprintf("%.6f", quantity))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ = ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	orderId, ok := result["orderId"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get orderId from Binance API response")
	}

	return orderId, nil
}

func buyNFTTicket(cryptoAmount float64) (string, error) {
	nftMarketplaceAPIKey := os.Getenv("NFT_MARKETPLACE_API_KEY")
	url := "https://api.nftmarketplace.com/buy_ticket"

	requestBody, _ := json.Marshal(map[string]interface{}{
		"cryptoAmount":   cryptoAmount,
		"cryptoCurrency": "BTC",
		"nftId":          "TICKET_NFT_ID",
	})

	req, err := http.NewRequest("POST", url, ioutil.NopCloser(bytes.NewBuffer(requestBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+nftMarketplaceAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	transactionHash, ok := result["transactionHash"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get transactionHash from NFT marketplace response")
	}

	return transactionHash, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var requestBody map[string]interface{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	json.Unmarshal(body, &requestBody)

	amountIDR, ok := requestBody["amountIDR"].(float64)
	if !ok {
		http.Error(w, "Invalid amountIDR", http.StatusBadRequest)
		return
	}

	usdAmount, err := convertIDRToUSD(amountIDR)
	if err != nil {
		http.Error(w, "Error converting IDR to USD", http.StatusInternalServerError)
		return
	}
	log.Printf("USD Amount: %.2f", usdAmount)

	orderId, err := buyCrypto(usdAmount)
	if err != nil {
		http.Error(w, "Error buying crypto on Binance", http.StatusInternalServerError)
		return
	}
	log.Printf("Binance Order ID: %s", orderId)

	transactionHash, err := buyNFTTicket(usdAmount)
	if err != nil {
		http.Error(w, "Error purchasing NFT ticket", http.StatusInternalServerError)
		return
	}
	log.Printf("NFT Purchase Transaction Hash: %s", transactionHash)

	responseBody, _ := json.Marshal(map[string]interface{}{
		"success":         true,
		"transactionHash": transactionHash,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

func main() {
	loadEnv()

	http.HandleFunc("/purchase-nft-ticket", handler)
	log.Printf("Server running on port 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
