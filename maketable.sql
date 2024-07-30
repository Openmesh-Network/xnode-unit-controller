CREATE TABLE IF NOT EXISTS sponsors (
	sponsor_id SERIAL PRIMARY KEY,
	api_key VARCHAR(200) NOT NULL,
	credit_initial NUMERIC(11, 2) NOT NULL,
	credit_spent NUMERIC(11, 2) NOT NULL DEFAULT 0,
	enabled BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS deployments (
	id SERIAL PRIMARY KEY,
	sponsor_id INT,
	FOREIGN KEY (sponsor_id) REFERENCES sponsors(sponsor_id),
	provider VARCHAR(100),
	nft VARCHAR(100),
	instance_id VARCHAR(200),
	activation_date DATE
);

/* Get the ratio for all sponsors. */
SELECT CAST(credit_spent AS FLOAT) / CAST(credit_initial AS FLOAT)
FROM sponsors;

/* Get the sponsor with the lowest ratio that still has enough money to pay for one machine for a year. */
/*
SELECT sponsor_id, api_key, (CAST(credit_spent AS FLOAT) / CAST(credit_initial AS FLOAT)) AS ratio
FROM sponsors
WHERE credit_initial - credit_spent > (9.15 * 12) AND enabled
ORDER BY ratio ASC;
*/

/* 
UPDATE sponsors SET credit_spent = credit_spent + $1 WHERE sponsor_id = $2;
*/

/*
*/
