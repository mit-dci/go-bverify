
const BVerifyPenalty = artifacts.require("BVerifyPenalty")
const BN = require('bn.js');
const Web3 = require("web3");
const exceptions = require("./exceptions");

const sleep = function(ms) {
    return new Promise(resolve => setTimeout(resolve,ms));
}

    contract('BVerifyPenalty', function(accounts) {
        var params = {}
        var coin = new BN("1000000000000000000");

        before(async () => {
            params.web3 = new Web3(web3.currentProvider);
            params.accounts = accounts;
            params.penaltyContract = await BVerifyPenalty.new();

            // Send 10 ETH into the contract as collateral
            await params.penaltyContract.send(new BN(10).mul(coin).toString(), {from: accounts[0]});

       });

        // This test checks if the penalty contract received the coins as distributed in the before()
        it("should have distributed ETH to the penalty contract on startup", async function() {
            var balance = await web3.eth.getBalance(params.penaltyContract.address)
            var expectedBalance = new BN(10).mul(coin).toString();

            assert(balance.toString() == expectedBalance, "Penalty contract did not have the right amount of coins (" + balance.toString() + " vs " + expectedBalance + ")");
        });

        it("should accept a valid challenge", async function() {
            await params.penaltyContract.challengeLackOfProof.sendTransaction(
                "0x22d1019dac766d785de93e537f312370f2a96279eeb6f3597e3f51c9cedd4942467da5e30fdbbe668ef6007f2ec68cfe4ed3b4d2c8566e4cd1963db65098ab15032e29081106fa7838991c61b33b282213c0a60215ba8c3f84572b260f15766c012094a61d77bd5ef28d3e126bf5a2d27f1bc8e8972dfb73b73dd2ef55c11418d726",
                "0x04696380f1941bef7db2deac499384ff9ae1da8e172ad2c4e65cd213372bdbb4ff4bedb0f60545ccd88a787a93c4015b52b21e143660c6d76cd7fe70bd70dc167e",
                28,
                "0x22d1019dac766d785de93e537f312370f2a96279eeb6f3597e3f51c9cedd4942",
                "0x467da5e30fdbbe668ef6007f2ec68cfe4ed3b4d2c8566e4cd1963db65098ab15",
                
                
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should accept a valid challenge with a wrong logID", async function() {
            await params.penaltyContract.challengeLackOfProof.sendTransaction(
                "0xae730553e3c95064147b4b2db38d270d58dc1d2fa0637afe1cec6ea7bd4eae564704ae093c94be7083c2a72e0f5fddb86a47cc00ff4a77a84561e557e1836ef3032e29081106fa7838991c61b33b282213c0a60215ba8c3f84572b260f15766c0120fe8e35559687763220f9fd32fa7fbd642a0e62eaf282938f7f37d33d94a1c5c5",
                "0x0474ff618feb17f3b7ddcb9d78aabcaad8182c6b5f7bb85e65b5215d09156e953c292a9c33b57e80701ae1ed0e2d4fd5e12ba23dca969d48c0a4ff947c3c471cc6",
                28,
                "0xae730553e3c95064147b4b2db38d270d58dc1d2fa0637afe1cec6ea7bd4eae56",
                "0x4704ae093c94be7083c2a72e0f5fddb86a47cc00ff4a77a84561e557e1836ef3",

                
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should disallow withdrawing collateral within challenge period", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0xe1c328291ab0f126d9fd64866fedaadb8b2c06c375d549caee7fd59db42a910a",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        it("should return false for challengeResponded before response", async function() {
            var result = await params.penaltyContract.challengeResponded.call("0xe1c328291ab0f126d9fd64866fedaadb8b2c06c375d549caee7fd59db42a910a");
            assert.isFalse(result);
        });

        it("should not accept a invalid wrong key response", async function() {
            await exceptions.catchRevert(params.penaltyContract.respondLackOfProofWithWrongKey.sendTransaction(
                "0xe1c328291ab0f126d9fd64866fedaadb8b2c06c375d549caee7fd59db42a910a",
                "0x02696380f1941bef7db2deac499384ff9ae1da8e172ad2c4e65cd213372bdbb4ff20986ee148d0906f9335f2fe790154e7b260fdaed34fe1e3916f6d26f93a95a0f1",
            { from: params.accounts[1], gasLimit: 10000000 }));
        });

        it("should accept a valid wrong key response", async function() {
            await params.penaltyContract.respondLackOfProofWithWrongKey.sendTransaction(
                "0xfc60cc22c129526bff55c21a10bd7f16972c66a38593595536880d534f04c709",
                "0x02696380f1941bef7db2deac499384ff9ae1da8e172ad2c4e65cd213372bdbb4ff20986ee148d0906f9335f2fe790154e7b260fdaed34fe1e3916f6d26f93a95a0f1",
                { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should accept a valid proof response", async function() {
           params.penaltyContract.respondLackOfProofWithProof.sendTransaction(
                "0xe1c328291ab0f126d9fd64866fedaadb8b2c06c375d549caee7fd59db42a910a",
                "0x",
                "0x",
            { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should return true for challengeResponded after response", async function() {
            var result = await params.penaltyContract.challengeResponded.call("0xe1c328291ab0f126d9fd64866fedaadb8b2c06c375d549caee7fd59db42a910a");
            assert.isTrue(result);
        });

        it("should disallow withdrawing collateral when marked resolved", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0xe1c328291ab0f126d9fd64866fedaadb8b2c06c375d549caee7fd59db42a910a",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        

    });

    