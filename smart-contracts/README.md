# B-Verify collateral contract

This directory contains the collateral contract for the B_Verify server. It allows clients to challenge the server for not sending proofs for its commitments. If so, the server can respond with the proof (and resolve the dispute) or the client can withdraw the collateral locked in the contract after 24h if no response was sent by the server.

## (Re-)generate and run contract tests

Because the inputs to the contract are very specific to the B-verify code, the contract tests can be re-generated using the [cmd/generator](Generator). Place the resulting file `penalty_generated.js` into the [test](test) directory and run `truffle test` to execute the test scripts. There is a copy of this test included in the github repo if you prefer just running it.