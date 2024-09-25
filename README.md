# Tss Lib Cli

This project is a CLI tool to demonstrate Binance's [tss-lib](https://github.com/bnb-chain/tss-lib).

This library is an implementation of ECDSA threshold signature. The goal is to sign a message by multiple participants, who all have a piece of the private key, but are not able to access the complete private key.

This tool simulates this kind of process on the local machine. However the goal of threshold signature is to split the process across multiple distinct devices.

This Go program simulate participants on a single local machine.

Participants can generate keys and sign a message. We can then verify the signature.

The key generation process will create `keygen-x.json files`. Those files include the private key piece and the public key.

The signing process will create `sig-x.json` files that will have the same signature

# Dev

## Setup

    task install

## Run

    task run -- generate 4 2
    task run -- sign 4 2 hello
    task run -- verify 4 2 hello

# Build

    task build

# Run

    ./bin/tss-lib-cli generate 4 2

# Format

    task fmt

# Assumptions

We assume that all participants are involved in the signature.

The tss-lib...
- calculate the ECDSA public key, so we don't have to do it again, it's available in the data given at the end of the generation process
