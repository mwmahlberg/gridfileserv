package main

import "io"

// Storer takes the source and stores its contents under path for further reading via
// Retriever.
type Storer interface {
	StreamTo(path string, source io.Reader) (err error)
}

// Retriever takes a path and streams the file it has stored under path to w.
type Retriever interface {
	StreamFrom(path string, w io.Writer) (err error)
}

// Repository is a composite interface. It requires a
// repository to accept andf provide streams of files
type Repository interface {
	Storer
	Retriever
	Close() error
}
