# Tss Lib Cli

This project is a CLI tool to demonstrate Binance's [tss-lib](https://github.com/bnb-chain/tss-lib).

This library is an implementation of ECDSA threshold signature. The goal is to sign a message by multiple participants, who all have a piece of the private key, but are not able to access the complete private key.

This tool simulates this kind of process on the local machine. However the goal of threshold signature is to split the process across multiple distinct devices.

This Go program simulate participants on a single local machine.

Participants can generate keys and sign a message. We can then verify the signature.

The key generation process will create `keygen-x.json files`. Those files include the private key piece and the public key.

The signing process will create `sig-x.json` files that will have the same signature

# Documentation

    tss-lib-cli generate [n] [t]
    tss-lib-cli sign [n] [t] [message]
    tss-lib-cli verify [message]

- You can invoke the sign command with `t` above the value used to generate, because you can involve more party than the minimum required.
- `t` must be strictly lower than `n`, because we need at least `t+1` signers.
- You can invoke the sign command with `n` lower than the one you used to generate, as long as you respect the previous condition.
- The verify command needs at least one valid signature file to be present, because in the context of distributed signature, each participant creates one signature file.
  - In this demonstration CLI, we end up with `t+1` signature files because all signers run in the same computer.
  - In the same logic, it needs only one key file because each key file contains the common ECDSA public key.

# Dev

## Setup

    task install

## Run

    task run -- generate 4 2
    task run -- sign 4 2 hello
    task run -- verify hello

## Format

    task fmt

# Build

    task build

# Run

    ./bin/tss-lib-cli generate 4 2

# Remarks about tss-lib

The tss-lib...
- calculates the ECDSA public key, so we don't have to do it again, it's available in the data given at the end of the generation process.
- calculates the final signature, ready to be verified, and all participants have it, this is why the CLI creates identical `t+1` signature files.
