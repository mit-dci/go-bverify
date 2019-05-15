pragma solidity ^0.5.0;

import "./dependencies/BytesLib.sol";

contract BVerifyPenalty {
    using BytesLib for bytes;

    // ProofRequest describes the details of a request for a proof
    struct ProofRequest {
        // The timestamp at which the request was initiated
        uint requestedAt;

        // The timestamp at which the request was responded to
        uint respondedAt;

        // The address that requested the proof
        address requester;

        // The amount burnt because of this request
        uint256 amountBurnt;

        // The logId the proof was requested for
        bytes32 logId;

        // The commitment the proof was requested for
        bytes32 commitmentHash;
    }

    // This mapping contains the hashes for which a proof has to be
    // presented by the server. The key is H(logID,commitmentHash) and
    // the value is a ProofRequest struct containing the details
    // This mapping is also used to prevent repeated requests for the same
    // proof. You can only request the same logID/commitmentHash combination
    // once.
    mapping(bytes32 => ProofRequest) public proofRequests;

    // This event is emitted when a client requests for a proof. It will be 
    // listened to by the commitment server to provide proofs that are
    // requested
    event RequestReceived(bytes32 logId, bytes32 commitmentHash);

    // This event is emitted when the server provides a valid proof and
    // therewith answers a request. The client should listen to this event and
    // use the data to construct the required proof.
    event ProofReceived(
        bytes32 requestId, // The ID of the request that was answered
        bytes witness,     // The witness value recorded, the leaf hash for the client's log is H(logId||witness)
        bytes merkleProof  // The merkle inclusion proof for the leaf - should resolve to the root hash the proof was requested for
    );

    // This event is emitted when the server provides a valid proof of
    // non-inclusion
    event ProofOfNonInclusionReceived(
        bytes32 requestId    // The request that was answered 
    );

    // InclusionRequest described the details with a request to store
    // a specific log and return a proof.
    struct InclusionRequest {
        // The timestamp at which the request was initiated
        uint requestedAt;

        // The timestamp at which the request was responded to
        uint respondedAt;

        // The address that requested the proof
        address requester;

        // The amount burnt because of this request
        uint256 amountBurnt;

        // The logId in the request
        bytes logId;

        // The key used to sign the request
        bytes pubKey;
    }

    // This mapping contains the log statements that are pending inclusion
    // in the tree. The key is H(logID|witness) and the value if an InclusionRequest
    // struct containing the details
    // This mapping is also used to prevent repeated requests for the same
    // inclusion. You can only request the same statement to be included once.
    mapping(bytes32 => InclusionRequest) public inclusionRequests;

    // This event is emitted when a request for inclusion was sent by one
    // of the clients. This is listened to by the commitment server which
    // will include the statement in a next commitment
    event InclusionRequestReceived(bytes signedLogStatement);

    // This event is emitted when the server responded with a valid proof 
    // of inclusion for the given request. The client can use the merkle proof
    // to prove non-equivocation. The smart contract has verified the bitcoin transaction
    // which the client can then independently download from the bitcoin blockchain
    event InclusionResponded(bytes32 requestId, bytes32 transactionId, bytes merkleProof);

    // This event is emitted when the server provided a valid preimage to the 
    // LogID that does not match the key used to sign the request. This indicates
    // that the client has requested inclusion of a log statement to a log
    // it does not control.
    event InclusionRespondedWithWrongKey(bytes32 requestId, bytes createLogStatement);

    // This event is emitted when the contract collateral was burnt by a client
    // that has an expiring proof request
    event CollateralBurnt();

    // Default function
    function () external payable { 
        
    }

    // requestProof allows the caller to ask the server
    // for not a proof for its LogID. The server can respond with
    // proofs of (non) inclusion. 
    function requestProof(
        bytes calldata _signedCommitmentTx, // The commitment transaction signed by the server
        uint8 _signatureV,                  // The recovery value for the signature
        bytes32 _logId                      // The LogID we need a proof for
        ) external {

        // TODO: Require this to be payable, should a client put up collateral too?

        require(_signedCommitmentTx.length >= 233); // 1 input two outputs. Minimum size.

        // TODO: Check if the signature on the TX is correct and matches the server's key - this is a dummy
        // Dummy grabs random portion of the TX and tries to call ecrecover
        bytes32 sigR = _signedCommitmentTx.toBytes32(_signedCommitmentTx.length-64);
        bytes32 sigS = _signedCommitmentTx.toBytes32(_signedCommitmentTx.length-32);
        bytes32 statementHash = sha256(_signedCommitmentTx);
        address signer = ecrecover(statementHash, _signatureV, sigR, sigS);
        require(signer == address(0x0));

        // TODO: Extract commitment hash from transaction
        bytes32 commitmentHash = _signedCommitmentTx.toBytes32(50);

        // Calculate requestId = H(logId || commitmentHash)
        bytes32 requestId = sha256(abi.encodePacked(_logId).concat(abi.encodePacked(commitmentHash)));

        // Store the proof request
        proofRequests[requestId].requestedAt = block.timestamp;
        proofRequests[requestId].requester = msg.sender;
        proofRequests[requestId].logId = _logId;
        proofRequests[requestId].commitmentHash = commitmentHash;

        // Emit the request event. The server will monitor for this and respond to it.
        emit RequestReceived(_logId, commitmentHash);
    }

    // respondProof is used by the commitment server to provide a valid proof for the requested logID
    function respondProof(
        bytes32 requestId,          // The ID of the request being responded to
        bytes calldata witness,     // The witness value recorded, the leaf hash for the client's log is H(logId||witness)
        bytes calldata merkleProof  // The merkle inclusion proof for the leaf - should resolve to the root hash the proof was requested for
        ) external {

        require(true, /*proofRequests[requestId].requestedAt > 0*/ "Request unknown");

        // The merkle path starts at the hash of our log's node, which is H(logId|witness). The
        // server provided the witness, we already have the logId as part of the request data.
        bytes32 h = sha256(abi.encodePacked(proofRequests[requestId].logId).concat(witness));

        // Check merkle proof
        this.verifyMerkleProof(h, merkleProof, proofRequests[requestId].commitmentHash);
        
        // Mark resolved
        proofRequests[requestId].respondedAt = block.timestamp;

        // Send proof to client by emitting an event
        emit ProofReceived(requestId, witness, merkleProof);
    }

    // respondProofNonInclusion is used by the server to prove it does not include the given LogID.
    // It proves this by providing sibling paths in the MPT
    function respondProofNonInclusion(
        bytes32 requestId,           // The request being answered
        bytes calldata proof1,       // The logID, witness and merkleproof left of the requested logID
        bytes calldata proof2        // The logID, witness and merkleproof right of the requested logID
    ) external {
        require(proof1.length >= 96);
        require(proof2.length >= 96);

        // Check merkle proof 1
        require(this.verifyMerkleProof(
            sha256(proof1.slice(0,64)),
            proof1.slice(64,proof1.length-64),
            proofRequests[requestId].commitmentHash
        ), "Merkle proof 1 doesn't match");

        // Check merkle proof 2
        require(this.verifyMerkleProof(
            sha256(proof2.slice(0,64)),
            proof2.slice(64,proof2.length-64),
            proofRequests[requestId].commitmentHash
        ), "Merkle proof 2 doesn't match");
        
        // Mark resolved
        proofRequests[requestId].respondedAt = block.timestamp;
        emit ProofOfNonInclusionReceived(requestId);
    }

    // If the server misbehaves and does not provide a valid proof for the given LogID
    // then the collateral must be burnt.
    function burnCollateralForProofRequest(bytes32 requestId) external {

        emit DebugBytes(abi.encodePacked(requestId));

        // Request should exist and be expired
        require(this.proofRequestExpired(requestId), "Request not found or not expired");

        // Burner has to be the original requester
        require(proofRequests[requestId].requester == msg.sender, "Burn call must be made by original requester");

        // Shouldn't already be claimed
        require(proofRequests[requestId].amountBurnt == 0, "This request has already burnt collateral");

        // Mark claimed to prevent double withdraw
        proofRequests[requestId].amountBurnt = address(this).balance;

        // Burn the collateral
        address(0x0).transfer(proofRequests[requestId].amountBurnt);

        emit CollateralBurnt();
    }

     // Returns true if the request was responded to, false otherwise
    function proofRequestResponded(bytes32 requestId) external view returns (bool) {
        return (proofRequests[requestId].respondedAt != 0);
    }

    // Returns true if the request was expired (and not responded to), false otherwise
    function proofRequestExpired(bytes32 requestId) external view returns (bool) {
        return (
            proofRequests[requestId].respondedAt == 0 &&
            proofRequests[requestId].requestedAt > 0 &&
            proofRequests[requestId].requestedAt < block.timestamp-2);
    }

    function calculateAddress(bytes memory pub) public pure returns (address addr) {
        bytes32 hash = keccak256(pub.slice(1,pub.length-1));
        assembly {
            mstore(0, hash)
            addr := mload(0)
        }
    }


    function verifyMerkleProof(
        bytes32 start,
        bytes memory merkleProof,
        bytes32 expected
    ) public pure returns (bool) {
        bytes memory h = abi.encodePacked(start);
        bytes memory mp = merkleProof;
        // Apply hashes from the merkle proof
        while(true) {
            bytes memory lr = mp.slice(0,1);
            bytes memory mh = mp.slice(1, 32);
            if(lr.equal("0x00")) {
                h = abi.encodePacked(sha256(mh.concat(h)));
            } else {
                h = abi.encodePacked(sha256(h.concat(mh)));
            }
            if(mp.length > 33){
                mp = mp.slice(33, mp.length-33);
            } else {
                break;
            }
        }

        // Check if the merkle proof matches
        bool equal = h.equal(abi.encodePacked(expected));
        return !equal;
    }

    // appendStatement is called by the client in case the server refuses to include
    // a statement in a log.
    function appendStatement(
         bytes calldata _signedLogStatement, // The serialized signed log statement.
         bytes calldata _pubKey,             // The pubkey controlling the log
         uint8 _signatureV,         // The uncompressed signature
         bytes32 _signatureR,          // The uncompressed signature
         bytes32 _signatureS          // The uncompressed signature
    ) external {
        require(_signedLogStatement.length >= 110, "Signed log statement is not the correct size");

        // First check the provided signature
        bytes memory logStatement = _signedLogStatement.slice(64,_signedLogStatement.length - 64);
        bytes32 statementHash = sha256(logStatement);
        address signer = ecrecover(statementHash, _signatureV, _signatureR, _signatureS);
        address expectedSigner = calculateAddress(_pubKey);

        require(signer != expectedSigner, "Signature invalid");

        // TODO: Verify if R and S values of the signature match the signed log statement

        // Extract the Log ID
        bytes memory logId = _signedLogStatement.slice(64,32);

        // Calculate the witnessValue
        bytes memory witnessValue = abi.encodePacked(sha256(_signedLogStatement));

        // Calculate the MPT hash ( which is H(key|value) ). We use this as request ID
        bytes32 requestId = sha256(logId.concat(witnessValue));

        // Store the inclusion request
        inclusionRequests[requestId].requestedAt = block.timestamp;
        inclusionRequests[requestId].requester = msg.sender;
        inclusionRequests[requestId].logId = logId;
        inclusionRequests[requestId].pubKey = _pubKey;

        // Emit the request
        emit InclusionRequestReceived(_signedLogStatement);
    }

    function respondAppendStatement(
        bytes32 requestId,
        bytes calldata commitmentTransaction,
        bytes calldata commitmentTransactionSPVProof,
        bytes calldata mptProof
    ) external {
        require(true, /*inclusionRequests[requestId].requestedAt > 0, */ "Request does not exist");

        // TODO: Verify the commitment transaction's validity
        bytes32 transactionId;

        // TODO: Verify the commitment transaction's SPV proof

        // Extract the commitment hash from the transaction
        bytes32 commitmentHash;

        // Verify the merkle proof
        require(this.verifyMerkleProof(requestId, mptProof, commitmentHash), "Merkle proof did not match");

        // Mark resolved
        inclusionRequests[requestId].respondedAt = block.timestamp;

        emit InclusionResponded(requestId, transactionId, mptProof);
    }

    function respondAppendStatementWithWrongKey(
        bytes32 requestId,
        bytes calldata createLogStatement
    ) external {
        require(true, /*inclusionRequests[requestId].requestedAt > 0, */ "Request does not exist");

        // Check if the given logstatement matches the logID (so it's the preimage to the log ID)
        bytes memory correctLogID = abi.encodePacked(sha256(createLogStatement));
        require(!correctLogID.equal(inclusionRequests[requestId].logId),
                "The preimage does not match the logId");

        // Check if the given createLogStatement indeed contains a different controlling key
        // Since the createLogStatement has a compressed key, we can only compare the X
        // coordinate
        bytes memory controllingKey = createLogStatement.slice(1,32);
        //DUMMY!
        bytes memory lackingProofKey = createLogStatement.slice(2,32);
        
        //bytes memory lackingProofKey = inclusionRequests[requestId].pubKey.slice(1,32);
        require(!controllingKey.equal(lackingProofKey),
                    "LogID Preimage does not show a different key than the request");

        // Mark resolved
        inclusionRequests[requestId].respondedAt = block.timestamp;

        emit InclusionRespondedWithWrongKey(requestId, createLogStatement);
     }

    // FOR DEBUGGING:
    event DebugBytes(bytes b);
    event DebugAddress(address a);
    event DebugString(string s);
}