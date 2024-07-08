# Xnode Unit Controller This program talks directly with hivelocity to provision Xnode Units / Xnode Ones.
Rewritten due to change in constraints.


## How it works roughly:

There's two tables:
1. sponsors
    - Api key for sponsor
    - Initial credit
    - Credit spent
2. deployment
    - Deployment id (Just a number)
    - Nft
    - Sponsor id
    - Provider (Just "hivelocity" for now)
    - Instance id (The deviceId in hivelocity API for reset)
    - Activation date (The date the machine was provisioned, not nft redeem time)

On provision request the server checks if the nft is in the deployment table.
If it is, then it will get the API key from the sponsor table (By doing a join on the ID), and reset the machine (Requires shutting down and other stuff unfortunately).
If it's not, then it will run the provision request and return the info.

## How to use:

1. Make a new postgres database and create the tables using the `maketable.sql` file as reference.
2. Add login details to env vars, follow .env.sample format.
3. Run `go run .` to start the server.
4. Add at least one sponsor to the sponsor table: 
    1. Set the credits to at least 1000, since each server is about $110 of credit.
    2. Set the API key to an API key you control.

Now you should be able to do:
`curl -X POST http://localhost:8080/provision/<NFT-ID>`

And also:
`curl -X POST http://localhost:8080/info/<NFT-ID>`

For testing, set all the sponsors to the same API key and manually call the endpoint.

Also, there's currently (7-7-2024) an issue with hivelocity where they mark the servers deployed with our product id on their api as "verification" status.
So, until that's fixed we can't really test it completely (Since we need the API to return an ip for the system to work).

## TODO
- [X] Parse env vars to get database info.
- [O] Add provision endpoint.
    - [X] Database logic.
    - [X] Add hivelocity provisioning API.
    - [O] Add hivelocity reset API.
        - [X] Shutdown
        - [X] Wait
        - [o] Provision
            - [X] API
            - [ ] Hivelocity marks the servers as "verification" status for whatever reason .
    - [o] Test the APIs work.
        - [X] Test reset API works.
        - [ ] Set provision api to do round robin over known keys
            - We know that the api for deploy and for provision return the same json anyways, so we should be good.
- [X] Add transactions to avoid race conditions!
- [X] Modify database logic for new tables.
- [X] Retest database logic.
    - [X] Provision new machine.
    - [X] Provision same machine.
- [X] Add cloud-init script to both requests.
- [X] Integrate into DPL.
    - [X] Read parameters from request body.
    - [X] Return expected json.
    - [X] Return errors.
- [X] Add info endpoint.
    - Just parse the hivelocity info endpoint.
    - Why exactly is this needed? I'll integrate the dpl first and then see what's happened.
- [ ] Re-enable the provisioning.
    - [ ] SAD, hivelocity doesn't always provision servers on our account. Have started a chat with them on slack, but can't do anything until they confirm what's going on.
- [ ] Add mock provisioning.
    - [ ] Add an environment variable that "provisions" by resetting a machine from an existing set of machines instead of actually deploying a new vps every time.
    - This would massively simplify testing.
- [X] Estimate cost based on NFT time not just 12 month time.
    - Take NFT activation date (From DPL) and assign enough credit for 12 - months since activation time.
    - Rounded up!
- [X] Robust error handling.
    - [X] API calls should return errors on failure.
    - [X] Check status code of responses.
- [ ] Clean up TODOs.
- [ ] Generate table if not provided.
    - Use the sql file?
- [ ] What does SSL mode do? Might have to enable it for railway?
- [X] Add new table for sponsors
    - [X] Api keys
    - [X] Associate each deployment with a sponsor id
- [X] Associate metadata with each request?
    - Can use **tags** for this in hivelocity API.
        - [X] Ask hivelocity maximum size for tags in API.
    - Could come in handy if we have to trace machines.
    - Data to store:
        - [X] Xnode UUID
        - Xnode Controller ID
        - Sponsor ID
        - NFT ID
- [ ] Add new cloud-init script.
- [ ] Add more constraints to database to prevent entry errors?
    - [ ] Make sure there can be no newlines on the strings.
    - [ ] Credits should be minimum 1k maybe?
- [ ] Improve logs so that they include request information.
    - [ ] Which nft is being targetted at least + XnodeId
## Maybe worth exploring?
- [ ] Generic cloud-init options converter.
    - Turn arguments to proc cmdline things.

## Before launch
- [ ] Might have to enable special account permission on subaccount.
    - "You have not been granted permission to complete this action. Please contact your account manager to grant the proper permission." 
    - https://developers.hivelocity.net/reference/post_compute_resource
- [ ] Make sure the network is set up correctly:
    - Private networking for xu controller.
    - Private networking for postgres database.
- [ ] Make a production version:
    - [ ] Make sure the env vars match to the postgres database.
- [ ] Make sure it plugs into the DPL
- [ ] Ability to manually disable accounts from provisioning new servers. Just add a boolean.
            
## Nice to have
- [X] Add sample env.

## On Security
- Data (Most importantly sponsor API keys) would ONLY be leaked if:
    - Railway is compromised (anyone with admin access can log into the database and view API keys).
    - There's an SQL injection bug in the xu controller where some user can input a SQL command into some parameter and get the result back somehow.
        - Unlikely because the only "user data" passed from frontend it the nft id, and this is has to be verified on ethereum by the DPL.
        - Also unlikely because the go program does proper SQL parameters, we don't just concatenate strings.
        - **ACTUALLY** the info request might have this problem, have to parse NFT in DPL in that case.
    - There's a fuck up and the database and/or xu controller aren't isolated from the public internet.
    - There's an exploit in the DPL that lets people send arbitrary get requests to the xu controller.

# Notes
- Gock for request mocking? 
https://medium.com/zus-health/mocking-outbound-http-requests-in-go-youre-probably-doing-it-wrong-60373a38d2aa

- Check `maketable.sql` for how the schema is created.
