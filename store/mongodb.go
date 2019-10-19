package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBStore represents a GridFS based file Repository.
type MongoDBStore struct {
	client *mongo.Client
	bucket *gridfs.Bucket
}

// ErrNoMongoHost is returned if no[[host]:port] combination for MongoDB
// was provided,
var ErrNoMongoHost = errors.New("parameter 'url' is empty")

// NewMongoDB creates a GridFS based repository.
func NewMongoDB(url, user, pass, db, name string) (store *MongoDBStore, err error) {

	store = &MongoDBStore{}

	if url == "" {
		return nil, ErrNoMongoHost
	}

	// We construct our URI based on the flags given.
	var uri strings.Builder

	// The protocol...
	uri.WriteString("mongodb://")

	if user != "" {
		// Let's check wether we can construct a `user:password` combination.
		// We have a username, we add it.
		uri.WriteString(user)

		if user != "" {
			// Dito for password, with divider
			uri.WriteString(":")
			uri.WriteString(pass)
		} else {
			// We do NOT have a password, which warrants a warning.
			log.Println("WARNING! User given without password!")
		}

		// Add the divider between `user:pass` and host
		uri.WriteString("@")
	}
	// Append the host[:port] combination
	uri.WriteString(url)

	// TODO: Should be checked for timeout.
	store.client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(uri.String()))

	if err != nil {
		return nil, fmt.Errorf("connecting to '%s': %s", url, err)
	}

	store.bucket, err = gridfs.NewBucket(store.client.Database(db))
	if err != nil {
		return nil, fmt.Errorf("accessing bucket '%s' in '%s' on '%s': %s", name, db, url, err)
	}
	return
}

// StreamFrom statisfies the Retriever interface.
func (s MongoDBStore) StreamFrom(path string, w io.Writer) (err error) {
	_, err = s.bucket.DownloadToStreamByName(path, w)
	return err
}

// StreamTo satisfies the Storer interface.
func (s MongoDBStore) StreamTo(path string, source io.Reader) (err error) {
	_, err = s.bucket.UploadFromStream(path, source)
	return
}

// Close satisfies the Repository interface.
func (s MongoDBStore) Close() error {
	return s.client.Disconnect(context.TODO())
}
