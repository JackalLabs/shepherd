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
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger/v4"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gorilla/mux"
	filetreeTypes "github.com/jackalLabs/canine-chain/v3/x/filetree/types"
	rnsTypes "github.com/jackalLabs/canine-chain/v3/x/rns/types"
	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
)

type ContentResponse struct {
	Fids []string `json:"fids"`
}

func addToMerkle(path string, append string) string {
	total := path

	k := fmt.Sprintf("%s%s", total, append)

	h := sha256.New()
	h.Write([]byte(k))
	total = fmt.Sprintf("%x", h.Sum(nil))
	return total
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

func makeDBAddress(owner, path string) []byte {
	return []byte(fmt.Sprintf("%s_%s", owner, path))
}

func downloadByPath(ctx client.Context, db *badger.DB, isMarkdown bool, web bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		owner := vars["owner"]
		path := vars["path"]

		if len(path) < 1 {
			log.Warn().Msg("needs to supply path")
			w.WriteHeader(400)
			return
		}

		adr := makeDBAddress(owner, path)

		var item []byte
		err := db.View(func(txn *badger.Txn) error {
			i, err := txn.Get(adr)
			if err != nil {
				return err
			}
			_ = i.Value(func(val []byte) error {
				item = val
				return nil
			})
			return nil
		})
		if err == nil {
			log.Info().Msgf("Found %s in cache...", path)
		}
		if item != nil {
			ext := filepath.Ext(path)
			if len(ext) > 0 {
				mime := mimeTypes[ext]
				if len(mime) > 0 {
					w.Header().Set("Content-Type", mime)
				}
			}
			_, _ = w.Write(item)
			log.Info().Msgf("Served %s from cache.", path)
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

		if web {
			base := filepath.Base(rawPath)
			if !strings.Contains(base, ".") {
				rawPath = filepath.Join(rawPath, "index.html")
			}
		}

		_, _, err = bech32.Decode(owner)
		if err != nil {
			rnsReq := rnsTypes.QueryNameRequest{
				Index: fmt.Sprintf("%s.jkl", owner),
			}
			rnsRes, err := rqc.Names(context.Background(), &rnsReq)
			if err != nil {
				log.Warn().Err(err)
				w.WriteHeader(500)
				return
			}

			owner = rnsRes.Names.Value
		}

		parentHex := merkleMeBro(filepath.Dir(rawPath))
		myHex := hashAndHex(filepath.Base(rawPath))

		hexAddress := addToMerkle(parentHex, myHex)

		hexedOwner := hashAndHex(fmt.Sprintf("o%s%s", hexAddress, hashAndHex(owner)))

		req := filetreeTypes.QueryFileRequest{
			Address:      hexAddress,
			OwnerAddress: hexedOwner,
		}

		res, err := qc.Files(context.Background(), &req)
		if err != nil {
			log.Warn().Msg(fmt.Errorf("cannot find file on jackal %w | %s", err, path).Error())
			w.WriteHeader(500)
			return
		}

		var contents ContentResponse

		err = json.Unmarshal([]byte(res.Files.Contents), &contents)
		if err != nil {
			log.Warn().Msg(fmt.Errorf("cannot unmarshal file %w", err).Error())
			w.WriteHeader(500)
			return
		}

		fids := contents.Fids
		fid := fids[0]

		err = downloadFile(sqc, db, adr, filepath.Base(rawPath), fid, w, isMarkdown, splitPath[len(splitPath)-1])
		if err != nil {
			log.Warn().Msg(fmt.Errorf("cannot download file %w", err).Error())
			w.WriteHeader(500)
			return
		}
	}
}

func rnsSite(ctx client.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		rns := vars["rns"]
		path := vars["path"]

		rqc := rnsTypes.NewQueryClient(ctx)
		qc := filetreeTypes.NewQueryClient(ctx)
		sqc := storageTypes.NewQueryClient(ctx)

		rnsReq := rnsTypes.QueryNameRequest{
			Index: fmt.Sprintf("%s.jkl", rns),
		}
		rnsRes, err := rqc.Names(context.Background(), &rnsReq)
		if err != nil {
			log.Warn().Err(err)
			w.WriteHeader(500)
			return
		}

		var rnsWWW RNSWeb
		nameData := rnsRes.Names.Data
		err = json.Unmarshal([]byte(nameData), &rnsWWW)
		if err != nil {
			log.Warn().Err(err)
			w.WriteHeader(500)
			return
		}

		path = filepath.Join("s", rnsWWW.WWW, path)

		base := filepath.Base(path)
		if !strings.Contains(base, ".") {
			path = filepath.Join(path, "index.html")
		}

		parentHex := merkleMeBro(filepath.Dir(path))
		myHex := hashAndHex(filepath.Base(path))

		hexAddress := addToMerkle(parentHex, myHex)

		hexedOwner := hashAndHex(fmt.Sprintf("o%s%s", hexAddress, hashAndHex(rnsRes.Names.Value)))

		req := filetreeTypes.QueryFileRequest{
			Address:      hexAddress,
			OwnerAddress: hexedOwner,
		}

		res, err := qc.Files(context.Background(), &req)
		if err != nil {
			log.Warn().Msg(fmt.Errorf("cannot find file on jackal %w", err).Error())
			w.WriteHeader(500)
			return
		}

		var contents ContentResponse

		err = json.Unmarshal([]byte(res.Files.Contents), &contents)
		if err != nil {
			log.Warn().Msg(fmt.Errorf("cannot unmarshal file %w", err).Error())
			w.WriteHeader(500)
			return
		}

		fids := contents.Fids
		fid := fids[0]

		err = downloadFile(sqc, nil, nil, filepath.Base(path), fid, w, false, filepath.Base(path))
		if err != nil {
			log.Warn().Msg(fmt.Errorf("cannot download file %w", err).Error())
			w.WriteHeader(500)
			return
		}
	}
}

