import chai from "chai";
import { ethers } from "@nomiclabs/buidler";
import { solidity } from "ethereum-waffle";

import { deployContracts } from "../test-utils";
import {
  getSignerAddresses,
  makeCheckpoint,
  signHash,
  makeTxBatchHash,
  examplePowers
} from "../test-utils/pure";

chai.use(solidity);
const { expect } = chai;


async function runTest(opts: {
  // Issues with the new valset
  malformedNewValset?: boolean;
  newValsetNonceNotIncremented?: boolean;

  // Issues with the tx batch
  batchNonceNotHigher?: boolean;
  malformedTxBatch?: boolean;

  // Issues with the current valset and signatures
  nonMatchingCurrentValset?: boolean;
  badValidatorSig?: boolean;
  zeroedValidatorSig?: boolean;
  notEnoughPower?: boolean;
  barelyEnoughPower?: boolean;
  malformedCurrentValset?: boolean;
}) {



  // Prep and deploy contract
  // ========================
  const signers = await ethers.getSigners();
  const peggyId = ethers.utils.formatBytes32String("foo");
  // This is the power distribution on the Cosmos hub as of 7/14/2020
  let powers = examplePowers();
  let validators = signers.slice(0, powers.length);
  const powerThreshold = 6666;
  const {
    peggy,
    testERC20,
    checkpoint: deployCheckpoint
  } = await deployContracts(peggyId, validators, powers, powerThreshold);



  // Make new valset
  // ===============
  let newPowers = examplePowers();
  newPowers[0] -= 3;
  newPowers[1] += 3;
  let newValidators = signers.slice(0, newPowers.length);
  if (opts.malformedNewValset) {
    // Validators and powers array don't match
    newValidators = signers.slice(0, newPowers.length - 1);
  }
  let newValsetNonce = 1;
  if (opts.newValsetNonceNotIncremented) {
    newValsetNonce = 0;
  }
  const checkpoint = makeCheckpoint(
    await getSignerAddresses(newValidators),
    newPowers,
    newValsetNonce,
    peggyId
  );



  // Transfer out to Cosmos, locking coins
  // =====================================
  await testERC20.functions.approve(peggy.address, 1000);
  await peggy.functions.sendToCosmos(
    testERC20.address,
    ethers.utils.formatBytes32String("myCosmosAddress"),
    1000
  );



  // Prepare batch
  // ===============================
  const numTxs = 100;
  const txDestinationsInt = new Array(numTxs);
  const txFees = new Array(numTxs);

  const txAmounts = new Array(numTxs);
  for (let i = 0; i < numTxs; i++) {
    txFees[i] = 1;
    txAmounts[i] = 1;
    txDestinationsInt[i] = signers[i + 5];
  }
  const txDestinations = await getSignerAddresses(txDestinationsInt);
  if (opts.malformedTxBatch) {
    // Make the fees array the wrong size
    txFees.pop();
  }

  let batchNonce = 1
  if (opts.batchNonceNotHigher) {
    batchNonce = 0
  }


  // Call method
  // ===========
  const methodName = ethers.utils.formatBytes32String(
    "valsetAndTransactionBatch"
  );
  let abiEncoded = ethers.utils.defaultAbiCoder.encode(
    [
      "bytes32",
      "bytes32",
      "bytes32",
      "uint256[]",
      "address[]",
      "uint256[]",
      "uint256",
      "address"
    ],
    [
      peggyId,
      methodName,
      checkpoint,
      txAmounts,
      txDestinations,
      txFees,
      batchNonce,
      testERC20.address
    ]
  );
  let digest = ethers.utils.keccak256(abiEncoded);
  let sigs = await signHash(validators, digest);
  let currentValsetNonce = 0;
  if (opts.nonMatchingCurrentValset) {
    // Wrong nonce
    currentValsetNonce = 420;
  }
  if (opts.malformedCurrentValset) {
    // Remove one of the powers to make the length not match
    powers.pop();
  }
  if (opts.badValidatorSig) {
    // Switch the first sig for the second sig to screw things up
    sigs.v[1] = sigs.v[0];
    sigs.r[1] = sigs.r[0];
    sigs.s[1] = sigs.s[0];
  }
  if (opts.zeroedValidatorSig) {
    // Switch the first sig for the second sig to screw things up
    sigs.v[1] = sigs.v[0];
    sigs.r[1] = sigs.r[0];
    sigs.s[1] = sigs.s[0];
    // Then zero it out to skip evaluation
    sigs.v[1] = 0;
  }
  if (opts.notEnoughPower) {
    // zero out enough signatures that we dip below the threshold
    sigs.v[1] = 0;
    sigs.v[2] = 0;
    sigs.v[3] = 0;
    sigs.v[5] = 0;
    sigs.v[6] = 0;
    sigs.v[7] = 0;
    sigs.v[9] = 0;
    sigs.v[11] = 0;
    sigs.v[13] = 0;
  }
  if (opts.barelyEnoughPower) {
    // Stay just above the threshold
    sigs.v[1] = 0;
    sigs.v[2] = 0;
    sigs.v[3] = 0;
    sigs.v[5] = 0;
    sigs.v[6] = 0;
    sigs.v[7] = 0;
    sigs.v[9] = 0;
    sigs.v[11] = 0;
  }

  // await peggy.updateValsetAndSubmitBatch(
  //   await getSignerAddresses(validators),
  //   powers,
  //   currentValsetNonce,
  //   sigs.v,
  //   sigs.r,
  //   sigs.s,
  //   await getSignerAddresses(newValidators),
  //   newPowers,
  //   newValsetNonce,
  //   txAmounts,
  //   txDestinations,
  //   txFees,
  //   txNonces
  // );

  await peggy.updateValsetAndSubmitBatch(

    await getSignerAddresses(newValidators),
    newPowers,
    newValsetNonce,

    await getSignerAddresses(validators),
    powers,
    currentValsetNonce,

    sigs.v,
    sigs.r,
    sigs.s,

    txAmounts,
    txDestinations,
    txFees,
    batchNonce,
    testERC20.address
  );
}

