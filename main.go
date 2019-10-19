// Package main provides a small example for a webserver storing uploaded files in MongoDB's GridFS.
// To easily test this, you should call
//
//  docker run -d -p 27017:27017 --name mongotest mongo
//
// so that you have a running MongoDB instance.
// You can build and run the server and the MongoDB docker image either manually or
// use the docker-compose file
//
//  docker-compose up
//
// Then, you can send data using curl, for example
//
//   curl --data-binary "@/path/to/beautiful.png" http://localhost:4711/files/beautiful.png
//
// and retrieve data with
//
//  curl -O http://localhost:4711/files/beautiful.png
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type contextKey string

type meta struct {
	ID   primitive.ObjectID `bson:"_id" json:"_id"`
	Name string
}

const filename contextKey = "filename"

// the
var mongoHost string
var mongoUser string
var mongoPass string
var db string
var fs string
var listen string

// Since we are using only the standard library aside from the MongoDB driver,
// we need to be able to examine the URLs called.
// Basically, it accepts stuff like
//  /files/foo.txt
// and
// /files/bar.baz.png
var pattern = `^/files/(?P<filename>[\w.]*)$`

// Precompile the regex on startup.
var pathRegex = regexp.MustCompile(pattern)

func init() {
	// Setup the required flags
	flag.StringVar(&mongoHost, "url", "localhost:27017", "MongoDB URL to connect to")
	flag.StringVar(&mongoUser, "user", "", "username to authenticate against MongoDB")
	flag.StringVar(&mongoPass, "pass", "", "password to use for authentication")

	flag.StringVar(&db, "db", "test", "database to use")
	flag.StringVar(&fs, "gridfs", "example", "GridFS to use")

	flag.StringVar(&listen, "listen", ":9090", "adress to listen on")
}

// dispatch handles each incoming request.
// It first checks wether the request URI fits within our format and
// if it does, delegates the request to either the get or post handler.
func dispatch(get, post http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Check wether the RequestURI fits our pattern
		p := pathRegex.FindStringSubmatch(r.RequestURI)

		// If it fits, the number of found patterns should be 2:
		// the complete match and the group we defined.
		if len(p) < 2 {
			log.Printf("404: '%s' does not match '%s'", r.RequestURI, pattern)
			http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNotFound)
			return
		}

		// We add the found filename to the request context (request scoped variables)
		ctx := context.WithValue(context.Background(), filename, p[1])

		// Finally, we delegate to the according handler
		switch r.Method {
		case http.MethodGet:
			get.ServeHTTP(w, r.WithContext(ctx))
		case http.MethodPost:
			post.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

// getHandler extracts the requested filename from the context, as the dispatcher already
// extracted it for us. It then tries to find a file with the according name and efficiently
// streams it to the response writer.
func getHandler(bucket *gridfs.Bucket) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Something is seriously wrong - all requests should have a filename in the context after
		// going through dispatcher.
		if r.Context().Value(filename) == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		// Extract the filename from the context...
		name := r.Context().Value(filename).(string)

		// ...and simply let the bucket handle the streaming.
		_, err := bucket.DownloadToStreamByName(name, w)

		// Technically, it could be a NotFound and stuff, but I leave this for brevity.
		if err != nil {
			status := http.StatusInternalServerError
			txt := fmt.Sprintf("sending data to client: %#v", err)
			http.Error(w, txt, status)
			log.Printf("'%s' %d %s", r.RequestURI, status, txt)
			return
		}
		log.Printf("'%s' %d", r.RequestURI, http.StatusOK)
	})
}

// postHandler takes the request body verbatim and writes it to the GridFS bucket.
// Note that you should add some additional security measures here,  other than limiting the size
// via io.LimitReader.
func postHandler(bucket *gridfs.Bucket) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Something is seriously wrong - all requests should have a filename in the context after
		// going through dispatcher.
		if r.Context().Value(filename) == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		// Extract the filename from the context...
		name := r.Context().Value(filename).(string)

		// And create an upload of the request body, limited to 32MB in size
		gid, err := bucket.UploadFromStream(name, io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, fmt.Sprintf("uploading '%s': %s", name, err), http.StatusInternalServerError)
		}

		// Answer with a JSON containing the filename and the ID of the newly uploaded file.
		enc := json.NewEncoder(w)
		enc.Encode(&meta{ID: gid, Name: name})
	})
}

func main() {
	flag.Parse()

	if mongoHost == "" {
		fmt.Println("Need a MongoDB host to connect to")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// We construct our URI based on the flags given.
	var uri strings.Builder

	// The protocol...
	uri.WriteString("mongodb://")

	if mongoUser != "" {
		// Let's check wether we can construct a `user:password` combination.
		// We have a username, we add it.
		uri.WriteString(mongoUser)

		if mongoPass != "" {
			// Dito for password, with divider
			uri.WriteString(":")
			uri.WriteString(mongoPass)
		} else {
			// We do NOT have a password, which warrants a warning.
			log.Println("WARNING! User given without password!")
		}

		// Add the divider between `user:pass` and host
		uri.WriteString("@")
	}
	// Append the host[:port] combination
	uri.WriteString(mongoHost)

	// TODO: Should be checked for timeout.
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri.String()))

	if err != nil {
		log.Fatalf("connecting to MongoDB: %s", err)
	}

	// TODO: Should be checked for timeout, too.
	// Not sure about ongoing processes, for example.
	defer client.Disconnect(context.TODO())

	// Let us get our GridFS bucket.
	bucket, err := gridfs.NewBucket(client.Database(db), options.GridFSBucket().SetName(fs))
	if err != nil {
		log.Fatalf("accessing bucket: %s", err)
	}

	http.Handle("/", dispatch(getHandler(bucket), postHandler(bucket)))

	log.Printf("Starting webserver on %s", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}
