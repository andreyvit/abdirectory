package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var client = &http.Client{
	Timeout: 30 * time.Second,
}

func download(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	log.Printf("%v %v", req.Method, req.URL.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: network: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GET %s: reading body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: HTTP %s: %s", resp.Status, body)
	}

	return body, nil
}
