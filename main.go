package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
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

func rowToDeployment(row *sql.Row, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return row.Scan(&deployment.id, &deployment.sponsor_id, &deployment.nft, &deployment.provider, &deployment.instance_id, &deployment.activation_date)
}

func rowsToDeployment(rows *sql.Rows, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return rows.Scan(&deployment.id, &deployment.sponsor_id, &deployment.nft, &deployment.provider, &deployment.instance_id, &deployment.activation_date)
}

var vpsCostMonthly = 9.15
var vpsCostyearly = math.Ceil((vpsCostMonthly * 12))

func provision(db *sql.DB, nftId, xnodeId, xnodeAccessToken, xnodeConfigRemote string, timeNFTMinted time.Time) (ServerInfo, error) {
	// NOTE: For now we assume all NFTs ids are valid.
	// We trust DPL to only give reliable data.

	// XXX: Add independent verification of NFT.
	fmt.Println(nftId)

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
				return ServerInfo{}, errors.New("Error couldn't find viable sponsor: " + err.Error())
			} else {
				// XXX: Untested.

				info, err := hivelocityApiProvision(apiKey, xnodeId, xnodeAccessToken, xnodeConfigRemote)

				if err != nil {
					return ServerInfo{}, err
				}

				// Calculate the yearly cost

				_, err = db.Exec("INSERT INTO deployments (nft, sponsor_id, provider, instance_id, activation_date) VALUES ($1, $2, $3, $4, $5)",
					nftId, sponsorId, "hivelocity", info.id, timeNFTMinted)

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
		panic("Couldn't load env variables. Is .env not defined?")
	}

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
			panic("Failed to connect to database.")
		}
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
			}
			rows.Close()
		}
	}

	r := gin.Default()
	// XXX: This might cause connection problems on railway, double check that doesn't happen.
	r.SetTrustedProxies(nil)

	r.POST("/v1/provision/:nftid", func(c *gin.Context) {
		// TODO: Actually parse these.
		nftId := c.Param("nftid")

		xnodeId := c.GetString("xnodeId")
		xnodeAccessToken := c.GetString("xnodeAccessToken")
		xnodeConfigRemote := c.GetString("xnodeConfigRemote")
		nftActivationTime := c.GetTime("nftActivationTime")

		info, err := provision(db, nftId, xnodeId, xnodeAccessToken, xnodeConfigRemote, nftActivationTime)

		if err != nil {
			fmt.Println("Error in provision request.")
			fmt.Println(err.Error())
			c.JSON(400, map[string]string{
				"message": err.Error(),
			})
		} else {
			fmt.Println(info)
			c.JSON(200, map[string]interface{}{
				"id":        info.id,
				"ipAddress": info.ipAddress,
			})
		}
	})

	r.POST("/v1/info/:nftid", func(c *gin.Context) {

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
					"id":        info.id,
					"ipAddress": info.ipAddress,
				})
			}
		}
	})

	r.Run(":8080")
}
