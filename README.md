gridfileserv
============

This repository contains code for the answer to a question on Stackoverflow: [How to handle video and image upload to storage servers?](https://stackoverflow.com/questions/58459117/how-to-handle-video-and-image-upload-to-storage-servers)

gridfileserv implements a http server which uses either a directory or MongoDB GridFS to store files uploaded to it.

Installation
------------

### Recommended: Using docker-compose

```shell
$ git clone https://github.com/mwmahlberg/gridfileserv.git
Cloning into 'gridfileserv'...
remote: Enumerating objects: 14, done.
remote: Counting objects: 100% (14/14), done.
remote: Compressing objects: 100% (12/12), done.
remote: Total 14 (delta 2), reused 11 (delta 2), pack-reused 0
Unpacking objects: 100% (14/14), done.
$ cd gridfileserv/
$ docker-compose build
docker-compose build
mongodb uses an image, skipping
Building app
Step 1/11 : FROM golang:1.13-alpine as BUILD
[...]
Successfully built 7f1ecb89df2f
Successfully tagged gridfileserv_app:latest
$ docker-compose up -d
Creating network "gridfileserv_default" with the default driver
Creating gridfileserv_mongodb_1 ... done
Creating gridfileserv_app_1     ... done
```

The application is now accessible via http://localhost:4711.

For usage, see below.

### Manually

First, you need to build the application

```shell
$ git clone https://github.com/mwmahlberg/gridfileserv.git
Cloning into 'gridfileserv'...
remote: Enumerating objects: 14, done.
remote: Counting objects: 100% (14/14), done.
remote: Compressing objects: 100% (12/12), done.
remote: Total 14 (delta 2), reused 11 (delta 2), pack-reused 0
Unpacking objects: 100% (14/14), done.
$ go build
[...]
```

#### Option 1: MongoDB GridFS storage
Next, you need a connection to a MongoDB instance. If you have Docker installed, you can simply run:

```shell
$ docker run -d --name mongotest -p 27017:27017 mongo
1d1492408639f5ef650b25357af7392f56a9b710b904a0831dc65db4396624c2
```

Note that the output on your machine will be different.

Last but not least, you need to start the application.

    gridfileserv mongodb [-listen [[hostnameOrIp]:port]] [-url otherThanLocalhost:27017] [-user <mongodbUsername> [-pass <mongodbPassword>] ]

You can pass the `-h` flag for details.

For example:

```shell
$ path/to/gridfileserv mongodb -listen :4711
2019/10/19 19:54:36 Starting webserver on :4711
```

#### Option 2: Filesystem storage

   gridfileserv file [-listen [[hostnameOrIp]:port]] [-base <path/to/file/repository>]

For example:

```shell
$ path/to/gridfileserv file -listen :4711 -base ./data
2019/10/19 19:54:36 Starting webserver on :4711
```

Usage
-----

### Add a file

```shell
$ curl --data-binary "@/path/to/beautiful.png" http://localhost:4711/files/beautiful.png
{"_id":"5dab4e2a8a7a4e05d6295c72","Name":"beautiful.png"}
*   Trying ::1:4711...
* TCP_NODELAY set
* Connected to localhost (::1) port 4711 (#0)
> POST /files/beautiful.png HTTP/1.1
> Host: localhost:4711
> User-Agent: curl/7.66.0
> Accept: */*
> Content-type: image/png
> Content-Length: 23949
> Expect: 100-continue
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 100 Continue
HTTP/1.1 100 Continue

* We are completely uploaded and fine
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
< Date: Sat, 19 Oct 2019 18:03:03 GMT
Date: Sat, 19 Oct 2019 18:03:03 GMT
< Content-Length: 52
Content-Length: 52
< Content-Type: text/plain; charset=utf-8
Content-Type: text/plain; charset=utf-8
```

### Retrieve a file

```shell
$ curl -O http://localhost:4711/files/osx.png
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
  0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0*   Trying ::1:4711...
* TCP_NODELAY set
* Connected to localhost (::1) port 4711 (#0)
> GET /files/osx.png HTTP/1.1
> Host: localhost:4711
> User-Agent: curl/7.66.0
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Date: Sat, 19 Oct 2019 18:04:39 GMT
< Content-Type: image/png
< Transfer-Encoding: chunked
< 
{ [23962 bytes data]
100 23949    0 23949    0     0  3341k      0 --:--:-- --:--:-- --:--:-- 3897k
* Connection #0 to host localhost left intact
```

Contributing
------------

This repository will be read-only soon. However, fork it and make the code your own as per...

Unlicense
---------

This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to http://unlicense.org/