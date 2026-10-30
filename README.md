# Raft KV Store Project

### Definition of Done
- A five-node cluster elects a leader and re-elects within one second of leader failure
- Committed entries are identical across all nodes and survive full cluster restart
- GET, PUT and DELETE work through a client API with correct leader redirection
- The cluster tolerates the loss of a minority of nodes with no data loss and no dual leadership
- The test suite includes partition, crash and restart scenarios and passes a hundred consecutive runs
clean under the race detector
- README, design document and architecture diagram are complete
- Known limitations (no snapshotting, no membership changes) are documented explicitly
- Demo video is recorded and linked from the README
