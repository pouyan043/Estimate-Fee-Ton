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

## Description of entries :
1 : GET transaction : addresss - Maximum number of transactions in response - Logical time of transaction to start with must be sent with hash - Hash of transaction to start with, in base64 or hex encoding , must be sent with lt - Logical time of transaction to finish with (to get tx from lt to to_lt) - and a bool

2 : POST estimated fee : "address": "string",
  "body": "string",
  "init_code": "", /// this one is the value that u wanna send
  "init_data": "", /// this is gas limit
  "ignore_chksig": true


## step 1 : go run main.go

### Step 2: Generate Hash and Send GET Request to Fetch Transactions

In this step, we filing inputes and then send a GET request with parameters inputs .

then it returns body 

## what's body??
GetTransaction returns body
so we make body from informations in transaction 

### step 3 : now we Generated body so we send a new POST Request to GetEstimate api 

 ### we filing inputs :
  then it will returns 2 fees:
  -1 : the fees for transaction that u were looking for
  -2 : the estimated fee for future
 
