# Distributed File Storage

A peer-to-peer distributed file storage system written in Go. Files written to any node are replicated to connected peers over TCP, encrypted in transit with AES-CTR, and stored on disk using content-addressed paths.

## Features

- **Peer-to-peer networking** over a pluggable TCP transport
- **Content-addressed storage** — keys are hashed (SHA-1) and sharded into nested directories
- **AES-CTR encryption** of data streamed between peers (per-file random IV)
- **Per-node isolation** — each node stores files under its own ID namespace
- **Streaming reads/writes** — files are piped through `io.Reader` end-to-end
- **Bootstrap nodes** — new nodes discover the network by dialing a configurable list of peers

## Project layout

```
.
├── main.go        # Entry point — spins up three sample nodes
├── server.go      # FileServer: Store / Get, broadcast, peer handling
├── store.go       # Disk store with CAS path transform
├── crypto.go      # AES-CTR encrypt/decrypt stream helpers, ID + key generation
└── p2p/           # Transport, Peer, RPC, handshake, encoding
    ├── transport.go
    ├── tcp_transport.go
    ├── message.go
    ├── encoding.go
    └── handshake.go
```

## Requirements

- Go 1.25+

## Build & run

```sh
make build   # produces bin/app
make run     # build and run the demo
make test    # run all tests
```

The demo in `main.go` starts three nodes on `:3000`, `:4000`, `:7000`, stores a file from one node, deletes it locally, then reads it back (pulling it from a peer).

## How it works

1. **Store** — `FileServer.Store(key, reader)` writes the file to the local disk store, then broadcasts a `MessageStoreFile` to all peers and streams the AES-encrypted payload to them via an `io.MultiWriter`.
2. **Get** — `FileServer.Get(key)` returns the file from local disk if present; otherwise it broadcasts a `MessageGetFile`, decrypts the incoming stream from a peer, persists it locally, and returns a reader.
3. **Paths** — keys are hashed and split into 5-character segments (e.g. `abcde/fghij/...`) under `<listen-addr>_network/<node-id>/`.
