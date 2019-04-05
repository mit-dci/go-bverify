
const BVerifyPenalty = artifacts.require("BVerifyPenalty")
const BN = require('bn.js');
const Web3 = require("web3");

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

        it("should accept a call with valid payload", async function() {
            var tx0 = Buffer.from("0100000000010160cc1a18208363de4c97d184bb8e021028d4b5f541a2327ae2264555ec8a73560100000000ffffffff020000000000000000226a20d3bea4803f5d80b56e4d5085a9f0620defdd657aea74e04b410b6ca965156bd57833860000000000160014f7ccc2053561be6896b777a5be8d2cbfeef3fea702473044022060bb937ebaeb573a2628c7d28f73a5836d9f7c40fa8e6011009d0924ab22b754022036765d573ef30262b945fedc048ffa3ffd0e5f9290d0a8f130f5d109830d52310121030e0e6938fc9d9741b948850c39817e69c22171631000967fc8095a9b6b1e830d00000000", "hex");
            var tx1 = Buffer.from("0100000000010157ebf4fe8f22c5c25001cd2cdb36c352d7d679303c6234745cb0d3e97a053f5d0100000000ffffffff020000000000000000226a2096657a28218d689563a2339607fc14e015ce6e4195e01d7eaf6164a67731605e902f860000000000160014f7ccc2053561be6896b777a5be8d2cbfeef3fea70248304502210080c09b6ef4106c526f47b5501a9894fb34b8a2581d62484880c31ea0901b98b402204f7c825117d82643cc4fa0e0487ea700ad0e7f298d9c0259ae7c214d5824a43b0121030e0e6938fc9d9741b948850c39817e69c22171631000967fc8095a9b6b1e830d00000000", "hex");
            var srvrAck = Buffer.from("000000000000000000000000000000000","hex");

            var param = Buffer.alloc(12+tx0.length+tx1.length+srvrAck.length);
            param.writeInt32BE(tx0.length, 0);
            tx0.copy(param, 4);
            param.writeInt32BE(tx1.length, 4+tx0.length);
            tx1.copy(param, 8+tx0.length);
            param.writeInt32BE(srvrAck.length, 8+tx0.length+tx1.length);
            srvrAck.copy(param, 12+tx0.length+tx1.length);

            var stringParam = "0x" + param.toString("hex");
            console.log(stringParam);

            await params.penaltyContract.challenge.sendTransaction(stringParam, { from: params.accounts[0] });
        });
    });

    