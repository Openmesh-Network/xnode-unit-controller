package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

const baseUrl = "http://127.0.0.1:8080"

func TestInfo(t *testing.T) {
	nftId := "66638962393212801359315401625300803155691041113216855832713493800930215134027"
	genericInfoTest(t, nftId)

	nftId = "78754463791077132556305196583841820451073308244165952054046138079426471331474"
	genericInfoTest(t, nftId)
}

func genericInfoTest(t *testing.T, nftId string) {
	requestUrl := fmt.Sprintf("%s/info/%s", baseUrl, nftId)

	t.Log(requestUrl)

	resp, err := http.Get(requestUrl)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp.StatusCode)
	respBody := readall(resp.Body)

	t.Log(string(respBody))
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

		t.Log(string(respBody))
	}
}

func TestProvision(t *testing.T) {
	nftId := "666"
	genericProvision(t, nftId)

	nftId = "66638962393212801359315401625300803155691041113216855832713493800930215134027"
	genericProvision(t, nftId)

}

func TestMockProvision(t *testing.T) {
	genericProvision(t, "345")
}

func genericProvision(t *testing.T, nftId string) {
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
		t.Log(string(respBody))
	}
}

func TestAvailability(t *testing.T) {
	// Unit test for hivelocityAvailableRegions
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Couldn't load env variables. Is .env not defined?")
	}
	apiKey := os.Getenv("HVE_API_KEY")
	productId := "2379"
	t.Log("Using key", apiKey, "finding inventory for", productId)

	availableRegion, capacityError := hivelocityAvailableRegions(apiKey, productId)
	if capacityError != nil {
		t.Fatal(capacityError)
	} else {
		t.Log("Found availability at", availableRegion)
	}
}
