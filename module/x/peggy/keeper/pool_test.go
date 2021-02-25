package keeper

import (
	"math/big"
	"testing"

	"github.com/althea-net/peggy/module/x/peggy/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToOutgoingPool(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context
	var (
		mySender, _         = sdk.AccAddressFromBech32("cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn")
		myReceiver          = "0xd041c41EA1bf0F006ADBb6d2c9ef9D425dE5eaD7"
		myTokenContractAddr = "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5"
	)
	// mint some voucher first
	allVouchers := sdk.Coins{types.NewERC20Token(99999, myTokenContractAddr).PeggyCoin()}
	err := input.BankKeeper.MintCoins(ctx, types.ModuleName, allVouchers)
	require.NoError(t, err)

	// set senders balance
	input.AccountKeeper.NewAccountWithAddress(ctx, mySender)
	err = input.BankKeeper.SetBalances(ctx, mySender, allVouchers)
	require.NoError(t, err)

	// when
	for i, v := range []uint64{2, 3, 2, 1} {
		amount := types.NewERC20Token(uint64(i+100), myTokenContractAddr).PeggyCoin()
		fee := types.NewERC20Token(v, myTokenContractAddr).PeggyCoin()
		r, err := input.PeggyKeeper.AddToOutgoingPool(ctx, mySender, myReceiver, amount, fee)
		require.NoError(t, err)
		t.Logf("___ response: %#v", r)
	}
	// then
	var got []*types.OutgoingTx
	input.PeggyKeeper.IterateOutgoingPoolByFee(ctx, myTokenContractAddr, func(_ uint64, tx *types.OutgoingTx) bool {
		got = append(got, tx)
		return false
	})
	exp := []*types.OutgoingTx{
		{
			BridgeFee: types.NewERC20Token(3, myTokenContractAddr).PeggyCoin(),
			Sender:    mySender.String(),
			DestAddr:  myReceiver,
			Amount:    types.NewERC20Token(101, myTokenContractAddr).PeggyCoin(),
		},
		{
			BridgeFee: types.NewERC20Token(2, myTokenContractAddr).PeggyCoin(),
			Sender:    mySender.String(),
			DestAddr:  myReceiver,
			Amount:    types.NewERC20Token(100, myTokenContractAddr).PeggyCoin(),
		},
		{
			BridgeFee: types.NewERC20Token(2, myTokenContractAddr).PeggyCoin(),
			Sender:    mySender.String(),
			DestAddr:  myReceiver,
			Amount:    types.NewERC20Token(102, myTokenContractAddr).PeggyCoin(),
		},
		{
			BridgeFee: types.NewERC20Token(1, myTokenContractAddr).PeggyCoin(),
			Sender:    mySender.String(),
			DestAddr:  myReceiver,
			Amount:    types.NewERC20Token(103, myTokenContractAddr).PeggyCoin(),
		},
	}
	assert.Equal(t, exp, got)
}

func TestTotalFeeForBatchPool(t *testing.T) {
	input := CreateTestEnv(t)
	ctx := input.Context

	// token1
	var (
		mySender, _         = sdk.AccAddressFromBech32("cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn")
		myReceiver          = "0xd041c41EA1bf0F006ADBb6d2c9ef9D425dE5eaD7"
		myTokenContractAddr = "0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5"
	)
	// mint some voucher first
	allVouchers := sdk.Coins{types.NewERC20Token(99999, myTokenContractAddr).PeggyCoin()}
	err := input.BankKeeper.MintCoins(ctx, types.ModuleName, allVouchers)
	require.NoError(t, err)

	// set senders balance
	input.AccountKeeper.NewAccountWithAddress(ctx, mySender)
	err = input.BankKeeper.SetBalances(ctx, mySender, allVouchers)
	require.NoError(t, err)

	// create outgoing pool
	for i, v := range []uint64{2, 3, 2, 1} {
		amount := types.NewERC20Token(uint64(i+100), myTokenContractAddr).PeggyCoin()
		fee := types.NewERC20Token(v, myTokenContractAddr).PeggyCoin()
		r, err := input.PeggyKeeper.AddToOutgoingPool(ctx, mySender, myReceiver, amount, fee)
		require.NoError(t, err)
		t.Logf("___ response: %#v", r)
	}

	// token 2
	var (
		myToken2ContractAddr = "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0"
	)
	// mint some voucher first
	allVouchers = sdk.Coins{types.NewERC20Token(18446744073709551615, myToken2ContractAddr).PeggyCoin()}
	err = input.BankKeeper.MintCoins(ctx, types.ModuleName, allVouchers)
	require.NoError(t, err)

	// set senders balance
	input.AccountKeeper.NewAccountWithAddress(ctx, mySender)
	err = input.BankKeeper.SetBalances(ctx, mySender, allVouchers)
	require.NoError(t, err)

	// create outgoing pool
	for i, v := range []uint64{4, 1844674407370955141, 1844674407370955141, 1844674407370955141} {
		amount := types.NewERC20Token(uint64(i+100), myToken2ContractAddr).PeggyCoin()
		fee := types.NewERC20Token(v, myToken2ContractAddr).PeggyCoin()
		r, err := input.PeggyKeeper.AddToOutgoingPool(ctx, mySender, myReceiver, amount, fee)
		require.NoError(t, err)
		t.Logf("___ response: %#v", r)
	}

	tokenFeeMap := input.PeggyKeeper.CreateTokenFeeMap(ctx)
	/*
		tokenFeeMap should be
		map[0x429881672B9AE42b8EbA0E26cD9C73711b891Ca5:8 0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0:5534023222112865427]
		**/
	assert.Equal(t, tokenFeeMap[myTokenContractAddr], big.NewInt(int64(8)).String())
	assert.Equal(t, tokenFeeMap[myToken2ContractAddr], big.NewInt(int64(5534023222112865427)).String())
}
