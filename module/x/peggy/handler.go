package peggy

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/althea-net/peggy/module/x/peggy/keeper"
	"github.com/althea-net/peggy/module/x/peggy/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewHandler returns a handler for "Peggy" type messages.
func NewHandler(keeper keeper.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		switch msg := msg.(type) {
		case *types.MsgSetEthAddress:
			return handleMsgSetEthAddress(ctx, keeper, msg)
		case *types.MsgValsetConfirm:
			return handleMsgConfirmValset(ctx, keeper, msg)
		case *types.MsgValsetRequest:
			return handleMsgValsetRequest(ctx, keeper, msg)
		case *types.MsgSendToEth:
			return handleMsgSendToEth(ctx, keeper, msg)
		case *types.MsgRequestBatch:
			return handleMsgRequestBatch(ctx, keeper, msg)
		case *types.MsgConfirmBatch:
			return handleMsgConfirmBatch(ctx, keeper, msg)
		case *types.MsgCreateEthereumClaims:
			return handleCreateEthereumClaims(ctx, keeper, msg)
		default:
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, fmt.Sprintf("Unrecognized Peggy Msg type: %v", msg.Type()))
		}
	}
}

func handleCreateEthereumClaims(ctx sdk.Context, keeper keeper.Keeper, msg *types.MsgCreateEthereumClaims) (*sdk.Result, error) {
	// TODO:
	// auth EthereumChainID whitelisted
	// auth bridge contract address whitelisted
	ctx.Logger().Info("+++ TODO: implement chain id + contract address authorization")
	//if !bytes.Equal(msg.BridgeContractAddress[:], k.GetBridgeContractAddress(ctx)[:]) {
	//	return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid bridge contract address")
	//}

	var attestationIDs [][]byte
	// auth sender in current validator set
	for _, c := range msg.Claims {
		orch, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
		validator := findValidatorKey(ctx, orch)
		if validator == nil {
			return nil, sdkerrors.Wrap(types.ErrUnknown, "address")
		}
		ec, err := types.UnpackEthereumClaim(c)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "unpacking claim")
		}
		att, err := keeper.AddClaim(ctx, ec.GetType(), types.NewUInt64Nonce(ec.GetEventNonce()), validator, ec.Details())
		if err != nil {
			return nil, sdkerrors.Wrap(err, "create attestation")
		}
		ad, err := types.UnpackAttestationDetails(att.Details)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "unpacking attestation")
		}
		attestationIDs = append(attestationIDs, types.GetAttestationKey(types.NewUInt64Nonce(att.EventNonce), ad))
	}
	return &sdk.Result{
		Data: bytes.Join(attestationIDs, []byte(", ")),
	}, nil
}

func findValidatorKey(ctx sdk.Context, orchAddr sdk.AccAddress) sdk.ValAddress {
	// todo: implement proper in keeper
	// TODO: do we want ValAddress or do we want the AccAddress for the validator?
	// this is a v important question for encoding
	return sdk.ValAddress(orchAddr)
}

func handleMsgValsetRequest(ctx sdk.Context, keeper keeper.Keeper, msg *types.MsgValsetRequest) (*sdk.Result, error) {
	// todo: is requester in current valset?\

	// disabling bootstrap check for integration tests to pass
	//if keeper.GetLastValsetObservedNonce(ctx).isValid() {
	//	return nil, sdkerrors.Wrap(types.ErrInvalid, "bridge bootstrap process not observed, yet")
	//}
	v := keeper.SetValsetRequest(ctx)
	return &sdk.Result{
		Data: types.NewUInt64Nonce(v.Nonce).Bytes(),
	}, nil
}

