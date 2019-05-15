
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

        it("test requestProof gas usage", async function() {
            var txReceipt = await params.penaltyContract.requestProof.sendTransaction(
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c466fe01f4d3ccf0f0000000000ffffffff020000000000000000226a20523e59cfc5235b915dc89de188d87449453b083a8b7d97c1ee64d875da4033619892980000000000160014f7ccc2053561be6896b777a5be8d2cbfeef3fea702463043021f0a0a5a61e948429674878bd9fb87906069fb4978c1193cdc533c76c598678202205d9559897795daf66c860785fa292d13af40e1003da478fb6cdd1972983481d80121030e0e6938fc9d9741b948850c39817e69c22171631000967fc8095a9b6b1e830d00000000",
                27,
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",                
            { from: params.accounts[0], gasLimit: 10000000 });
            console.log("Gas used for requestProof: ", txReceipt.receipt.gasUsed);
        });

        it("test respondProof gas usage", async function() {
            var txReceipt = await params.penaltyContract.respondProof.sendTransaction(
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x01460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",                
            { from: params.accounts[0], gasLimit: 10000000 });
            console.log("Gas used for respondProof: ", txReceipt.receipt.gasUsed);
        });

        it("test respondProofNonInclusion gas usage", async function() {
            var txReceipt = await params.penaltyContract.respondProofNonInclusion.sendTransaction(
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
            { from: params.accounts[0], gasLimit: 10000000 });
            console.log("Gas used for respondProofNonInclusion: ", txReceipt.receipt.gasUsed);
        });

        it("test appendStatement gas usage", async function() {
            var txReceipt = await params.penaltyContract.appendStatement.sendTransaction(
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c466fe01f4d3ccf0f0000000000ffffffff020000000000000000226a20523e59cfc5235b915dc89de188d87449453b083a8b7d97c1ee64d875da4033619892980000000000160014f7ccc2053561be6896b777a5be8d2cbfeef3fea702463043021f0a0a5a61e948429674878bd9fb87906069fb4978c1193cdc533c76c598678202205d9559897795daf66c860785fa292d13af40e1003da478fb6cdd1972983481d80121030e0e6938fc9d9741b948850c39817e69c22171631000967fc8095a9b6b1e830d00000000",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                27,
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",                
            { from: params.accounts[0], gasLimit: 10000000 });
            console.log("Gas used for appendStatement: ", txReceipt.receipt.gasUsed);
        });

        it("test respondAppendStatement gas usage", async function() {
            var txReceipt = await params.penaltyContract.respondAppendStatement.sendTransaction(
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c466fe01f4d3ccf0f0000000000ffffffff020000000000000000226a20523e59cfc5235b915dc89de188d87449453b083a8b7d97c1ee64d875da4033619892980000000000160014f7ccc2053561be6896b777a5be8d2cbfeef3fea702463043021f0a0a5a61e948429674878bd9fb87906069fb4978c1193cdc533c76c598678202205d9559897795daf66c860785fa292d13af40e1003da478fb6cdd1972983481d80121030e0e6938fc9d9741b948850c39817e69c22171631000967fc8095a9b6b1e830d00000000",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x01460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c4601460000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
            { from: params.accounts[0], gasLimit: 10000000 });
            console.log("Gas used for respondAppendStatement: ", txReceipt.receipt.gasUsed);
        });

        it("test respondAppendStatementWithWrongKey gas usage", async function() {
            var txReceipt = await params.penaltyContract.respondAppendStatementWithWrongKey.sendTransaction(
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c46",
                "0x010000000001012f00c06758bddd3ced39d3110d73994eed7940618a49b72c466fe01f4d3ccf0f0000000000ffffffff02000000000000",
            { from: params.accounts[0], gasLimit: 10000000 });
            console.log("Gas used for respondAppendStatementWithWrongKey: ", txReceipt.receipt.gasUsed);
        });

    });