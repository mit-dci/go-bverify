
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
                "%sal1%",
                "%pub1%",
                %v1%,
                "%r1%",
                "%s1%",
                
                
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should accept a valid challenge with a wrong logID", async function() {
             await params.penaltyContract.challengeLackOfProof.sendTransaction(
                "%sal2%",
                "%pub2%",
                %v2%,
                "%r2%",
                "%s2%",
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should accept a valid 2nd challenge", async function() {
            await params.penaltyContract.challengeLackOfProof.sendTransaction(
                "%sal3%",
                "%pub1%",
                %v3%,
                "%r3%",
                "%s3%",
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should disallow withdrawing collateral within challenge period", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "%ph1%",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        it("should return false for challengeResponded before response", async function() {
            var result = await params.penaltyContract.challengeResponded.call("%ph1%");
            assert.isFalse(result);
        });

        it("should not accept a invalid wrong key response", async function() {
            await exceptions.catchRevert(params.penaltyContract.respondLackOfProofWithWrongKey.sendTransaction(
                "%ph1%",
                "%cls%",
            { from: params.accounts[1], gasLimit: 10000000 }));
        });

        it("should accept a valid wrong key response", async function() {
            await params.penaltyContract.respondLackOfProofWithWrongKey.sendTransaction(
                "%ph2%",
                "%cls%",
                { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should accept a valid proof response", async function() {
           params.penaltyContract.respondLackOfProofWithProof.sendTransaction(
                "%ph1%",
                "0x",
                "0x",
            { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should return true for challengeResponded after response", async function() {
            var result = await params.penaltyContract.challengeResponded.call("%ph1%");
            assert.isTrue(result);
        });

        it("should disallow withdrawing collateral when marked resolved", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "%ph1%",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

 		it("should allow withdrawing collateral when challenge period ends", async function() {
            // Wait for expiry
            while(!(await params.penaltyContract.challengeExpired.call("%ph3%"))) {
                await sleep(500);
            }
        
            var balBefore = await web3.eth.getBalance(params.accounts[0])
            await params.penaltyContract.withdrawCollateral.sendTransaction(
                "%ph3%",
            { from: params.accounts[0], gasLimit: 10000000 });
            var balAfter = await web3.eth.getBalance(params.accounts[0])

            // Difference should be 10 ETH -/- gas cost for sending
            expectedBalanceDiff = "9998839720000000000";
            realBalanceDiff  = new BN(balAfter).sub(new BN(balBefore)).toString();
            assert(expectedBalanceDiff === realBalanceDiff, "Balance increase did not match collateral " + expectedBalanceDiff + " vs " + realBalanceDiff);
        });
    });