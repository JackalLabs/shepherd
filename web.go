package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gorilla/mux"
	filetreeTypes "github.com/jackalLabs/canine-chain/v3/x/filetree/types"
	rnsTypes "github.com/jackalLabs/canine-chain/v3/x/rns/types"
	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"github.com/rs/cors"
)

type ContentResponse struct {
	Fids []string `json:"fids"`
}

func hashAndHex(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	data := h.Sum(nil)

	return hex.EncodeToString(data)
}

func hexFullPath(path string, fileName string) string {
	return hashAndHex(fmt.Sprintf("%s%s", path, hashAndHex(fileName)))
}

func merkleMeBro(rawpath string) string {
	pathArray := strings.Split(rawpath, "/")
	merkle := ""
	for i := 0; i < len(pathArray); i++ {
		merkle = hexFullPath(merkle, pathArray[i])
	}

	return merkle
}

func initRouter(ctx client.Context) http.Handler {
	router := mux.NewRouter()
	router.SkipClean(true)

	router.HandleFunc("/f/{fid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fid := vars["fid"]

		if len(fid) < 1 {
			fmt.Println("needs to supply fid")
			w.WriteHeader(400)
			return
		}

		qc := storageTypes.NewQueryClient(ctx)

		err := downloadFile(qc, fid, w, false, "")
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(500)
			return
		}
	})

	downloadByPath := func(isMarkdown bool) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			owner := vars["owner"]

			path := vars["path"]

			if len(path) < 1 {
				fmt.Println("needs to supply path")
				w.WriteHeader(400)
				return
			}

			rqc := rnsTypes.NewQueryClient(ctx)
			qc := filetreeTypes.NewQueryClient(ctx)
			sqc := storageTypes.NewQueryClient(ctx)

			splitPath := strings.Split(path, "/")

			rawPath := "s"
			for i := 0; i < len(splitPath); i++ {
				rawPath = rawPath + "/" + splitPath[i]
			}
			_, _, err := bech32.Decode(owner)
			if err != nil {
				rnsReq := rnsTypes.QueryNameRequest{
					Index: fmt.Sprintf("%s.jkl", owner),
				}
				rnsRes, err := rqc.Names(context.Background(), &rnsReq)
				if err != nil {
					fmt.Println(err)
					w.WriteHeader(500)
					return
				}

				owner = rnsRes.Names.Value
			}

			hexAddress := merkleMeBro(rawPath)
			hexedOwner := hashAndHex(fmt.Sprintf("o%s%s", hexAddress, hashAndHex(owner)))

			req := filetreeTypes.QueryFileRequest{
				Address:      hexAddress,
				OwnerAddress: hexedOwner,
			}

			res, err := qc.Files(context.Background(), &req)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(500)
				return
			}

			var contents ContentResponse

			err = json.Unmarshal([]byte(res.Files.Contents), &contents)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(500)
				return
			}

			fids := contents.Fids
			fid := fids[0]

			err = downloadFile(sqc, fid, w, isMarkdown, splitPath[len(splitPath)-1])
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(500)
				return
			}
		}
	}

	router.HandleFunc(`/p/{owner}/{path:.+}`, downloadByPath(false))

	router.HandleFunc(`/md/{owner}/{path:.+}`, downloadByPath(true))

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := "Shepherd Gateway"
		w.Write([]byte(s))
	})

	handler := cors.Default().Handler(router)

	return handler
}

func startServer(ctx client.Context, rpc string, port int64) {
	handler := initRouter(ctx)

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
