package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

func initRouter(ctx client.Context) http.Handler {
	router := httprouter.New()

	router.GET("/f/:fid", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		fid := ps.ByName("fid")

		if len(fid) < 1 {
			fmt.Println("needs to supply fid")
			w.WriteHeader(500)
			return
		}

		qc := storageTypes.NewQueryClient(ctx)

		err := downloadFile(qc, fid, w)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(500)
			return
		}
	})

	handler := cors.Default().Handler(router)

	return handler
}

func startServer(ctx client.Context, rpc string) {

	handler := initRouter(ctx)

	port := 5656

	fmt.Printf("🌍 Started Shepherd: http://0.0.0.0:%d using %s\n", port, rpc)
	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), handler)
	if err != nil {
		fmt.Println(err)
		return
	}

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Storage Provider Closed\n")
		return
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}

}
