package utils

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tyler-smith/go-bip39"
	"github.com/xssnick/tonutils-go/address"
	"golang.org/x/crypto/ed25519"
)

func LoadEnvData(publicKey, privateKey, address, mnemonic, seed *string) {
	file, err := os.Open(".env")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "PUBLIC_KEY="):
			*publicKey = strings.TrimPrefix(line, "PUBLIC_KEY=")
		case strings.HasPrefix(line, "PRIVATE_KEY="):
			*privateKey = strings.TrimPrefix(line, "PRIVATE_KEY=")
		case strings.HasPrefix(line, "WALLET_ADDRESS="):
			*address = strings.TrimPrefix(line, "WALLET_ADDRESS=")
		case strings.HasPrefix(line, "MNEMONIC="):
			*mnemonic = strings.TrimPrefix(line, "MNEMONIC=")
		case strings.HasPrefix(line, "SEED="):
			*seed = strings.TrimPrefix(line, "SEED=")
		}
	}
	if *publicKey == "" || *privateKey == "" || *address == "" || *mnemonic == "" || *seed == "" {
		log.Fatal("Failed to load wallet data from .env")
	}
}

func SaveToEnvFile(publicKey, privateKey, address, mnemonic, seed string) {
	file, err := os.Create(".env")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	envContent := fmt.Sprintf("PUBLIC_KEY=%s\nPRIVATE_KEY=%s\nWALLET_ADDRESS=%s\nMNEMONIC=%s\nSEED=%s\n",
		publicKey, privateKey, address, mnemonic, seed)
	_, err = file.WriteString(envContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Keys, address, mnemonic, and seed saved to .env file")
}

func GenerateWalletData() (string, string, string, string, string) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		log.Fatal(err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		log.Fatal(err)
	}

	seed := bip39.NewSeed(mnemonic, "")
	privateKey := ed25519.NewKeyFromSeed(seed[:32])

	publicKey := privateKey.Public().(ed25519.PublicKey)
	address := generateAddressFromPublicKey(publicKey)

	return base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(privateKey), address, mnemonic, string(seed)
}

func generateAddressFromPublicKey(pubKey ed25519.PublicKey) string {
	addr := address.NewAddress(0x1, 0x0, pubKey)
	return addr.String()
}

func DecodeBase64WithPadding(encoded string) (ed25519.PrivateKey, error) {
	padding := 4 - len(encoded)%4
	if padding != 4 {
		encoded += strings.Repeat("=", padding)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("failed to decode private key")
	}
	return decoded[:ed25519.PrivateKeySize], nil
}

func EstimateFee(walletAddress, body, initCode, initData string) (float64, error) {
	requestPayload := map[string]interface{}{
		"address":      walletAddress,
		"body":         body,
		"ignoreChksig": true,
		"initCode":     initCode,
		"initData":     initData,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return 0, fmt.Errorf("error marshaling request: %s", err)
	}

	url := "https://toncenter.com/api/v2/estimateFee"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, fmt.Errorf("error creating request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	var respPayload struct {
		Ok     bool `json:"ok"`
		Result struct {
			Fee string `json:"fee"`
		} `json:"result"`
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get estimate fee: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&respPayload)
	if err != nil {
		return 0, fmt.Errorf("error decoding response: %s", err)
	}

	if !respPayload.Ok {
		return 0, fmt.Errorf("error from API: estimate fee not found")
	}

	fee, err := strconv.ParseFloat(respPayload.Result.Fee, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing fee: %s", err)
	}

	return fee, nil
}
func GenerateURL(walletAddress, body, initCode, initData string) string {

	return fmt.Sprintf("https://toncenter.com/api/v2/estimateFee?address=%s&body=%s&initCode=%s&initData=%s", walletAddress, body, initCode, initData)
}