describe("updateValsetAndSubmitBatch tests", function () {
  it("throws on malformed current valset", async function () {
    await expect(runTest({ malformedCurrentValset: true })).to.be.revertedWith(
      "Malformed current validator set"
    );
  });

  it("throws on malformed new valset", async function () {
    await expect(runTest({ malformedNewValset: true })).to.be.revertedWith(
      "Malformed new validator set"
    );
  });

  it("throws on new valset nonce not incremented", async function () {
    await expect(runTest({ newValsetNonceNotIncremented: true })).to.be.revertedWith(
      "New valset nonce must be greater than the current nonce"
    );
  });

  it("throws on malformed txbatch", async function () {
    await expect(runTest({ malformedTxBatch: true })).to.be.revertedWith(
      "Malformed batch of transactions"
    );
  });

  it("throws on batch nonce not incremented", async function () {
    await expect(runTest({ batchNonceNotHigher: true })).to.be.revertedWith(
      "New batch nonce must be greater than the current nonce"
    );
  });

  it("throws on non matching checkpoint for current valset", async function () {
    await expect(
      runTest({ nonMatchingCurrentValset: true })
    ).to.be.revertedWith(
      "Supplied current validators and powers do not match checkpoint"
    );
  });


  it("throws on bad validator sig", async function () {
    await expect(runTest({ badValidatorSig: true })).to.be.revertedWith(
      "Validator signature does not match"
    );
  });

  it("allows zeroed sig", async function () {
    await runTest({ zeroedValidatorSig: true });
  });

  it("throws on not enough signatures", async function () {
    await expect(runTest({ notEnoughPower: true })).to.be.revertedWith(
      "Submitted validator set signatures do not have enough power"
    );
  });

  it("does not throw on barely enough signatures", async function () {
    await runTest({ barelyEnoughPower: true });
  });
});
