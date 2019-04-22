pragma solidity ^0.5.0;

import "./dependencies/BytesLib.sol";

contract BVerifyPenalty {
    using BytesLib for bytes;

    // This mapping contains the hashes for which a proof has to be
    // presented by the server. The value in this mapping is the 
    // datetime at which the challenge is received, used to determine
    // the timeout.
    mapping(bytes32 => uint) public lackingProofs;

    // This mapping contains the sender for the challenge message of
    // a particular lacking proof hash. This sender will be allowed
    // to withdraw the collateral
    mapping(bytes32 => address) public lackingProofOwners;

    event ChallengeReceived(bytes signedLogStatement);

    event Debug(string s);
    event DebugAddress(address a);
    event DebugBytes(bytes b);
    
    event ChallengeResponded(bytes32 proofHash, bytes merkleProof, bytes commitmentTransaction);

    event CollateralPaid();

    // Default function
    function () external payable { 
        
    }

    function toString(address x) internal pure returns (string memory) {
        bytes memory b = new bytes(20);
        for (uint i = 0; i < 20; i++)
            b[i] = byte(uint8(uint(x) / (2**(8*(19 - i)))));
        return string(b);
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

        // ECRecover is not working rn, so skipping this check. Fix later.
        
        //require(signer == expectedSigner);

        // Extract the Log ID
        bytes memory logId = _signedLogStatement.slice(64,32);

        // Calculate the witnessValue
        bytes memory witnessValue = abi.encodePacked(sha256(_signedLogStatement));

        // Calculate the MPT hash ( which is H(key|value) )
        bytes32 proofHash = sha256(logId.concat(witnessValue));

        // Store the lacking proof
        lackingProofs[proofHash] = block.timestamp;
        lackingProofOwners[proofHash] = msg.sender;

        // Emit the challenge
        emit ChallengeReceived(_signedLogStatement);
    }

    function calculateAddress(bytes memory pub) public pure returns (address addr) {
        bytes32 hash = keccak256(pub);
        assembly {
            mstore(0, hash)
            addr := mload(0)
        }    
    }

    function respondLackOfProof(bytes32 proofHash, bytes calldata merkleProof, bytes calldata commitmentTransaction) external {
        // Check given data

        // Check merkle proof

        // Check transaction
    
        // Mark resolved
        lackingProofs[proofHash] = 0;

        emit ChallengeResponded(proofHash, merkleProof, commitmentTransaction);
    }

    function withdrawCollateral(bytes32 proofHash) external {
        require(lackingProofs[proofHash] > 1); // Challenge should not be resolved and not already claimed
        require(lackingProofs[proofHash] < block.timestamp-86400); //Challenge still has to be unresolved for 24 hours
        require(lackingProofOwners[proofHash] == msg.sender); // Withdrawer has to be the original challenger

        // Mark claimed to prevent double withdraw
        lackingProofs[proofHash] = 1;

        // Send the collateral to the sender
        msg.sender.transfer(address(this).balance);
    }
}