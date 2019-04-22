# B-Verify collateral contract

This directory contains the collateral contract for the B_Verify server. It allows clients to challenge the server for not sending proofs for its commitments. If so, the server can respond with the proof (and resolve the dispute) or the client can withdraw the collateral locked in the contract after 24h if no response was sent by the server.