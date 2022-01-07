# OpenTrace Server - Hyperjump Golang Implementation

This is an adaptation implementation from OpenTrace community server
that implements BlueTrace.io specification.

## Build

```shell
$ make build
```

this will produce an executable called `OTMock.app`

## Execute

```shell
$ OTMock.app
```

The server will run on port `8080`
Implementor should modify this bare server to be more configurable as needed.

## API

After the server, you can go to `/docs` path. Eg.

```text
http://localhost:8080/docs
```