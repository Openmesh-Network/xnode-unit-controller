package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Deployment struct {
	id              sql.NullInt32
	nft             sql.NullString
	provider        sql.NullString
	instance_id     sql.NullString
	activation_date sql.NullTime
}

func rowToDeployment(row *sql.Row, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return row.Scan(&deployment.id, &deployment.nft, &deployment.provider, &deployment.instance_id, &deployment.activation_date)
}

func rowsToDeployment(rows *sql.Rows, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return rows.Scan(&deployment.id, &deployment.nft, &deployment.provider, &deployment.instance_id, &deployment.activation_date)
}

func hivelocityApiProvision(hveApiKey string) {

	body, err := json.Marshal(map[string]interface{}{
		"osName": "Ubuntu 22.04 (VPS)",
        "period":"monthly",
        // XXX: Might have to change region depending on settings.
        "locationName":"NYC1",
        // "productId": "2313", // Subject to future change (2311 for testing)

        // XXX: Change this to custom product id.
        "productId": "2311", // Subject to future change (2311 for testing)
        "hostname": "xnode.openmesh.network",

        // XXX: This has to be updated to latest boot command from dpl.
        "script": "",
	})

	req, err := http.NewRequest("POST", "https://core.hivelocity.net/api/v2/compute/", bytes.NewBuffer(body))

	if err != nil {
		panic(err)
	}

	req.Header.Add("X-API-KEY", hveApiKey)
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		panic(err)
	} else {
		val, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		fmt.Printf("res.Body: %v\n", val)
	}
}

func hivelocityApiReset() {
}
func hivelocityApiInfo() {
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Couldn't load env variables. Is .env not defined?")
	}

	// TODO: Re-enable later
	// hivelocity_api_key := os.Getenv("HVE_API_KEY")

	connectString := fmt.Sprintf(
		"user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"))

	db, err := sql.Open("postgres", connectString)
	defer db.Close()

	if err != nil {
		panic(err)
	} else {
		err := db.Ping()
		if err == nil {
			fmt.Println("Successfully connected to database.")
		} else {
			fmt.Println("Failed to connect to database.")
		}
	}

	{ // XXX: Delete this, it's just for reference
		rows, err := db.Query("SELECT * FROM deployments")
		defer rows.Close()

		if err != nil {
			panic(err)
		} else {

			fmt.Println("Looking through rows.")
			for rows.Next() {
				fmt.Println("Ran once?")
				var deployment Deployment

				rowsToDeployment(rows, &deployment)

				fmt.Println(deployment.provider)
			}

			rows.Close()
		}
	}

	r := gin.Default()

	r.POST("/v1/provision/:nftid", func(c *gin.Context) {

		fmt.Println("This ran?")

		// NOTE: For now we assume all NFTs ids are valid.
		// We trust DPL to only give reliable data.

		nftid := c.Param("nftid")

		// TODO: Add independent verification of NFT.
		fmt.Println(nftid)

		row := db.QueryRow("SELECT * FROM deployments WHERE nft = $1", nftid)
		err := rowToDeployment(row, nil)

		if err == nil {
			// 1st. Is the NFT already in the database?
			fmt.Println("Matched NFT")

			// TODO: Reset hivelocity vps.
			fmt.Println("Already deployed, resetting machine.")
		} else {
			if err == sql.ErrNoRows {
				fmt.Println("No such NFT.")

				row := db.QueryRow("SELECT * FROM deployments WHERE nft IS NULL ORDER BY id")
				deployment := Deployment{}
				err := rowToDeployment(row, &deployment)

				if err == nil {
					// 2nd. Is there an empty slot?

					// TODO: Reset this machine.

					fmt.Println("Resetting existing machine and updating database.")

					// TODO: Update matching row.
					_, err := db.Exec("WITH first_available AS ( SELECT id, nft FROM deployments WHERE nft IS NULL ORDER BY id LIMIT 1 ) UPDATE deployments SET nft = $1 FROM first_available WHERE deployments.id = first_available.id", nftid)

					if err != nil {
						fmt.Println("Error updating vps in database:", err.Error())
					} else {
						fmt.Println("Updated unused vps in database.")
					}
				} else {
					// TODO: Provision new machine.

					// TODO: Create row in database.
					fmt.Println("Provisioning new machine and creating row in database.")

					_, err := db.Exec("INSERT INTO deployments (nft, provider, instance_id, activation_date) VALUES ($1, $2, $3, $4)",
						nftid, "hivelocity", "placeholder", time.Now())

					if err != nil {
						fmt.Println("Error adding new vps to database:", err.Error())
					} else {
						fmt.Println("Added new vps to database.")
					}

				}
			} else {
				fmt.Println("Error: running select from database.", err.Error())
			}
		}

		c.JSON(200, gin.H{})
	})

	r.Run(":8080")
}
