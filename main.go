package main

import (
	"context"
	"database/sql"
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
	instance_id     sql.NullString
	activation_date sql.NullTime
}

func rowToDeployment(row *sql.Row, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return row.Scan(&deployment.id, &deployment.nft, &deployment.sponsor_id, &deployment.instance_id, &deployment.activation_date)
}

func rowsToDeployment(rows *sql.Rows, deployment *Deployment) error {
	if deployment == nil {
		deployment = &Deployment{}
	}
	return rows.Scan(&deployment.id, &deployment.nft, &deployment.sponsor_id, &deployment.instance_id, &deployment.activation_date)
}

var vps_cost_monthly = 9.15
var vps_cost_yearly = int(math.Ceil((vps_cost_monthly * 12)))

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Couldn't load env variables. Is .env not defined?")
	}

	// TODO: Re-enable later
	hivelocityApiKey := os.Getenv("HVE_API_KEY")

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
				var deployment Deployment

				rowsToDeployment(rows, &deployment)
			}
			rows.Close()
		}
	}

	r := gin.Default()

	r.POST("/v1/provision/:nftid", func(c *gin.Context) {

		fmt.Println("This ran?")

		// NOTE: For now we assume all NFTs ids are valid.
		// We trust DPL to only give reliable data.

		// TODO: Might have to convert date from js format to Go format.

		nftid := c.Param("nftid")
		xnodeId := "testId"
		xnodeAccessToken := "dummyAccessToken"

		// TODO: Add independent verification of NFT.
		fmt.Println(nftid)

		row := db.QueryRow("SELECT * FROM deployments WHERE nft = $1", nftid)
		deployment := Deployment{}
		err := rowToDeployment(row, &deployment)

		if err == nil {
			// 1st. Is the NFT already in the database?

			if !deployment.instance_id.Valid {
				fmt.Println("No instance for deployment id: ", deployment.id)
				panic("")
			}

			fmt.Println("Matched NFT")

			hivelocityApiReset(hivelocityApiKey, deployment.instance_id.String, xnodeId, xnodeAccessToken)
			fmt.Println("Already deployed, resetting machine.")

			// XXX: Does the database need an update at all here? I guess not?
		} else {
			if err == sql.ErrNoRows {
				fmt.Println("No such NFT.")

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// XXX: Might have to give this serializeable option to avoid race conditions (Check definition of BeginTx for what to pass).
				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					panic(err)
				}

				defer tx.Commit()

				row := db.QueryRow("SELECT * FROM deployments WHERE nft IS NULL ORDER BY id")
				deployment := Deployment{}
				err = rowToDeployment(row, &deployment)

				if err == nil {
					// 2nd. Is there an empty slot?
					fmt.Println("Resetting existing machine and updating database.")

					_, err := db.Exec("WITH first_available AS ( SELECT id, nft FROM deployments WHERE nft IS NULL ORDER BY id LIMIT 1 ) UPDATE deployments SET nft = $1 FROM first_available WHERE deployments.id = first_available.id", nftid)

					if err != nil {
						fmt.Println("Error updating vps in database:", err.Error())
					} else {
						fmt.Println("Updated unused vps in database.")
					}
				} else {
					fmt.Println("Provisioning new machine and creating row in database.")

					// TODO: Chose sponsor here.
					row := db.QueryRow(
						`SELECT sponsor_id, api_key, (CAST(credit_spent AS FLOAT) / CAST(credit_initial AS FLOAT)) AS ratio
					FROM sponsors
					WHERE credit_initial - credit_spent > ($1)
					ORDER BY ratio ASC;`, vps_cost_yearly)

					sponsor_id := 0
					ratio := 0.0
					api_key := ""

					err := row.Scan(&sponsor_id, &api_key, &ratio)

					if err != nil {
						fmt.Println("Error couldn't find viable sponsor: ", err.Error())
					} else {
						// XXX: Untested.
						hivelocityApiProvision(api_key, xnodeId, xnodeAccessToken)

						_, err = db.Exec("INSERT INTO deployments (nft, sponsor_id, instance_id, activation_date) VALUES ($1, $2, $3, $4)",
							nftid, sponsor_id, "placeholder", time.Now())

						db.Exec("UPDATE sponsors SET credit_spent = credit_spent + $1 WHERE sponsor_id = $2;", vps_cost_yearly, sponsor_id)

						if err != nil {
							fmt.Println("Error adding new vps to database:", err.Error())
						} else {
							fmt.Println("Added new vps to database.")
						}
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
