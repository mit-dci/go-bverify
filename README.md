# B_Verify

B_Verify is a protocol for scalable, practical non-equivocation using Bitcoin. b verify provides non-equivocation to client applications by witnessing a log of statements to Bitcoin. b verify gaurantees that all clients will see the same sequence of statements. Unlike previous work b verify can scale to provide non-equivocation to millions of application: b verify can witness thousands of new log statements per second at a cost of fractions of a cent per statement, and making equivocation as hard as double spending Bitcoin. b verify accomplishes this by using an untrusted server incentivized by a smart contract to witness many new log statements with a single Bitcoin transaction. Users in b verify maintain small proofs of non-equivocation which require them to download only kilobytes of data per day. We implemented a prototype of b verify in Go and tested its ability to scale. To demonstrate how b verify can be used we built a pollution monitoring application on top of b verify that records readings from sensors. By using b verify we gaurantee the consistency and integrity of the data. This application can run on a mobile phone or as a web application and can scale to millions of sensors.

## Dependencies

In order to run the B_Verify server you will need to run a Bitcoin node accessible over RPC

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

