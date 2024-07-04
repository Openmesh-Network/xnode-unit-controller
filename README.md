# Xnode Unit Controller
This program manages a postgres database with deployments for Xnodes


## TODO
- [X] Parse env vars to get database info.
- [o] Add provision endpoint.
    - [X] Database logic.
    - [ ] Add hivelocity provisining API.
- [ ] Add info endpoint.
    - Just parse the hivelocity info endpoint.
- [ ] Add hivelocity reset API.
- [ ] Add cloud-init script to both requests.
- [ ] Generate table if not provided.
- [ ] Integrate into DPL.
- [ ] What does SSL mode do? Might have to enable on railway?

## Nice to have
- [ ] Add sample env
- [ ] Add unique constraint to NFT in database
