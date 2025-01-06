# Estimate-Fee-Ton

# TON Transaction Fee Estimator

This project is a Go application that estimates the transaction fees for TON and USDT using the `toncenter.com` API.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
  - [createHTTPClient](#createHTTPClient)
  - [estimateFee](#estimateFee)
  - [tonTransactionParams](#tonTransactionParams)
  - [usdtTransactionParams](#usdtTransactionParams)
  - [printFees](#printFees)
- [Contributing](#contributing)
- [License](#license)
- [Documentation](#documentation)

## Prerequisites

1. [Go](https://golang.org/doc/install) version 1.15 or higher
2. A valid API key from [toncenter.com](https://toncenter.com/)

## Installation

To install and run the project, follow these steps:

1. Clone this repository:
    ```sh
    git clone https://github.com/pouyan043/ton-transaction-fee-estimator.git
    ```

2. Navigate to the project directory:
    ```sh
    cd ton-transaction-fee-estimator
    ```

3. Install dependencies:
    ```sh
    go mod tidy
    ```

## Usage

To run the application and estimate transaction fees, use the following command:

```sh
go run main.go
