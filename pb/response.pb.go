// Code generated by protoc-gen-go.
// source: response.proto
// DO NOT EDIT!

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	response.proto

It has these top-level messages:
	Header
	Response
*/
package pb

import proto "github.com/golang/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Header struct {
	Key              *string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value            *string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Header) Reset()         { *m = Header{} }
func (m *Header) String() string { return proto.CompactTextString(m) }
func (*Header) ProtoMessage()    {}

func (m *Header) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *Header) GetValue() string {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return ""
}

type Response struct {
	Code             *int32    `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Headers          []*Header `protobuf:"bytes,2,rep,name=headers" json:"headers,omitempty"`
	Body             []byte    `protobuf:"bytes,3,opt,name=body" json:"body,omitempty"`
	XXX_unrecognized []byte    `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func (m *Response) GetCode() int32 {
	if m != nil && m.Code != nil {
		return *m.Code
	}
	return 0
}

func (m *Response) GetHeaders() []*Header {
	if m != nil {
		return m.Headers
	}
	return nil
}

func (m *Response) GetBody() []byte {
	if m != nil {
		return m.Body
	}
	return nil
}

func init() {
}
