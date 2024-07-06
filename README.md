# Xnode Unit Controller This program talks directly with hivelocity to provision Xnode Units / Xnode Ones.
Rewritten due to change in constraints.

## TODO
- [X] Parse env vars to get database info.
- [O] Add provision endpoint.
    - [X] Database logic.
    - [X] Add hivelocity provisioning API.
    - [X] Add hivelocity reset API.
        - [X] Shutdown
        - [X] Wait
        - [X] Provision
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
- [ ] Integrate into DPL.
- [ ] Add info endpoint.
    - Just parse the hivelocity info endpoint.
    - Why exactly is this needed? I'll integrate the dpl first and then see what's happened.
- [ ] Robust error handling.
    - [ ] CTRL+F panic and replace (or keep)
- [ ] Generic cloud-init options converter.
    - Turn arguments to proc cmdline things.
- [ ] Clean up TODOs.
- [ ] Generate table if not provided.
    - Use the sql file?
- [ ] What does SSL mode do? Might have to enable it for railway?
- [X] Add new table for sponsors
    - [X] Api keys
    - [X] Associate each deployment with a sponsor id
- [ ] Associate metadata with each request?
    - Can use **tags** for this in hivelocity API.
        - [ ] Ask hivelocity maximum size for tags in API.
    - Could come in handy if we have to trace machines.
    - Data to store:
        - Xnode UUID
        - Xnode Controller ID
        - Sponsor ID
        - NFT ID
- [ ] Add constrains to prevent data entry errors?
    - [ ] Make sure there can be no newlines on the strings.
    - [ ] Initial credit amounts have to be in the
        - 50k, 100k, 200k, 500k
- [ ] Include request information on logs.

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
            
## Nice to have
- [ ] Add sample env.

# New Strategy for sponsor accounts
I can get the credit from the account.
Then I can reserve the maximum number of servers (Multiplying by 11.5 months OR 12 months), setting the NFT field to NULL.
After that I have one database with all the sponsors.

Alternative plan:
- Steps 1, 2 for provision endpoint stay the same
- Only provision servers when requested, with same logic as before.
- We go sequentially through the providers and just go by order added.
    1. Check credit.
    2. If credit can afford a server for 1 year, provision a server (steps 2, or 3).
    3. If we're out of credit, trip a flag that marks the account as empty.
This needs to be unfuckable!!!!

- [ ] Add SQL transactions for race conditions!!!
- [ ] Might have to do round robbin, but how do I keep it asynchronous?
    - Could do best effort?
        - We prioritize the sponsor with the lowest credit / deployment ratio first.
        - OR we prioritize lowest spent credits first? (But ideally we'd have a ratio, yes?)

- [ ] Does the credit go down right away? Or is it only for show.
    - Harry says it does. Will have to test myself though.
    - Also might have to add extra padding just in case the timing doesn't work out.
    - Doesn't matter lol. Well, kind of does but don't really need it.

Might have to do round robbin of accounts for this to make sense.

**When I reset a server I treat the credits the same way I do provisioning? (ONLY ON CASE 2).**

What I need is total / initial credits. And then reserved credits?
- Should this be put in manually somehow?
- How is this put into the database?
    - Have a simple website / script that generates all this stuff and double checks?
        - Write API key down for sponsor.
        - Checks it's correct.
        - Sets the credit amount.
    - Have an API on the xu controller that does this?
        - Would have to expose to internet. NOT GOOD!
        - Make a script that does this? But isn't connected to the database?
            - Have a boolean for enabled?
            - Called a bit in SQL, make it default to 0 (false)
    - 

Still need an internal credits tracker for credits.
- Have initial + stored.

Safety measures:
- [ ] Ability to manually disable accounts from provisioning new servers.
- [ ] .

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
