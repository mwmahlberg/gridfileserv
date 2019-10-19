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
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/mwmahlberg/gridfileserv/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type contextKey string

type meta struct {
	ID   primitive.ObjectID `bson:"_id" json:"_id"`
	Name string
}

const filename contextKey = "filename"

var mongodb = flag.NewFlagSet("mongodb", flag.ExitOnError)
var mongoHost = mongodb.String("url", "localhost:27017", "MongoDB URL to connect to")
var mongoUser = mongodb.String("user", "", "username to authenticate against MongoDB")
var mongoPass = mongodb.String("pass", "", "password to use for authentication")
var db = mongodb.String("db", "test", "database to use")
var fs = mongodb.String("gridfs", "example", "GridFS to use")

var listen string

var filedb = flag.NewFlagSet("file", flag.ExitOnError)
var basepath = filedb.String("base", "./data", "base path to store files into")

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

	// A rather ugly hack caused by the fact that the flag package does not know shared
	// flags
	mongodb.StringVar(&listen, "listen", ":9090", "port the webserver listens on")
	filedb.StringVar(&listen, "listen", ":9090", "port the webserver listens on")

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
// extracted it for us. It then tries to find a file in the repo with the according name and efficiently
// streams it to the response writer.
func getHandler(ret Retriever) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Something is seriously wrong - all requests should have a filename in the context after
		// going through dispatcher.
		if r.Context().Value(filename) == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		// Extract the filename from the context...
		name := r.Context().Value(filename).(string)

		// ...and simply let the repo handle the streaming.
		err := ret.StreamFrom(name, w)

		// Technically, it could be a NotFound and stuff, but I leave this for brevity.
		if err != nil {
			status := http.StatusInternalServerError
			txt := fmt.Sprintf("sending data to client: %#v", err)
			http.Error(w, txt, status)
			log.Printf("'%s' %d %s", r.RequestURI, status, txt)
			return
		}
	})
}

// postHandler takes the request body verbatim and streams it to the repo.
func postHandler(store Storer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Something is seriously wrong - all requests should have a filename in the context after
		// going through dispatcher.
		if r.Context().Value(filename) == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		// Extract the filename from the context...
		name := r.Context().Value(filename).(string)

		// And create an upload of the request body, limited to 32MB in size
		if err := store.StreamTo(name, io.LimitReader(r.Body, 32<<20)); err != nil {
			http.Error(w, fmt.Sprintf("uploading '%s': %s", name, err), http.StatusInternalServerError)
		}
		r.Body.Close()

	})
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("mongodb or file subcommand is required")
		os.Exit(1)
	}

	// Our repository, which is a composition of Storer and Reetriever plus a Close method for cleanup.
	var repo Repository
	var err error

	switch os.Args[1] {
	// Depending on the subcommand selected, we parse our flags and set the repo accordingly
	case "mongodb":
		mongodb.Parse(os.Args[2:])
		repo, err = store.NewMongoDB(*mongoHost, *mongoUser, *mongoPass, *db, *fs)

	case "file":
		filedb.Parse(os.Args[2:])
		repo, err = store.NewFile(*basepath)
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	// If an error occured during the initialization of the repo, now it is time to deal with it.
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Should be checked for timeout, too.
	// Not sure about ongoing processes, for example.
	defer repo.Close()

	http.Handle("/", dispatch(getHandler(repo), postHandler(repo)))

	log.Printf("Starting webserver on %s", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}
