package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
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
	nftId := 123
	baseUrl := "http://127.0.0.1:8080"
	requestUrl := fmt.Sprintf("%s/provision/%d", baseUrl, nftId)
	bodyData := fmt.Sprintf(`{"xnodeId": "test", "xnodeAccessToken": "test", "xnodeConfigRemote": "test", "nftActivationTime": "%s"}`, "2024-06-19T02:51:48.000Z")
	requestBody := []byte(bodyData)
	t.Log(bodyData)
	resp, err := http.Post(requestUrl, "json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatal(err)
	} else {
		if resp.StatusCode != 200 {
			t.Fatal(resp.StatusCode)
		}
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(string(respBody))
		}
	}
	t.Log(time.Now().Unix())
}
