//go:generate protoc --go_out=. response.proto

// Store a HTTP response description in a protobuf message
// so that it can be used with groupcache.ProtoSink.
package pb
