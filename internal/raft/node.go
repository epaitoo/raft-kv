package raft

import (
	"sync"
	"time"
)

type Role int

const (
	Follower Role = iota + 1
	Candidate
	Leader
)

type Node struct {
	id              int64
	currentTerm     int64
	votedFor        int64
	log             []LogEntry
	commitIndex     int
	lastApplied     int
	nextIndex       map[int64]int
	matchIndex      map[int64]int
	mu              sync.Mutex
	peers           []int64
	transport       Transport
	role            Role
	lastResetTime   time.Time
	electionTimeout time.Duration
}

func (n *Node) AppendEntriesRPC(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	return nil
}

func (n *Node) RequestVoteRPC(args *RequestVoteArgs, reply *RequestVoteReply) error {
	return nil
}
