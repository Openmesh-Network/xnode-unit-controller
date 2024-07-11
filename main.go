package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Deployment struct {
	id              int
	nft             sql.NullString
	sponsor_id      int
	provider        string
	instance_id     sql.NullString
	activation_date sql.NullTime
}

type ProvisionRequestBody struct {
	XnodeId           string `json:"xnodeId"`
	XnodeAccessToken  string `json:"xnodeAccessToken"`
	XnodeConfigRemote string `json:"xnodeConfigRemote"`
	NftActivationTime string `json:"nftActivationTime"`
}

func parseProvisionReq(provisionReq ProvisionRequestBody) (string, string, string, time.Time) {
	activationTime, err := time.Parse(time.RFC3339, provisionReq.NftActivationTime)
	if err != nil {
		fmt.Println("Unable to parse nft activation time.", provisionReq.NftActivationTime)
	} else {
		fmt.Println(activationTime)
	}
	return provisionReq.XnodeId, provisionReq.XnodeAccessToken, provisionReq.XnodeConfigRemote, activationTime
}

func rowToDeployment(row *sql.Row, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return row.Scan(&deployment.id, &deployment.sponsor_id, &deployment.provider, &deployment.nft, &deployment.instance_id, &deployment.activation_date)
}

func rowsToDeployment(rows *sql.Rows, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return rows.Scan(&deployment.id, &deployment.sponsor_id, &deployment.provider, &deployment.nft, &deployment.instance_id, &deployment.activation_date)
}

var vpsCostMonthly = 9.15
var vpsCostyearly = math.Ceil((vpsCostMonthly * 12))

