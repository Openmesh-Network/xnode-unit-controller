package main

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

const baseUrl = "http://127.0.0.1:8080"

func TestInfo(t *testing.T) {
	nftId := "66638962393212801359315401625300803155691041113216855832713493800930215134027"
	requestUrl := fmt.Sprintf("%s/info/%s", baseUrl, nftId)

	t.Log(requestUrl)

	resp, err := http.Get(requestUrl)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp.StatusCode)
	respBody := readall(resp.Body)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(string(respBody))
	}

}

func TestInfoNumeric(t *testing.T) {
	numericNftId := 123
	requestUrl := fmt.Sprintf("%s/info/%d", baseUrl, numericNftId)
	resp, err := http.Get(requestUrl)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(resp.StatusCode)
		respBody := readall(resp.Body)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(string(respBody))
		}
	}
}

func TestProvision(t *testing.T) {
	nftId := "66638962393212801359315401625300803155691041113216855832713493800930215134027"
	requestUrl := fmt.Sprintf("%s/provision/%s", baseUrl, nftId)
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
		respBody := readall(resp.Body)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(string(respBody))
		}
	}
}
