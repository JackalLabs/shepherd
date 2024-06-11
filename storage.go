package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"github.com/rs/zerolog/log"
)
import _ "embed"

//go:embed mime.json
var mimes []byte

var mimeTypes map[string]string

func init() {
	err := json.Unmarshal(mimes, &mimeTypes)
	if err != nil {
		log.Error().Err(err)
	}
}

func downloadFileFromURL(url string, fileName string, fid string, writer http.ResponseWriter, isMarkdown bool, title string) ([]byte, error) {
	// Get the data

	u := fmt.Sprintf("%s/download/%s", url, fid)

	client := http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"User-Agent":                {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
		"Upgrade-Insecure-Requests": {"1"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		//"Accept-Encoding":           {"gzip, deflate, br"},
		"Accept-Language": {"en-US,en;q=0.8"},
		"Connection":      {"keep-alive"},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if isMarkdown {
		html := mdToHTML(bodyBytes, title)
		tmp := make([]byte, len(html))
		copy(tmp, html)
		_, _ = writer.Write(html)
		return tmp, nil
	}

	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("file cannot be empty")
	}

	ext := filepath.Ext(fileName)
	if len(ext) > 0 {
		mime := mimeTypes[ext]
		if len(mime) > 0 {
			writer.Header().Set("Content-Type", mime)
		}
	}

	tmp := make([]byte, len(bodyBytes))
	copy(tmp, bodyBytes)

	fmt.Printf("%s real %d\n", fileName, len(bodyBytes))

	_, err = writer.Write(bodyBytes)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%s temp %d\n", fileName, len(tmp))

	return tmp, nil
}

func downloadFile(qc storageTypes.QueryClient, db *badger.DB, key []byte, fileName string, fid string, writer http.ResponseWriter, isMarkdown bool, title string) error {
	req := &storageTypes.QueryFindFileRequest{
		Fid: fid,
	}

	providers, err := qc.FindFile(context.Background(), req)
	if err != nil {
		return err
	}

	ips := providers.ProviderIps
	var arr []string
	err = json.Unmarshal([]byte(ips), &arr)
	if err != nil {
		return err
	}

	rand.Shuffle(len(arr), func(i, j int) { // randomize provider order
		arr[i], arr[j] = arr[j], arr[i]
	})

	failed := true
	var fd []byte
	for _, s := range arr {
		fileData, err := downloadFileFromURL(s, fileName, fid, writer, isMarkdown, title)
		fmt.Printf("%s returned %d\n", fileName, len(fileData))
		if err == nil {
			fd = make([]byte, len(fileData))
			copy(fd, fileData)
			failed = false
			break
		}

	}

	if failed {
		return fmt.Errorf("failed to download any files")
	}

	fmt.Printf("%s inserted %d\n", fileName, len(fd))

	if db != nil {
		_ = db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, fd)
		})
	}

	return nil
}
