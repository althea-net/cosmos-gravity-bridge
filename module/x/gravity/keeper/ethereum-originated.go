package keeper

import (
	"github.com/althea-net/cosmos-gravity-bridge/module/x/gravity/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// setEthereumOriginatedErc20 add a token to the list of tokens originating from Ethereum
func (k Keeper) SetEthereumOriginatedErc20(ctx sdk.Context, tokenContract string) {
	if types.ValidateEthAddress(tokenContract) == nil {
		store := ctx.KVStore(k.storeKey)
		store.Set(types.GetEthereumOriginatedTokensKey([]byte(tokenContract)), []byte{0x0})
	} else {
		panic("Tried to set non ERC20 string as Ethereum originated ERC20!")
	}
}

// getEthereumOriginatedErc20 determines if a a given token contract is in the list of ERC20
// tokens brought over from Ethereum to this Cosmos chain.
func (k Keeper) GetEthereumOriginatedErc20(ctx sdk.Context, tokenContract string) bool {
	store := ctx.KVStore(k.storeKey)
	res := store.Get(types.GetEthereumOriginatedTokensKey([]byte(tokenContract)))
	if res != nil {
		return true
	} else {
		return false
	}
}

// GetEthereumOriginatedErc20Tokens gets all ERC20 token addresses that have been bridge to this chain at
// any time
func (k Keeper) GetEthereumOriginatedErc20Tokens(ctx sdk.Context) (out []string) {
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.EthereumOriginatedTokensKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var erc20 string = string(iter.Key()[len(types.EthereumOriginatedTokensKey)-1:])
		out = append(out, erc20)
	}
	return out
}
