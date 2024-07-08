package main

import (
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestInfo(t *testing.T) {
	nftId := 0123
	requestUrl := "http://127.0.0.1:8080"
	location := fmt.Sprintf("%s/info/%d", requestUrl, nftId)
	resp, err := http.Get(location)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(resp.StatusCode)
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(string(respBody))
		}
	}
}

func TestProvision(t *testing.T) {
	nftId := 0123
	requestUrl := "http://127.0.0.1:8080"
	location := fmt.Sprintf("%s/provision/%d", requestUrl, nftId)
	resp, err := http.Post(location, "json", nil)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(resp.StatusCode)
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(string(respBody))
		}
	}
}
