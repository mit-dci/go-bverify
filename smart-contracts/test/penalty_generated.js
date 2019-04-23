
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
                "0xda1feb9bf385ea8167b32a638e0f5431e8b98755fd92b7abe62b94715abb445c4553b4bbc0d051a9bf731d9ea0a1347b4899d8a8c37eff2781277443aa99090bcdcd6ecf15b02a1991661e23f27f2e5314f03cce9b5df00880f1bd331f15b80e012094a61d77bd5ef28d3e126bf5a2d27f1bc8e8972dfb73b73dd2ef55c11418d726",
                "0x046f2780a2b33a0854156ad0f377146f02f7bcfc3f01863ea8dd07d1e3ade743faab79a6d8c68ec6287c7541fd2066a35d3740319830734616bbdfafa4dc8fb99b",
                28,
                "0xda1feb9bf385ea8167b32a638e0f5431e8b98755fd92b7abe62b94715abb445c",
                "0x4553b4bbc0d051a9bf731d9ea0a1347b4899d8a8c37eff2781277443aa99090b",
                
                
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should accept a valid challenge with a wrong logID", async function() {
             await params.penaltyContract.challengeLackOfProof.sendTransaction(
                "0x3782034830fed6bdf61414d40cfd72a1346899de0c7803535c33b4b14e5f068b023303b476a0e07c6f146598346081bfe7719b1f4ddcb5f565a5ba7f36af5440cdcd6ecf15b02a1991661e23f27f2e5314f03cce9b5df00880f1bd331f15b80e022093125478b9c24a3c2a05f20b6b8a0b37109ccd6e54d92429e9d25ddb76cbe24c",
                "0x04fec095aef9cf31a48f9411786d6add3fc3c111c88af3008519b88b1083e6f29fc3835e8f4328a783aa896782954c2912695c590fce3f1a77af6b385e2fc48841",
                28,
                "0x3782034830fed6bdf61414d40cfd72a1346899de0c7803535c33b4b14e5f068b",
                "0x023303b476a0e07c6f146598346081bfe7719b1f4ddcb5f565a5ba7f36af5440",
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should accept a valid 2nd challenge", async function() {
            await params.penaltyContract.challengeLackOfProof.sendTransaction(
                "0x9cce00870ef4b8e542061d85907f078ac0c2c69f7b0e7f283a01f8b9bc896c1e63a0199578f1546d3fd3b2b859f67a85ccd7268af0562e0968e3e8997ec85638cdcd6ecf15b02a1991661e23f27f2e5314f03cce9b5df00880f1bd331f15b80e01204ab0f90ba990712db752029640763202751ccc5f51b260d5eca282f653ea7a96",
                "0x046f2780a2b33a0854156ad0f377146f02f7bcfc3f01863ea8dd07d1e3ade743faab79a6d8c68ec6287c7541fd2066a35d3740319830734616bbdfafa4dc8fb99b",
                28,
                "0x9cce00870ef4b8e542061d85907f078ac0c2c69f7b0e7f283a01f8b9bc896c1e",
                "0x63a0199578f1546d3fd3b2b859f67a85ccd7268af0562e0968e3e8997ec85638",
            { from: params.accounts[0], gasLimit: 10000000 });
        });

        it("should disallow withdrawing collateral within challenge period", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0x60a0af72093f30340c3e2e4651a12cb2c350646658d35bd375e4f59a09cfbaf8",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

        it("should return false for challengeResponded before response", async function() {
            var result = await params.penaltyContract.challengeResponded.call("0x60a0af72093f30340c3e2e4651a12cb2c350646658d35bd375e4f59a09cfbaf8");
            assert.isFalse(result);
        });

        it("should not accept a invalid wrong key response", async function() {
            await exceptions.catchRevert(params.penaltyContract.respondLackOfProofWithWrongKey.sendTransaction(
                "0x60a0af72093f30340c3e2e4651a12cb2c350646658d35bd375e4f59a09cfbaf8",
                "0x036f2780a2b33a0854156ad0f377146f02f7bcfc3f01863ea8dd07d1e3ade743fa20986ee148d0906f9335f2fe790154e7b260fdaed34fe1e3916f6d26f93a95a0f1",
            { from: params.accounts[1], gasLimit: 10000000 }));
        });

        it("should accept a valid wrong key response", async function() {
            await params.penaltyContract.respondLackOfProofWithWrongKey.sendTransaction(
                "0x8b9b33ed4b8f44bed29ca8618e766e5446bdcf8e045387d20029ba9d7ad2b7a4",
                "0x036f2780a2b33a0854156ad0f377146f02f7bcfc3f01863ea8dd07d1e3ade743fa20986ee148d0906f9335f2fe790154e7b260fdaed34fe1e3916f6d26f93a95a0f1",
                { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should accept a valid proof response", async function() {
           params.penaltyContract.respondLackOfProofWithProof.sendTransaction(
                "0x60a0af72093f30340c3e2e4651a12cb2c350646658d35bd375e4f59a09cfbaf8",
                "0x",
                "0x",
            { from: params.accounts[1], gasLimit: 10000000 });
        });

        it("should return true for challengeResponded after response", async function() {
            var result = await params.penaltyContract.challengeResponded.call("0x60a0af72093f30340c3e2e4651a12cb2c350646658d35bd375e4f59a09cfbaf8");
            assert.isTrue(result);
        });

        it("should disallow withdrawing collateral when marked resolved", async function() {
            await exceptions.catchRevert(params.penaltyContract.withdrawCollateral.sendTransaction(
                "0x60a0af72093f30340c3e2e4651a12cb2c350646658d35bd375e4f59a09cfbaf8",
            { from: params.accounts[0], gasLimit: 10000000 }));
        });

 		it("should allow withdrawing collateral when challenge period ends", async function() {
            // Wait for expiry
            while(!(await params.penaltyContract.challengeExpired.call("0xf55c3621d4c79a26fd62638263f14cf17fa73d8f4e86b277df7919ea01751747"))) {
                await sleep(500);
            }
        
            var balBefore = await web3.eth.getBalance(params.accounts[0])
            await params.penaltyContract.withdrawCollateral.sendTransaction(
                "0xf55c3621d4c79a26fd62638263f14cf17fa73d8f4e86b277df7919ea01751747",
            { from: params.accounts[0], gasLimit: 10000000 });
            var balAfter = await web3.eth.getBalance(params.accounts[0])

            // Difference should be 10 ETH -/- gas cost for sending
            expectedBalanceDiff = "9998839720000000000";
            realBalanceDiff  = new BN(balAfter).sub(new BN(balBefore)).toString();
            assert(expectedBalanceDiff === realBalanceDiff, "Balance increase did not match collateral " + expectedBalanceDiff + " vs " + realBalanceDiff);
        });
    });