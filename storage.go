package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

func downloadFileFromURL(url string, fid string, writer io.Writer) error {
	// Get the data
	resp, err := http.Get(fmt.Sprintf("%s/download/%s", url, fid))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	fmt.Println("got file, forwarding file...")

	// Writer the body to writer
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func downloadFile(qc storageTypes.QueryClient, fid string, writer io.Writer) error {
	req := &storageTypes.QueryFindFileRequest{
		Fid: fid,
	}

	providers, err := qc.FindFile(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Println("found file...")

	ips := providers.ProviderIps
	var arr []string
	err = json.Unmarshal([]byte(ips), &arr)
	if err != nil {
		return err
	}

	fmt.Println("attempting to download file...")

	failed := true
	for _, s := range arr {
		err := downloadFileFromURL(s, fid, writer)
		if err == nil {
			failed = false
			break
		}
		fmt.Println("failed to download file, will try again...")
	}

	if failed {
		return fmt.Errorf("failed to download any files")
	}

	fmt.Println("complete")
	return nil
}
