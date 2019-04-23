
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
                "0x5a1612f1f1d52e7c49d654ee96162f9a1eb400d9bd9af7f9052654b8443c20685147f84443597b3ce73412d7f90c804991e6dc5b88d4b3c0bb83ea32b0c93aa0478b7bda4a06b7b8e2a4fa7566e407e5f0a79abf2bc0f2ad05174bd8484e6a2b012094a61d77bd5ef28d3e126bf5a2d27f1bc8e8972dfb73b73dd2ef55c11418d726",
                "0x04675d86f87459456d00ec06b66ad315f67ada3ed209c2b842c38d85790380b60486baf69d7d845e44b058bb405eac18bce389b1d6e1601c9c47f4e527c5a5dde6",
                28,
                "0x5a1612f1f1d52e7c49d654ee96162f9a1eb400d9bd9af7f9052654b8443c2068",
                "0x5147f84443597b3ce73412d7f90c804991e6dc5b88d4b3c0bb83ea32b0c93aa0",
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should disallow withdrawing collateral within challenge period", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0x478b7bda4a06b7b8e2a4fa7566e407e5f0a79abf2bc0f2ad05174bd8484e6a2b",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        it("should accept a valid response", async function() {
            await params.penaltyContract.respondLackOfProofWithProof.sendTransaction(
                "0x478b7bda4a06b7b8e2a4fa7566e407e5f0a79abf2bc0f2ad05174bd8484e6a2b",
                "0x",
                "0x",
            { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should disallow withdrawing collateral when marked resolved", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0x478b7bda4a06b7b8e2a4fa7566e407e5f0a79abf2bc0f2ad05174bd8484e6a2b",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        

    });

    