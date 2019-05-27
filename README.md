# B_Verify

__Warning:__ this is pre-release software. Use on testnet only.

B_Verify is a protocol to practically and scalably prove non-equivocation using Bitcoin. B_Verify witnesses logs of statements to Bitcoin, and, under the assumption that no one can double spend transaction outputs, B_Verify guarantees that all clients will see the same sequence of statements. B_Verify can scale to prevent equivocation for millions of applications: B_Verify can witness thousands of new log statements per second at a cost of fractions of a cent per statement. B_Verify accomplishes this by using an untrusted server incentivized by a smart contract to witness many new log statements with a single Bitcoin transaction. Users in B_Verify maintain small proofs of non-equivocation which require them to download only kilobytes of data per day. We implemented this prototype of B_Verify in Go.

## Dependencies

In order to run the B_Verify server you will need to run a Bitcoin (testnet) node accessible over RPC

## Build

You can grab the binary releases from this Github, or build the binaries yourself. In order to build them yourself, do the following:

```bash
go get github.com/mit-dci/go-bverify/...
cd $GOROOT/src/github.com/mit-dci/go-bverify/cmd/server
go build
```

## Running

You can run the server by executing the `server` executable. Make sure the following environment variables are properly configured:

| Variable             | Description                                                      | Default           |
|----------------------|------------------------------------------------------------------|-------------------|
| `BITCOINRPC`         | The server and port on which the Bitcoin RPC server is listening | `localhost:18443` |
| `BITCOINRPCUSER`     | The username for the Bitcoin RPC server                          | `bverify`         |
| `BITCOINRPCPASSWORD` | The password for the Bitcoin RPC server                          | `bverify`         |


## Code structure

The code in this project is structed as follows:

| Folder                                | Description                                                                       | 
|---------------------------------------|-----------------------------------------------------------------------------------|
| [`bitcoin`](bitcoin/)                 | Various imported libraries for Bitcoin                                            |
| [`client`](client/)                   | B_Verify client libraries                                                         | 
| [`client/uspv`](client/uspv)          | Simple Payment Verification library that is used to download headers from Bitcoin | 
| [`cmd/bench`](cmd/bench)              | Command-line utility for running the benchmarks included in the B_Verify paper    |
| [`cmd/server`](cmd/server)            | Entry point for running the B_Verify server                                       |
| [`crypto`](crypto/)                   | Cryptographic functions and imports for witnessing and signature verification     |
| [`logging`](logging/)                 | Very simple logging framework                                                     |
| [`mobile`](mobile/)                   | The entrypoint for the `gomobile` library used in the Android and iOS Client      |
| [`mpt`](mpt/)                         | Implementation of the Merkle Prefix Trie                                          |
| [`server`](server/)                   | Classes containing the B_Verify server logic                                      |
| [`smart-contracts`](smart-contracts/) | (Proof-of-concept) implementation of the Penalty Smart Contract                   |
| [`utils`](utils/)                     | Various utility functions                                                         |
| [`wallet`](wallet/)                   | Class for tracking UTXOs for the server (used to create witness transactions)     |
| [`wire`](wire/)                       | Classes for client/server communication                                           |

## Related repositories

### [BVerify-Mobile](https://github.com/mit-dci/bverify-mobile)

This repository contains the iOS and Android applications that are used for verifying the witnessed statements

### [BVerify-Sensor](https://github.com/mit-dci/bverify-sensor)

This repository contains an integrated example for a sensor reader that commits to a b_verify server

