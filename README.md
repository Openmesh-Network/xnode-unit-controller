# Xnode Unit Controller
This program manages a postgres database with deployments for Xnodes


## TODO
- [X] Parse env vars to get database info.
- [o] Add provision endpoint.
    - [X] Database logic.
    - [X] Add hivelocity provisioning API.
    - [X] Add hivelocity reset API.
    - [ ] Test the APIs work.
    - [ ] Test if `forceReload` has to be excluded on provision requests.
- [X] Add cloud-init script to both requests.
- [ ] Add info endpoint.
    - Just parse the hivelocity info endpoint.
- [ ] Robust error handling.
- [ ] Clean up TODOs
- [ ] Generate table if not provided.
- [ ] Integrate into DPL.
- [ ] What does SSL mode do? Might have to enable it for railway?

## Nice to have
- [ ] Add sample env
- [ ] Add unique constraint to NFT in database

# Notes
- Gock for request mocking? 
https://medium.com/zus-health/mocking-outbound-http-requests-in-go-youre-probably-doing-it-wrong-60373a38d2aa

- 
