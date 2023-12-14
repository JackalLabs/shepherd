package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

func downloadFileFromURL(url string, fid string, writer io.Writer, isMarkdown bool, title string) error {
	// Get the data

	client := http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/download/%s", url, fid), nil)
	if err != nil {
		return err
	}

	req.Header = http.Header{
		"User-Agent":                {"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36"},
		"Upgrade-Insecure-Requests": {"1"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Encoding":           {"gzip, deflate, br"},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Connection":                {"keep-alive"},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	if isMarkdown {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		html := mdToHTML(bodyBytes, title)
		writer.Write(html)
		return nil
	}

	// Writer the body to writer
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func downloadFile(qc storageTypes.QueryClient, fid string, writer io.Writer, isMarkdown bool, title string) error {
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
	for _, s := range arr {
		err := downloadFileFromURL(s, fid, writer, isMarkdown, title)
		if err == nil {
			failed = false
			break
		}
	}

	if failed {
		return fmt.Errorf("failed to download any files")
	}

	return nil
}
