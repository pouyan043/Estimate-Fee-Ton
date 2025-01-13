package main

import (
	"bytes"
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
	InitCode     string `json:"init_code"`
	InitData     string `json:"init_data"`
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

// a modifier to handle errors
func handleError(err error, context string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s: %v\n", context, err)
		os.Exit(1)
	}
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
func callGetTransaction(address string, limit int, hash string, toLt int, archival bool) (string, string, string, string, string, error) {
	url := generateURL(address, limit, hash, toLt, archival)
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
func callEstimateFee(walletAddress, body, initCode, initData string) (*EstimateResult, error) {
	requestPayload := EstimateRequestPayload{
		Address:      walletAddress,
		Body:         body,
		IgnoreChksig: true,
		InitCode:     initCode,
		InitData:     initData,
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

// convertNanotonToTon converts nanoton to ton
func convertNanotonToTon(nanoton int) float64 {
	return float64(nanoton) / 1e9
}

func main() {
	fmt.Print("Enter wallet address: ")
	var walletAddress string
	fmt.Scanln(&walletAddress)

	fmt.Print("Enter hash: ")
	var hash string
	fmt.Scanln(&hash)

	fmt.Print("Enter limit: ")
	var limit int
	fmt.Scanln(&limit)

	fmt.Print("Enter to_lt: ")
	var toLt int
	fmt.Scanln(&toLt)

	fmt.Print("Enter archival (true/false): ")
	var archival bool
	fmt.Scanln(&archival)

	fmt.Print("Enter init code: ")
	var initCode string
	fmt.Scanln(&initCode)

	fmt.Print("Enter init data: ")
	var initData string
	fmt.Scanln(&initData)

	// Call getTransaction with wallet address, hash, limit, toLt, and archival
	body, fee, storageFee, otherFee, fwdFee, err := callGetTransaction(walletAddress, limit, hash, toLt, archival)
	handleError(err, "calling getTransaction API")

	// Convert fees from nanoton to ton and aggregate them
	feeNanoton, _ := strconv.Atoi(fee)
	storageFeeNanoton, _ := strconv.Atoi(storageFee)
	otherFeeNanoton, _ := strconv.Atoi(otherFee)
	fwdFeeNanoton, _ := strconv.Atoi(fwdFee)

	totalFeeNanoton := feeNanoton + storageFeeNanoton + otherFeeNanoton + fwdFeeNanoton
	totalFeeTon := convertNanotonToTon(totalFeeNanoton)

	// Print the retrieved body and fees
	fmt.Printf("Retrieved Body: %s\n", body)
	fmt.Printf("Transaction Fee: %d nanoton (%.9f ton)\n", feeNanoton, convertNanotonToTon(feeNanoton))
	fmt.Printf("Storage Fee: %d nanoton (%.9f ton)\n", storageFeeNanoton, convertNanotonToTon(storageFeeNanoton))
	fmt.Printf("Other Fee: %d nanoton (%.9f ton)\n", otherFeeNanoton, convertNanotonToTon(otherFeeNanoton))
	fmt.Printf("Forward Fee: %d nanoton (%.9f ton)\n", fwdFeeNanoton, convertNanotonToTon(fwdFeeNanoton))
	fmt.Printf("Total Transaction Fee: %d nanoton (%.9f ton)\n", totalFeeNanoton, totalFeeTon)

	// Call estimateFee with wallet address, body, init code, and init data
	estimateResult, err := callEstimateFee(walletAddress, body, initCode, initData)
	handleError(err, "calling estimateFee API")

	// Convert estimated fees from nanoton to ton and aggregate them
	inFwdFeeNanoton := estimateResult.SourceFees.InFwdFee
	storageFeeEstNanoton := estimateResult.SourceFees.StorageFee
	gasFeeNanoton := estimateResult.SourceFees.GasFee
	fwdFeeEstNanoton := estimateResult.SourceFees.FwdFee

	totalEstimatedFeeNanoton := inFwdFeeNanoton + storageFeeEstNanoton + gasFeeNanoton + fwdFeeEstNanoton
	totalEstimatedFeeTon := convertNanotonToTon(totalEstimatedFeeNanoton)

	fmt.Println("\nEstimate Fee Details:")
	fmt.Printf("  Source Fees:\n")
	fmt.Printf("    In Forward Fee: %d nanoton (%.9f ton)\n", inFwdFeeNanoton, convertNanotonToTon(inFwdFeeNanoton))
	fmt.Printf("    Storage Fee: %d nanoton (%.9f ton)\n", storageFeeEstNanoton, convertNanotonToTon(storageFeeEstNanoton))
	fmt.Printf("    Gas Fee: %d nanoton (%.9f ton)\n", gasFeeNanoton, convertNanotonToTon(gasFeeNanoton))
	fmt.Printf("    Forward Fee: %d nanoton (%.9f ton)\n", fwdFeeEstNanoton, convertNanotonToTon(fwdFeeEstNanoton))
	fmt.Printf("  Total Estimated Fee: %d nanoton (%.9f ton)\n", totalEstimatedFeeNanoton, totalEstimatedFeeTon)
	fmt.Printf("  Extra: %s\n", estimateResult.Extra)
}
