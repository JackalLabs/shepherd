package main

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/std"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("Usage: %s [RPC-ENDPOINT]\n", args[0])
		return
	}
	rpc := args[1]

	encodingConfig := params.MakeTestEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	simapp.ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	simapp.ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	cfg := sdk.GetConfig()
	cfg.Seal()

	cl, err := client.NewClientFromNode(rpc)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastBlock).
		WithHomeDir("upgradetracker").
		WithViper("").
		WithNodeURI(rpc).
		WithClient(cl)

	//downloadFile(qc, "jklf1p5cm3z47rrcyaskqge3yc33xm7hdq7lken99ahluvuz67ugctleqmwv43a")

	startServer(ctx)
}
