package main

import (
	"os"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/std"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Logger = log.With().Caller().Logger()
	log.Logger = log.Level(zerolog.InfoLevel)

	rpc := os.Getenv("RPC")
	if rpc == "" {
		rpc = "https://rpc.jackalprotocol.com:443"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "5656"
	}
	portNum, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		panic(err)
	}

	encodingConfig := params.MakeTestEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	simapp.ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	simapp.ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	cfg := sdk.GetConfig()
	cfg.Seal()

	cl, err := client.NewClientFromNode(rpc)
	if err != nil {
		log.Error().Err(err)
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

	startServer(ctx, rpc, portNum)
}