func provision(db *sql.DB, nftId string, xnodeId string, xnodeAccessToken string, xnodeConfigRemote string, timeNFTMinted time.Time) (ServerInfo, error) {
	// NOTE: For now we assume all NFTs ids are valid.
	// We trust DPL to only give reliable data.
	// XXX: Add independent verification of NFT.
	// verifyNftExists(nftId) - NFT-Authorise Library

	row := db.QueryRow("SELECT * FROM deployments WHERE nft = $1", nftId)
	deployment := Deployment{}
	err := rowToDeployment(row, &deployment)

	if err == nil {
		if !deployment.instance_id.Valid {
			fmt.Println("No instance for deployment id: ", deployment.id)
			panic("")
		}

		// Get the api key from the sponsor!
		row := db.QueryRow("SELECT api_key FROM deployments NATURAL JOIN sponsors WHERE id = $1", deployment.id)
		api_key := ""
		err := row.Scan(&api_key)
		if err != nil {
			panic("No sponsor for existing deployment.")
		}

		fmt.Println("Found NFT in deployments table.")

		fmt.Println("Resetting...")

		info, err := hivelocityApiReset(api_key, deployment.instance_id.String, xnodeId, xnodeAccessToken, xnodeConfigRemote)

		if err != nil {
			return ServerInfo{}, err
		} else {
			return info, nil
		}
	} else {
		if err == sql.ErrNoRows {
			fmt.Println("Didn't find NFT in database.")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// XXX: Might have to make this a serializeable transaction to 100% avoid race conditions (Check definition of BeginTx for what args to pass).
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				panic(err)
			}

			defer tx.Commit()

			fmt.Println("Provisioning new machine and creating row in database.")

			projectedCost := vpsCostyearly
			{
				difference := time.Now().Unix() - timeNFTMinted.Unix()
				if difference < 0 {
					panic("NFT minted in the future or clock is out of date.")
				}

				// Assuming 730 hours in a month.
				monthsDifference := math.Floor(float64(difference) / (60 * 60 * 730))
				if monthsDifference > 12 {
					return ServerInfo{}, errors.New("Got expired NFT.")
				}
				// Calculate the yearly cost
				projectedCost = (12.0 - monthsDifference) * vpsCostMonthly
			}

			row := db.QueryRow(
				`SELECT sponsor_id, api_key, (CAST(credit_spent AS FLOAT) / CAST(credit_initial AS FLOAT)) AS ratio
				FROM sponsors
				WHERE credit_initial - credit_spent > ($1)
				ORDER BY ratio ASC;`, projectedCost)

			sponsorId := 0
			ratio := 0.0
			apiKey := ""

			err = row.Scan(&sponsorId, &apiKey, &ratio)

			if err != nil {
				// TODO: Return "Capacity reached, try again later" to the user in frontend.
				return ServerInfo{}, errors.New("Error couldn't find viable sponsor: " + err.Error())
			} else {

				info, err := hivelocityApiProvision(apiKey, xnodeId, xnodeAccessToken, xnodeConfigRemote)
				if err != nil {
					fmt.Println("SKIPPED ADDING TO DB; Error in provisioning: ", err)
					return ServerInfo{}, err
				}

				// Add the new server to the database, should never be skipped. Ensure that the program cannot return before INSERT INTO
				_, err = db.Exec("INSERT INTO deployments (nft, sponsor_id, provider, instance_id, activation_date) VALUES ($1, $2, $3, $4, $5)",
					nftId, sponsorId, "hivelocity", info.Id, timeNFTMinted)

				fmt.Println("Adding new server to database with a projected cost of: ", projectedCost)
				db.Exec("UPDATE sponsors SET credit_spent = credit_spent + $1 WHERE sponsor_id = $2;", projectedCost, sponsorId)

				if err != nil {
					return ServerInfo{}, errors.New("Error adding new vps to the database: " + err.Error())
				} else {
					fmt.Println("Added new vps to database.")
					return info, nil
				}
			}
		} else {
			return ServerInfo{}, errors.New("Error: running select from database: " + err.Error())
		}
	}
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Couldn't load env variables. Is .env not defined?")
	}
	user := os.Getenv("DB_USER")
	dbName := os.Getenv("DB_NAME")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := ""
	if os.Getenv("DB_PORT") != "" {
		dbPort = os.Getenv("DB_PORT")
	}
	dbDriver := "postgres"
	if os.Getenv("DB_DRIVER") != "" {
		dbDriver = os.Getenv("DB_DRIVER")
	}

	connectString := fmt.Sprintf(
		"user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
		user, dbName, dbPass, dbHost, dbPort)

	db, dbErr := sql.Open(dbDriver, connectString)
	if dbErr != nil {
		panic(dbErr)
	}
	defer db.Close()
	ping := db.Ping()
	if ping == nil {
		fmt.Println("Successfully connected to database.")
	} else {
		panic("Failed to connect to database.")
	}

	{ // XXX: Delete this, it's just for reference.
		rows, err := db.Query("SELECT * FROM deployments")
		defer rows.Close()

		if err != nil {
			panic(err)
		} else {
			fmt.Println("Looking through rows.")
			for rows.Next() {
				var deployment Deployment

				rowsToDeployment(rows, &deployment)
				fmt.Println("Entries", deployment.nft)
			}
			rows.Close()
		}
	}

	r := gin.Default()
	// XXX: This might cause connection problems on railway, double check that doesn't happen.
	r.SetTrustedProxies(nil)

	r.POST("/provision/:nftid", func(c *gin.Context) {
		nftId := c.Param("nftid")

		var requestBody ProvisionRequestBody
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"message": "Invalid request",
			})
			return
		}
		// TODO: Sanity check on data parsed to requestBody
		uuid, psk, remote, activationTime := parseProvisionReq(requestBody)
		info, err := provision(db, nftId, uuid, psk, remote, activationTime)

		if err != nil {
			fmt.Println("Error in provision request.")
			fmt.Println(err.Error())
			c.JSON(400, map[string]string{
				"message": err.Error(),
			})
		} else {
			fmt.Println(info)
			c.JSON(200, map[string]interface{}{
				"id":        info.Id,
				"ipAddress": info.IpAddress,
			})
		}
	})

	r.GET("/info/:nftid", func(c *gin.Context) {
		nftId := c.Param("nftid")

		row := db.QueryRow("SELECT api_key, instance_id FROM deployments NATURAL JOIN sponsors WHERE nft = $1", nftId)

		apiKey := ""
		instanceId := ""
		err := row.Scan(&apiKey, &instanceId)
		if err != nil {
			fmt.Println("Info request failed. Error: ", err.Error())
			c.JSON(400, "No such NFT.")
		} else {
			info, err := hivelocityApiInfo(apiKey, instanceId)
			if err != nil {
				c.JSON(400, "Failed to find hivelocity.")
			} else {
				c.JSON(200, map[string]interface{}{
					"id":        info.Id,
					"ipAddress": info.IpAddress,
				})
			}
		}
	})

	r.Run(":8080")
}
