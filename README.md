# stewdy

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
