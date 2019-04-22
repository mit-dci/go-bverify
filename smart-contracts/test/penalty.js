
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
                "0x34272d15b3fbac790ace6efafbbf98ccd0f857ea5ad1723a6f4689c3280c5d010b3f84c102155dda2984b36335d139f11937ce1da420853e516b6ce8aa39560d56c3f434bad6228acbc37457b74417533874a6c063018a349e142b5e8a340a0b012094a61d77bd5ef28d3e126bf5a2d27f1bc8e8972dfb73b73dd2ef55c11418d726",
                "0x04426e6a2b7e6965d22d404bdcfb318717a395626a8062ec8660df000abe4b04a70528349f0f5a1a76b83c1becba8d5bd942764ee9d79a2dd98803730e9a2e4724",
                0,
                "0x34272d15b3fbac790ace6efafbbf98ccd0f857ea5ad1723a6f4689c3280c5d01",
                "0x0b3f84c102155dda2984b36335d139f11937ce1da420853e516b6ce8aa39560d",
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should disallow withdrawing collateral within challenge period", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0x96eb5a4f646dcd53a181ae8be68c2fb53d9dfb84c8528884f76483e86894e2a1",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        it("should accept a valid response", async function() {
            await params.penaltyContract.respondLackOfProof.sendTransaction(
                "0x96eb5a4f646dcd53a181ae8be68c2fb53d9dfb84c8528884f76483e86894e2a1",
                "0x",
                "0x",
            { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should disallow withdrawing collateral when marked resolved", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0x96eb5a4f646dcd53a181ae8be68c2fb53d9dfb84c8528884f76483e86894e2a1",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        

    });

    