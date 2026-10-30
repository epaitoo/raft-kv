package raft

type Transport interface {
	AppendEntriesRPC(nodeId int64, args *AppendEntriesArgs, reply *AppendEntriesReply) error
	RequestVoteRPC(nodeId int64, args *RequestVoteArgs, reply *RequestVoteReply) error
}