func fidFile(ctx client.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fid := vars["fid"]

		if len(fid) < 1 {
			log.Warn().Msg("needs to supply fid")
			w.WriteHeader(400)
			return
		}

		qc := storageTypes.NewQueryClient(ctx)

		err := downloadFile(qc, nil, nil, fid, fid, w, false, "")
		if err != nil {
			log.Warn().Err(err)
			w.WriteHeader(500)
			return
		}
	}
}

func initRouter(ctx client.Context, db *badger.DB) http.Handler {
	router := mux.NewRouter()
	router.SkipClean(true)

	router.HandleFunc("/f/{fid}", fidFile(ctx))

	router.HandleFunc(`/p/{owner}/{path:.+}`, downloadByPath(ctx, db, false, false))
	router.HandleFunc(`/www/{owner}/{path:.+}`, downloadByPath(ctx, db, false, true))
	router.HandleFunc(`/md/{owner}/{path:.+}`, downloadByPath(ctx, db, true, false))

	router.HandleFunc(`/{rns}/{path:.+}`, rnsSite(ctx))
	router.HandleFunc(`/{rns}`, rnsSite(ctx))

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := "Shepherd Gateway"
		_, _ = w.Write([]byte(s))
	})

	handler := cors.Default().Handler(router)

	return handler
}

func startServer(ctx client.Context, db *badger.DB, rpc string, port int64) {
	handler := initRouter(ctx, db)

	log.Info().Msgf("🌍 Started Shepherd: http://0.0.0.0:%d using %s", port, rpc)
	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), handler)
	if err != nil {
		log.Error().Err(err)
		return
	}

	if errors.Is(err, http.ErrServerClosed) {
		log.Info().Msg("Storage Provider Closed")
		return
	} else if err != nil {
		log.Error().Err(fmt.Errorf("error starting server: %w", err))
		os.Exit(1)
	}
}
