package gravity

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/althea-net/cosmos-gravity-bridge/module/x/gravity/keeper"
)

func TestEthereumOriginated(t *testing.T) {
	env := keeper.CreateTestEnv(t)
	ctx := env.Context
	k := env.GravityKeeper
	tokens := [5]string{
		"0x1bAf18b886e7b957DdE69B20f9309920d3bf6cd2",
		"0xE1BBed0535b41497F0022D8Add45dD5f30e33ba9",
		"0xFefdE2f3e5b8C8B9B6838DC87DF9cB23684ea699",
		"0xd81ae426109eA3Ca9931E28583F59cd70aFbE196",
		"0xfF783f369Caa7A5Ab1441838D1b87B1121A9Ce40"}
	notInList := "0xeAA45200d903F08E1612fdbC82AF78993E503789"
	invalid := "bad"

	assert.Panics(t, func() { k.SetEthereumOriginatedErc20(ctx, invalid) }, "The code did not panic")

	for _, token := range tokens {
		k.SetEthereumOriginatedErc20(ctx, token)
	}
	for _, token := range tokens {
		res := k.GetEthereumOriginatedErc20(ctx, token)
		assert.True(t, res)
	}
	res := k.GetEthereumOriginatedErc20(ctx, notInList)
	assert.False(t, res)

	storeTokens := k.GetEthereumOriginatedErc20Tokens(ctx)
	sort.Strings(storeTokens)
	assert.Equal(t, storeTokens, tokens[:])
}
