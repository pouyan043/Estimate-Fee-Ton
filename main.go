package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// TransactionParams struct to hold the transaction parameters
type TransactionParams struct {
	Address  string `json:"address"`   // Address to send
	Body     string `json:"body"`      // Body of the transaction
	Value    int64  `json:"value"`     // Amount in nanograms (1 TON = 1e9 nanograms)
	GasPrice int64  `json:"gas_price"` // Gas price in nanograms
	GasLimit int64  `json:"gas_limit"` // Gas limit
}

// Fees struct to hold the fees information
type Fees struct {
	InFwdFee   int64 `json:"in_fwd_fee"`  // Inward forward fee
	StorageFee int64 `json:"storage_fee"` // Storage fee
	GasFee     int64 `json:"gas_fee"`     // Gas fee
	FwdFee     int64 `json:"fwd_fee"`     // Forward fee
}

// Balance struct to hold the balance information
type Balance struct {
	Balance string `json:"result"` // Balance as a string
}

// EstimatedFeeResponse struct to hold the response from the API
type EstimatedFeeResponse struct {
	Ok     bool `json:"ok"` // Response status
	Result struct {
		SourceFees Fees `json:"source_fees"` // Source fees
	} `json:"result"`
	Error string `json:"error"` // Error message if any
}

// createHTTPClient creates a custom HTTP client with a timeout
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 30, // Increase the timeout to 30 seconds
	}
}

// sendRequest handles sending the HTTP request and retrying in case of rate limit exceeded
func sendRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	for {
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make the HTTP request: %v", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			// Handle rate limit exceeded
			fmt.Println("Rate limit exceeded, waiting for 60 seconds before retrying...")
			time.Sleep(60 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
		}
		return resp, nil
	}
}

// estimateFee sends a request to estimate fees for a transaction using toncenter.com API
func estimateFee(transactionParams TransactionParams, apiKey string) (Fees, error) {
	// Marshal the transaction parameters to JSON
	jsonData, err := json.Marshal(transactionParams)
	if err != nil {
		return Fees{}, fmt.Errorf("failed to marshal transaction parameters: %v", err)
	}

	// Print the JSON data for debugging
	fmt.Println("JSON Data:", string(jsonData))

	client := createHTTPClient()
	apiURL := "https://toncenter.com/api/v2/estimateFee"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return Fees{}, fmt.Errorf("failed to create the HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := sendRequest(req, client)
	if err != nil {
		return Fees{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Fees{}, fmt.Errorf("failed to read the response body: %v", err)
	}

	// Print the response body for debugging
	fmt.Println("Response Body:", string(body))

	var estimatedFeeResponse EstimatedFeeResponse
	err = json.Unmarshal(body, &estimatedFeeResponse)
	if err != nil {
		return Fees{}, fmt.Errorf("failed to unmarshal the response body: %v", err)
	}

	if !estimatedFeeResponse.Ok {
		return Fees{}, fmt.Errorf("API error: %s", estimatedFeeResponse.Error)
	}

	return estimatedFeeResponse.Result.SourceFees, nil
}

// getBalance gets the balance for a given address using toncenter.com API
func getBalance(address, apiKey string) (string, error) {
	client := createHTTPClient()
	apiURL := fmt.Sprintf("https://toncenter.com/api/v2/getAddressBalance?address=%s", address)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create the HTTP request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := sendRequest(req, client)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read the response body: %v", err)
	}

	// Print the response body for debugging
	fmt.Println("Response Body (Balance):", string(body))

	var balanceResponse Balance
	err = json.Unmarshal(body, &balanceResponse)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal the response body: %v", err)
	}

	return balanceResponse.Balance, nil
}

// tonTransactionParams defines the parameters for a TON transaction
func tonTransactionParams() TransactionParams {
	return TransactionParams{
		Address:  "UQDnGipnUUjCzg3W-TslugOQluo45vC1Iqf3vI9TQxwd4vlg", // Correct TON address
		Body:     "te6ccgEBAQEAAgAAAA==",                             // Body of the transaction in base64 format
		Value:    1000000000,                                         // Amount in nanograms (1 TON = 1e9 nanograms)
		GasPrice: 1000000000,                                         // Gas price in nanograms
		GasLimit: 2000000,                                            // Gas limit
	}
}

// usdtTransactionParams defines the parameters for a USDT transaction
func usdtTransactionParams() TransactionParams {
	body := "te6ccgEBAQEAAgAAAA=="
	decodedBody, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		log.Fatalf("Failed to decode body: %v", err)
	}

	return TransactionParams{
		Address:  "EQCS4UEa5UaJLzOyyKieqQOQ2P9M-7kXpkO5HnP3Bv250cN3", // Correct USDT address
		Body:     base64.StdEncoding.EncodeToString(decodedBody),     // Ensure body is correctly formatted
		Value:    1000000000,                                         // Amount in nanograms (1 TON = 1e9 nanograms)
		GasPrice: 1000000000,                                         // Gas price in nanograms
		GasLimit: 2000000,                                            // Gas limit
	}
}

// printFees prints the estimated fees
func printFees(title string, fees Fees) {
	fmt.Println(title)
	fmt.Printf("Inward Forward Fee: %d nanograms\n", fees.InFwdFee)
	fmt.Printf("Storage Fee: %d nanograms\n", fees.StorageFee)
	fmt.Printf("Gas Fee: %d nanograms\n", fees.GasFee)
	fmt.Printf("Forward Fee: %d nanograms\n", fees.FwdFee)
}

func main() {
	apiKey := "71f38d1c2b38074d85e0bd035d8648bcb7e7be66c81f0756b497e682f29996a8" // Correct API key

	// Estimate fee for TON transaction
	tonFees, err := estimateFee(tonTransactionParams(), apiKey)
	if err != nil {
		log.Fatalf("Failed to estimate fee for TON transaction: %v", err)
	}

	// Estimate fee for USDT transaction
	usdtFees, err := estimateFee(usdtTransactionParams(), apiKey)
	if err != nil {
		log.Fatalf("Failed to estimate fee for USDT transaction: %v", err)
	}

	// Get balance for TON address
	tonBalance, err := getBalance(tonTransactionParams().Address, apiKey)
	if err != nil {
		log.Fatalf("Failed to get balance for TON address: %v", err)
	}

	// Get balance for USDT address
	usdtBalance, err := getBalance(usdtTransactionParams().Address, apiKey)
	if err != nil {
		log.Fatalf("Failed to get balance for USDT address: %v", err)
	}

	// Print the estimated fees for TON transaction
	printFees("TON Transaction Fees:", tonFees)
	fmt.Printf("TON Address Balance: %s nanograms\n", tonBalance)

	// Print the estimated fees for USDT transaction
	printFees("USDT Transaction Fees:", usdtFees)
	fmt.Printf("USDT Address Balance: %s nanograms\n", usdtBalance)
}

