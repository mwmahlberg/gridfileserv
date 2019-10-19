FROM golang:1.13-alpine as BUILD
# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

FROM alpine:latest
COPY --from=BUILD /app/main /usr/local/bin/gridfileserv 

# We use jwilder/dockerize to ensure MongoDB is started and ready before our application starts.
ENV DOCKERIZE_VERSION v0.6.1
RUN apk add --no-cache openssl && wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

CMD ["/usr/local/bin/dockerize" ,"-wait","tcp://localhost:27017","/usr/local/bin/gridfileserv"]