#!/bin/bash
# Initialize MongoDB replica set for Change Streams support

# Wait for MongoDB to be ready
sleep 5

# Initialize replica set
mongosh --eval '
rs.initiate({
  _id: "rs0",
  members: [
    { _id: 0, host: "localhost:27017" }
  ]
})
'

echo "MongoDB replica set initialized"
