# Tss Lib Cli

This project is a CLI tool to demonstrate Binance's [tss-lib](https://github.com/bnb-chain/tss-lib).

# Dev

## Setup

    task install

## Run

    task run -- generate 3 2
    task run -- sign 3 2 hello
    task run -- verify 3 2 hello

# Build

    task build

# Run

    ./bin/tss-lib-cli generate 4

# Format

    task fmt

# Assumptions

We assume that all participants are involved in the signature.

The tss-lib...
- calculate the ECDSA public key, so we don't have to do it again, it's available in the data given at the end of the generation process
- 