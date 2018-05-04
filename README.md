# stewdy

[![Stewdy](https://img.shields.io/badge/stewdy-alpha-lightgrey.svg)](https://github.com/ivahaev/stewdy)
[![Build Status](https://travis-ci.org/ivahaev/stewdy.svg?branch=master)](https://travis-ci.org/ivahaev/stewdy)
[![GoDoc](https://godoc.org/github.com/ivahaev/stewdy?status.svg)](https://godoc.org/github.com/ivahaev/stewdy)
[![License](https://img.shields.io/badge/license-MIT%20v3-blue.svg)](https://github.com/ivahaev/stewdy/blob/master/LICENSE)

Stupid dialer library written on Go

This is an attempt to create library for making dialers with any call targets sources and any PBX.

Project at a very early stage.

## Generating code

Install the standard protocol buffer implementation from [https://github.com/google/protobuf](https://github.com/google/protobuf).

Install [Protocol Buffers for Go with Gadgets](https://github.com/gogo/protobuf):

```bash
go get github.com/gogo/protobuf/protoc-gen-gofast
```

Generate structs from .proto:

```bash
protoc --gofast_out=. *.proto
```

Generate stringer

```bash
go generate
```
