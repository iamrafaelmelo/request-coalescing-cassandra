package cassandra

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gocql/gocql"
)

type Repository struct {
	session    *gocql.Session
	queryCount atomic.Int64
}

func NewRepository() (*Repository, error) {
	hosts := os.Getenv("CASSANDRA_HOSTS")

	if hosts == "" {
		hosts = "cassandra-1,cassandra-2,cassandra-3"
	}

	cluster := gocql.NewCluster(
		strings.Split(hosts, ",")...,
	)

	numConns, _ := strconv.Atoi(os.Getenv("CASSANDRA_NUM_CONNS"))
	if numConns == 0 {
		numConns = 10
	}

	cluster.Keyspace = "coalescing"
	cluster.Consistency = gocql.Quorum // requires majority os replica nodes to ack
	cluster.NumConns = numConns

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	return &Repository{
		session: session,
	}, nil
}

func (r *Repository) GetMessage(id string) (string, string, error) {
	r.queryCount.Add(1)

	// time.Sleep(rand.N(500 * time.Second)) // delay artificial

	var content string

	iter := r.session.Query(`
		SELECT content
		FROM messages
		WHERE id = ?
	`, id).Iter()

	if !iter.Scan(&content) {
		return "", "", gocql.ErrNotFound
	}

	// get the host of node that take and executed the operation
	host := iter.Host()

	if err := iter.Close(); err != nil {
		return "", "", err
	}

	return content, host.ConnectAddress().String(), nil
}

func (r *Repository) QueryCount() int64 {
	return r.queryCount.Load()
}

func (r *Repository) Close() {
	r.session.Close()
}
