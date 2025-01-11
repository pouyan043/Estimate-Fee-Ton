package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type TransactionResult struct {
	Fee        string `json:"fee"`
	StorageFee string `json:"storage_fee"`
	OtherFee   string `json:"other_fee"`
	InMsg      struct {
		MsgData struct {
			Body string `json:"body"`
		} `json:"msg_data"`
		FwdFee string `json:"fwd_fee"`
	} `json:"in_msg"`
}

type GetTransactionResponse struct {
	Ok     bool                `json:"ok"`
	Result []TransactionResult `json:"result"`
	Error  string              `json:"error"`
}

type EstimateRequestPayload struct {
	Address      string `json:"address"`
	Body         string `json:"body"`
	IgnoreChksig bool   `json:"ignore_chksig"`
}

type Fees struct {
	InFwdFee   int `json:"in_fwd_fee"`
	StorageFee int `json:"storage_fee"`
	GasFee     int `json:"gas_fee"`
	FwdFee     int `json:"fwd_fee"`
}

type EstimateResult struct {
	SourceFees Fees   `json:"source_fees"`
	Extra      string `json:"@extra"`
}

type EstimateResponsePayload struct {
	Ok     bool           `json:"ok"`
	Result EstimateResult `json:"result"`
	Error  string         `json:"error"`
}

// a modifire to handle erors
func handleError(err error, context string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s: %v\n", context, err)
		os.Exit(1)
	}
}

// generateHash generates a SHA256 hash from address
func generateHash(address string) string {
	hash := sha256.Sum256([]byte(address))
	return hex.EncodeToString(hash[:])
}

// generateURL generates the URL for the getTransactions API request.
func generateURL(address string, limit int, hash string, toLt int, archival bool) string {
	baseURL := "https://toncenter.com/api/v2/getTransactions"
	params := url.Values{}
	params.Add("address", address)
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("hash", hash)
	params.Add("to_lt", fmt.Sprintf("%d", toLt))
	params.Add("archival", fmt.Sprintf("%t", archival))

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	return fullURL
}

// callGetTransaction sends a GET request to the getTransactions API and returns body
func callGetTransaction(address string) (string, string, string, string, string, error) {
	hash := generateHash(address)
	url := generateURL(address, 100, hash, 0, true)
	req, err := http.NewRequest("GET", url, nil)
	handleError(err, "creating GET request")

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	handleError(err, "making GET request")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", "", fmt.Errorf("request failed with status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	handleError(err, "reading response body")

	var responsePayload GetTransactionResponse
	err = json.Unmarshal(body, &responsePayload)
	handleError(err, "decoding response payload")

	if !responsePayload.Ok {
		return "", "", "", "", "", fmt.Errorf("error from API: %s", responsePayload.Error)
	}

	if len(responsePayload.Result) == 0 {
		return "", "", "", "", "", fmt.Errorf("no transaction results found")
	}

	// Extract body and fees from transaction in the result
	transaction := responsePayload.Result[0]
	return transaction.InMsg.MsgData.Body, transaction.Fee, transaction.StorageFee, transaction.OtherFee, transaction.InMsg.FwdFee, nil
}

// callEstimateFee sends a POST request to the estimateFee API and returns the estimated fees
func callEstimateFee(walletAddress, body string) (*EstimateResult, error) {
	requestPayload := EstimateRequestPayload{
		Address:      walletAddress,
		Body:         body,
		IgnoreChksig: true,
	}

	requestBody, err := json.Marshal(requestPayload)
	handleError(err, "marshalling request payload")

	url := "https://toncenter.com/api/v2/estimateFee"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	handleError(err, "creating POST request")

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	handleError(err, "making POST request")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %s, response: %s", resp.Status, string(bodyBytes))
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	handleError(err, "reading response body")

	var responsePayload EstimateResponsePayload
	err = json.Unmarshal(bodyBytes, &responsePayload)
	handleError(err, "decoding response payload")

	if !responsePayload.Ok {
		return nil, fmt.Errorf("error from API: %s", responsePayload.Error)
	}

	return &responsePayload.Result, nil
}

// convertNanotonToTon
func convertNanotonToTon(nanoton string) (float64, error) {
	value, err := strconv.ParseFloat(nanoton, 64)
	if err != nil {
		return 0, err
	}
	return value / 1e9, nil
}

func main() {
	walletAddress := "EQCS4UEa5UaJLzOyyKieqQOQ2P9M-7kXpkO5HnP3Bv250cN3" /// replace your wallet address here 

	// Generate hash from wallet address
	hash := generateHash(walletAddress)
	fmt.Printf("Generated Hash: %s\n", hash)

	// Call getTransaction with wallet address
	body, fee, storageFee, otherFee, fwdFee, err := callGetTransaction(walletAddress)
	handleError(err, "calling getTransaction API")

	// Convert fees from nanoton to ton
	feeInTon, err := convertNanotonToTon(fee)
	handleError(err, "converting fee from nanoton to ton")

	storageFeeInTon, err := convertNanotonToTon(storageFee)
	handleError(err, "converting storage fee from nanoton to ton")

	otherFeeInTon, err := convertNanotonToTon(otherFee)
	handleError(err, "converting other fee from nanoton to ton")

	fwdFeeInTon, err := convertNanotonToTon(fwdFee)
	handleError(err, "converting forward fee from nanoton to ton")

	// Print the retrieved body and fees
	fmt.Printf("Retrieved Body: %s\n", body)
	fmt.Printf("Transaction Fee: %s nanoton (%.9f ton)\n", fee, feeInTon)
	fmt.Printf("Storage Fee: %s nanoton (%.9f ton)\n", storageFee, storageFeeInTon)
	fmt.Printf("Other Fee: %s nanoton (%.9f ton)\n", otherFee, otherFeeInTon)
	fmt.Printf("Forward Fee: %s nanoton (%.9f ton)\n", fwdFee, fwdFeeInTon)

	// Call estimateFee with wallet address and body
	estimateResult, err := callEstimateFee(walletAddress, body)
	handleError(err, "calling estimateFee API")

	// Convert estimated fees from nanoton to ton
	inFwdFeeInTon := float64(estimateResult.SourceFees.InFwdFee) / 1e9
	storageFeeEstInTon := float64(estimateResult.SourceFees.StorageFee) / 1e9
	gasFeeInTon := float64(estimateResult.SourceFees.GasFee) / 1e9
	fwdFeeEstInTon := float64(estimateResult.SourceFees.FwdFee) / 1e9

	fmt.Println("\nEstimate Fee Details:")
	fmt.Printf("  Source Fees:\n")
	fmt.Printf("    In Forward Fee: %d nanoton (%.9f ton)\n", estimateResult.SourceFees.InFwdFee, inFwdFeeInTon)
	fmt.Printf("    Storage Fee: %d nanoton (%.9f ton)\n", estimateResult.SourceFees.StorageFee, storageFeeEstInTon)
	fmt.Printf("    Gas Fee: %d nanoton (%.9f ton)\n", estimateResult.SourceFees.GasFee, gasFeeInTon)
	fmt.Printf("    Forward Fee: %d nanoton (%.9f ton)\n", estimateResult.SourceFees.FwdFee, fwdFeeEstInTon)
	fmt.Printf("  Extra: %s\n", estimateResult.Extra)
}
