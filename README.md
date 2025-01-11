# Transaction and Fee Estimation Tool

This tool is designed to fetch transaction details and estimate transaction fees using APIs from toncenter.com.

## Installation

To install this tool, follow these steps:

1. Ensure you have Golang installed on your system. If not, you can download it from [here](https://golang.org/dl/).
2. Clone the repository:
    ```sh
    git clone https://github.com/pouyan043/Estimate-Fee-Ton.git
    ```
3. Navigate to the project directory:
    ```sh
    cd transaction-fee-estimation
    ```
4. Install dependencies:
    ```sh
    go mod tidy
    ```

## Usage

### Step 1: Generate Hash and Send GET Request to Fetch Transactions

In this step, we generate a SHA256 hash from the wallet address and then send a GET request with address and hash inputs .

then it returns body 

## what's body??
GetTransaction returns body
so we make body from informations in transaction 

### step 2 : now we Generated body so we send a new GET Request to GetEstimate api 

 we use body and address in our GET Request inputs and it returns estimated fee 

 ## step 3 : go run main.go
