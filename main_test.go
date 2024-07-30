package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

const baseUrl = "http://127.0.0.1:8080"

func TestRoundtrip(t *testing.T) {
	assert := assert.New(t)
	failOnError := func(err error) {
		if err != nil {
			t.Fatal("Failed on error, ", err.Error())
		}
	}

	t.Log("Setting mock env variables.")
	err := godotenv.Load(".env.test")
	if err != nil {
		t.Fatal("Need a .env.test file to load hivelocity API key for testing.")
	}
	assert.Equal("localhost", os.Getenv("DB_HOST"), "Make sure that we're running testing on localhost and not on a remote database! Make sure your env points to a local database and NOT production!!!")

	// Connect to postgres database.
	db, err := sql.Open(connectPostgres())
	failOnError(err)
	defer db.Close()

	// Reset if it's already here.
	t.Log("Resetting database if it's already here.")
	{
		t.Log("Clearing tables")
		_, err = db.Exec("DROP TABLE IF EXISTS deployments")
		failOnError(err)

		_, err := db.Exec("DROP TABLE IF EXISTS sponsors")
		failOnError(err)

		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS sponsors (
				sponsor_id SERIAL PRIMARY KEY,
				api_key VARCHAR(200) NOT NULL,
				credit_initial NUMERIC(11, 2) NOT NULL,
				credit_spent NUMERIC(11, 2) NOT NULL DEFAULT 0,
				enabled BOOLEAN NOT NULL DEFAULT FALSE
			);
		`)
		failOnError(err)

		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS deployments (
				id SERIAL PRIMARY KEY,
				sponsor_id INT,
				FOREIGN KEY (sponsor_id) REFERENCES sponsors(sponsor_id),
				provider VARCHAR(100),
				nft VARCHAR(100),
				instance_id VARCHAR(200),
				activation_date DATE
			);
		`)
		failOnError(err)
	}

	// Test provision without sponsors.

	assertProvision := func(success bool) {

		info, err := provision(db, "", "xunit-test", "", "", time.Now())

		if success {
			assert.NotEqual(info, ServerInfo{})
			assert.Equal(err, nil)
		} else {
			assert.Equal(info, ServerInfo{})
			assert.NotEqual(err, nil)
		}
	}

	t.Log("Testing provisioning with no sponsors.")
	assertProvision(false)

	t.Log("Adding a disabled sponsor with no money.")
	_, err = db.Exec("INSERT INTO sponsors (api_key, credit_initial, credit_spent, enabled) VALUES ('', 10000, 10, FALSE)")
	failOnError(err)
	assertProvision(false)

	t.Log("Adding a sponsor with no money.")
	_, err = db.Exec("INSERT INTO sponsors (api_key, credit_initial, credit_spent, enabled) VALUES ('', 10, 10, TRUE)")
	failOnError(err)
	assertProvision(false)

	t.Log("Adding a viable sponsor with invalid api_key.")
	_, err = db.Exec("INSERT INTO sponsors (sponsor_id, api_key, credit_initial, credit_spent, enabled) VALUES (9999, 'invalid key', 1000, 10, TRUE)")
	failOnError(err)
	assertProvision(false)

	// Get rid of sponsor.
	db.Exec("DELETE FROM sponsors WHERE sponsor_id=9999")

	// Done should test with valid api key now
	api_key := os.Getenv("TEST_API_KEY")

	if api_key == "" {
		t.Log("No api TEST_API_KEY found on .env.test, not checking hivelocity API.")
	} else {
		t.Log("Found api key TEST_API_KEY on .env.test, checking hivelocity API.")

		t.Log("Adding viable sponsor with valid key.")
		_, err = db.Exec("INSERT INTO sponsors (api_key, credit_initial, credit_spent, enabled) VALUES ($1, 1000, 10, TRUE)", api_key)
		failOnError(err)
		assertProvision(true)
	}
}

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
	genericProvision(t, "789")
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
	err := godotenv.Load(".env.test")
	if err != nil {
		fmt.Println("Couldn't load env variables. Is .env.test not defined?")
	}
	apiKey := os.Getenv("TEST_API_KEY")
	productId := "2379"
	t.Log("Using key", apiKey, "finding inventory for", productId)

	availableRegion, capacityError := hivelocityFirstAvailableRegion(apiKey, productId)
	if capacityError != nil {
		t.Fatal(capacityError)
	} else {
		t.Log("Found availability at", availableRegion)
	}
}
