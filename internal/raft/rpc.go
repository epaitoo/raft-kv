package raft

type RequestVoteArgs struct {
	Term         int64
	CandidateId  int64
	LastLogIndex int
	LastLogTerm  int64
}

type RequestVoteReply struct {
	Term        int64
	VoteGranted bool
}

type LogEntry struct {
	Term    int64
	Command any
}

type AppendEntriesArgs struct {
	Term         int64
	LeaderId     int64
	PrevLogIndex int
	PrevLogTerm  int64
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int64
	Success bool
}
