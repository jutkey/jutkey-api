# Jutkey API

[![Go Reference](https://pkg.go.dev/badge/github.com/jutkey/jutkey-api.svg)](https://pkg.go.dev/github.com/jutkey/jutkey-api)

Jutkey API is a decentralized wallet API that runs on the IBAX Guardian node and can be used to query data on the IBAX network.


## Deploy and Configure
You need to deploy your own IBAX Guardian node. 

You can refer to [Deployment of A IBAX Network](https://docs.ibax.io/howtos/deployment.html)

Then connect to your IBAX Guardian node database.


## Build from Source

```
$ GO111MODULE=on go mod tidy -v

$ go build -o jutkey-api
```

## Run

Starting jutkey-api.

```bash
$   jutkey-api start
```