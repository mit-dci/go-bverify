pragma solidity ^0.5.0;

import "./dependencies/BytesLib.sol";

contract BVerifyPenalty {
    using BytesLib for bytes;

    // LackingProofChallenge describes the details of a challenge
    struct LackingProofChallenge {
        // The timestamp at which the challenge was initiated
        uint challengedAt;

        // The timestamp at which the challenge was responded to
        uint respondedAt;

        // The amount paid out on this challenge
        uint256 claimedAmount;

        // The address that challenged
        address challenger;

        // The pubKey controlling the log as indicated by the challenger
        bytes pubKey;

        // The LogID used in the challenge. This is used to match
        // to a preimage for the create statement. The server uses
        // this to respond to a challenge for a logID with the wrong
        // key
        bytes logID;
    }

    // This mapping contains the hashes for which a proof has to be
    // presented by the server. The value in this mapping is the 
    // LackingProofChallenge struct containing the details of the challenge
    mapping(bytes32 => LackingProofChallenge) public lackingProofs;

    event ChallengeReceived(bytes signedLogStatement);

    event Debug(string s);
    event DebugAddress(address a);
    event DebugBytes(bytes b);
    
    event ChallengeRespondedWithProof(bytes32 proofHash, bytes merkleProof, bytes commitmentTransaction);
    event ChallengeRespondedWithWrongKey(bytes32 proofHash, bytes createLogStatement);

    event CollateralPaid();

    // Default function
    function () external payable { 
        
    }

    // challengeLackOfProof allows the caller to challenge the server
    // for not providing a proof. The server can either immediately respond
    // to the challenge with a proof, or include the statement in the next
    // commitment and respond with that.
    function challengeLackOfProof(
        bytes calldata _signedLogStatement, // The serialized signed log statement.
        bytes calldata _pubKey,             // The pubkey controlling the log
        uint8 _signatureV,         // The uncompressed signature
        bytes32 _signatureR,          // The uncompressed signature
        bytes32 _signatureS          // The uncompressed signature
        ) external {
        
        require(_signedLogStatement.length >= 110);

        // First check the provided signature
        bytes memory logStatement = _signedLogStatement.slice(64,_signedLogStatement.length - 64);
        bytes32 statementHash = sha256(logStatement);
        address signer = ecrecover(statementHash, _signatureV, _signatureR, _signatureS);
        address expectedSigner = calculateAddress(_pubKey);

        require(signer == expectedSigner);

        // Extract the Log ID
        bytes memory logID = _signedLogStatement.slice(64,32);
        emit DebugBytes(abi.encodePacked(logID));

        // Calculate the witnessValue
        bytes memory witnessValue = abi.encodePacked(sha256(_signedLogStatement));

        // Calculate the MPT hash ( which is H(key|value) )
        bytes32 proofHash = sha256(logID.concat(witnessValue));

        // Store the lacking proof
        lackingProofs[proofHash].challengedAt = block.timestamp;
        lackingProofs[proofHash].challenger = msg.sender;
        lackingProofs[proofHash].logID = logID;
        lackingProofs[proofHash].pubKey = _pubKey;

        emit DebugBytes(abi.encodePacked(proofHash));

        // Emit the challenge
        emit ChallengeReceived(_signedLogStatement);
    }

    function calculateAddress(bytes memory pub) public pure returns (address addr) {
        bytes32 hash = keccak256(pub.slice(1,pub.length-1));
        assembly {
            mstore(0, hash)
            addr := mload(0)
        }    
    }

    function challengeResponded(bytes32 proofHash) external view returns (bool) {
        return (lackingProofs[proofHash].respondedAt != 0);
    }

    function respondLackOfProofWithProof(bytes32 proofHash, bytes calldata merkleProof, bytes calldata commitmentTransaction) external {
        // Check given data

        // Check merkle proof

        // Check transaction
    
        // Mark resolved
        lackingProofs[proofHash].respondedAt = block.timestamp;

        emit ChallengeRespondedWithProof(proofHash, merkleProof, commitmentTransaction);
    }

    function respondLackOfProofWithWrongKey(bytes32 proofHash, bytes calldata createLogStatement) external {
        // Check if the given logstatement matches the logID (so it's the preimage to the log ID)
        bytes memory correctLogID = abi.encodePacked(sha256(createLogStatement));
        require(correctLogID.equal(lackingProofs[proofHash].logID));

        // Check if the given createLogStatement indeed contains a different controlling key
        // Since the createLogStatement has a compressed key, we can only compare the X
        // coordinate
        bytes memory controllingKey = createLogStatement.slice(1,32);
        bytes memory lackingProofKey = lackingProofs[proofHash].pubKey.slice(1,32);
        require(!controllingKey.equal(lackingProofKey));

        // Mark resolved
        lackingProofs[proofHash].respondedAt = block.timestamp;

        emit ChallengeRespondedWithWrongKey(proofHash, createLogStatement);
    }
    
    function withdrawCollateral(bytes32 proofHash) external {
        //Challenge still has to be unresolved for 24 hours
        require(lackingProofs[proofHash].challengedAt < block.timestamp-86400); 

        // Should be left unresolved
        require(lackingProofs[proofHash].respondedAt == 0);

        // Withdrawer has to be the original challenger
        require(lackingProofs[proofHash].challenger == msg.sender); 

        // Shouldn't already be claimed
        require(lackingProofs[proofHash].claimedAmount == 0);

        // Mark claimed to prevent double withdraw
        lackingProofs[proofHash].claimedAmount = address(this).balance;

        // Send the collateral to the sender
        msg.sender.transfer(lackingProofs[proofHash].claimedAmount);
    }
}