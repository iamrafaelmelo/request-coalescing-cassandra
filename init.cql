CREATE KEYSPACE IF NOT EXISTS coalescing
WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'dc1': 3
};

USE coalescing;

CREATE TABLE IF NOT EXISTS messages (
    id text PRIMARY KEY,
    content text
);

INSERT INTO messages (id, content)
VALUES ('message-1', 'Hello from Cassandra cluster!');