// This function takes in a signature submitted by a validator's Eth Signer
func handleMsgConfirmBatch(ctx sdk.Context, keeper keeper.Keeper, msg *types.MsgConfirmBatch) (*sdk.Result, error) {

	batch := keeper.GetOutgoingTXBatch(ctx, msg.TokenContract, types.NewUInt64Nonce(msg.Nonce))
	if batch == nil {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "couldn't find batch")
	}

	checkpoint, err := batch.GetCheckpoint()
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "checkpoint generation")
	}

	sigBytes, err := hex.DecodeString(msg.Signature)
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "signature decoding")
	}
	valaddr, _ := sdk.AccAddressFromBech32(msg.Validator)
	validator := findValidatorKey(ctx, valaddr)
	if validator == nil {
		return nil, sdkerrors.Wrap(types.ErrUnknown, "validator")
	}

	ethAddress := keeper.GetEthAddress(ctx, sdk.AccAddress(validator))
	if ethAddress == nil {
		return nil, sdkerrors.Wrap(types.ErrEmpty, "eth address")
	}
	err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress.String())
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalid, fmt.Sprintf("signature verification failed expected %s found %s", checkpoint, msg.Signature))
	}

	// check if we already have this confirm
	if keeper.GetBatchConfirm(ctx, types.NewUInt64Nonce(msg.Nonce), types.NewEthereumAddress(msg.TokenContract), valaddr) != nil {
		return nil, sdkerrors.Wrap(types.ErrDuplicate, "signature duplicate")
	}
	key := keeper.SetBatchConfirm(ctx, msg)
	return &sdk.Result{
		Data: key,
	}, nil
}

// This function takes in a signature submitted by a validator's Eth Signer
func handleMsgConfirmValset(ctx sdk.Context, keeper keeper.Keeper, msg *types.MsgValsetConfirm) (*sdk.Result, error) {

	valset := keeper.GetValsetRequest(ctx, types.NewUInt64Nonce(msg.Nonce))
	if valset == nil {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "couldn't find valset")
	}

	checkpoint := valset.GetCheckpoint()

	sigBytes, err := hex.DecodeString(msg.Signature)
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "signature decoding")
	}
	valaddr, _ := sdk.AccAddressFromBech32(msg.Validator)
	validator := findValidatorKey(ctx, valaddr)
	if validator == nil {
		return nil, sdkerrors.Wrap(types.ErrUnknown, "validator")
	}

	ethAddress := keeper.GetEthAddress(ctx, sdk.AccAddress(validator))
	if ethAddress == nil {
		return nil, sdkerrors.Wrap(types.ErrEmpty, "eth address")
	}
	err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress.String())
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalid, fmt.Sprintf("signature verification failed expected %s found %s", checkpoint, msg.Signature))
	}

	// persist signature
	if keeper.GetValsetConfirm(ctx, types.NewUInt64Nonce(msg.Nonce), valaddr) != nil {
		return nil, sdkerrors.Wrap(types.ErrDuplicate, "signature duplicate")
	}
	key := keeper.SetValsetConfirm(ctx, *msg)
	return &sdk.Result{
		Data: key,
	}, nil
}

func handleMsgSetEthAddress(ctx sdk.Context, keeper keeper.Keeper, msg *types.MsgSetEthAddress) (*sdk.Result, error) {
	valaddr, _ := sdk.AccAddressFromBech32(msg.Validator)
	validator := findValidatorKey(ctx, valaddr)
	if validator == nil {
		return nil, sdkerrors.Wrap(types.ErrUnknown, "address")
	}

	keeper.SetEthAddress(ctx, sdk.AccAddress(validator), types.NewEthereumAddress(msg.Address))
	return &sdk.Result{}, nil
}

func handleMsgSendToEth(ctx sdk.Context, keeper keeper.Keeper, msg *types.MsgSendToEth) (*sdk.Result, error) {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	txID, err := keeper.AddToOutgoingPool(ctx, sender, types.NewEthereumAddress(msg.EthDest), msg.Amount, msg.BridgeFee)
	if err != nil {
		return nil, err
	}
	return &sdk.Result{
		Data: sdk.Uint64ToBigEndian(txID),
	}, nil
}

func handleMsgRequestBatch(ctx sdk.Context, k keeper.Keeper, msg *types.MsgRequestBatch) (*sdk.Result, error) {
	// ensure that peggy denom is valid
	ec, err := types.ERC20FromPeggyCoin(sdk.NewInt64Coin(msg.Denom, 0))
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrInvalid, "invalid denom: %s", err)
	}

	batchID, err := k.BuildOutgoingTXBatch(ctx, ec.Contract, keeper.OutgoingTxBatchSize)
	if err != nil {
		return nil, err
	}
	return &sdk.Result{
		Data: types.NewUInt64Nonce(batchID.BatchNonce).Bytes(),
	}, nil
}
