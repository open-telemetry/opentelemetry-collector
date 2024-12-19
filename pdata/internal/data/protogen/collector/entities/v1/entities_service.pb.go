// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: opentelemetry/proto/collector/entities/v1/entities_service.proto

package v1

import (
	context "context"
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	v1 "go.opentelemetry.io/collector/pdata/internal/data/protogen/entities/v1"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type ExportEntitiesServiceRequest struct {
	// An array of ResourceEntities.
	ResourceEntities []*v1.ResourceEntities `protobuf:"bytes,1,rep,name=resource_entities,json=resourceEntities,proto3" json:"resource_entities,omitempty"`
}

func (m *ExportEntitiesServiceRequest) Reset()         { *m = ExportEntitiesServiceRequest{} }
func (m *ExportEntitiesServiceRequest) String() string { return proto.CompactTextString(m) }
func (*ExportEntitiesServiceRequest) ProtoMessage()    {}
func (*ExportEntitiesServiceRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_fbd01640b6ae8848, []int{0}
}
func (m *ExportEntitiesServiceRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ExportEntitiesServiceRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ExportEntitiesServiceRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ExportEntitiesServiceRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ExportEntitiesServiceRequest.Merge(m, src)
}
func (m *ExportEntitiesServiceRequest) XXX_Size() int {
	return m.Size()
}
func (m *ExportEntitiesServiceRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ExportEntitiesServiceRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ExportEntitiesServiceRequest proto.InternalMessageInfo

func (m *ExportEntitiesServiceRequest) GetResourceEntities() []*v1.ResourceEntities {
	if m != nil {
		return m.ResourceEntities
	}
	return nil
}

type ExportEntitiesServiceResponse struct {
	// The details of a partially successful export request.
	//
	// If the request is only partially accepted
	// (i.e. when the server accepts only parts of the data and rejects the rest)
	// the server MUST initialize the `partial_success` field and MUST
	// set the `rejected_<signal>` with the number of items it rejected.
	//
	// Servers MAY also make use of the `partial_success` field to convey
	// warnings/suggestions to senders even when the request was fully accepted.
	// In such cases, the `rejected_<signal>` MUST have a value of `0` and
	// the `error_message` MUST be non-empty.
	//
	// A `partial_success` message with an empty value (rejected_<signal> = 0 and
	// `error_message` = "") is equivalent to it not being set/present. Senders
	// SHOULD interpret it the same way as in the full success case.
	PartialSuccess ExportEntitiesPartialSuccess `protobuf:"bytes,1,opt,name=partial_success,json=partialSuccess,proto3" json:"partial_success"`
}

func (m *ExportEntitiesServiceResponse) Reset()         { *m = ExportEntitiesServiceResponse{} }
func (m *ExportEntitiesServiceResponse) String() string { return proto.CompactTextString(m) }
func (*ExportEntitiesServiceResponse) ProtoMessage()    {}
func (*ExportEntitiesServiceResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_fbd01640b6ae8848, []int{1}
}
func (m *ExportEntitiesServiceResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ExportEntitiesServiceResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ExportEntitiesServiceResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ExportEntitiesServiceResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ExportEntitiesServiceResponse.Merge(m, src)
}
func (m *ExportEntitiesServiceResponse) XXX_Size() int {
	return m.Size()
}
func (m *ExportEntitiesServiceResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ExportEntitiesServiceResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ExportEntitiesServiceResponse proto.InternalMessageInfo

func (m *ExportEntitiesServiceResponse) GetPartialSuccess() ExportEntitiesPartialSuccess {
	if m != nil {
		return m.PartialSuccess
	}
	return ExportEntitiesPartialSuccess{}
}

type ExportEntitiesPartialSuccess struct {
	// The number of rejected EntityEvents.
	//
	// A `rejected_<signal>` field holding a `0` value indicates that the
	// request was fully accepted.
	RejectedEntities int64 `protobuf:"varint,1,opt,name=rejected_entities,json=rejectedEntities,proto3" json:"rejected_entities,omitempty"`
	// A developer-facing human-readable message in English. It should be used
	// either to explain why the server rejected parts of the data during a partial
	// success or to convey warnings/suggestions during a full success. The message
	// should offer guidance on how users can address such issues.
	//
	// error_message is an optional field. An error_message with an empty value
	// is equivalent to it not being set.
	ErrorMessage string `protobuf:"bytes,2,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
}

func (m *ExportEntitiesPartialSuccess) Reset()         { *m = ExportEntitiesPartialSuccess{} }
func (m *ExportEntitiesPartialSuccess) String() string { return proto.CompactTextString(m) }
func (*ExportEntitiesPartialSuccess) ProtoMessage()    {}
func (*ExportEntitiesPartialSuccess) Descriptor() ([]byte, []int) {
	return fileDescriptor_fbd01640b6ae8848, []int{2}
}
func (m *ExportEntitiesPartialSuccess) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ExportEntitiesPartialSuccess) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ExportEntitiesPartialSuccess.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ExportEntitiesPartialSuccess) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ExportEntitiesPartialSuccess.Merge(m, src)
}
func (m *ExportEntitiesPartialSuccess) XXX_Size() int {
	return m.Size()
}
func (m *ExportEntitiesPartialSuccess) XXX_DiscardUnknown() {
	xxx_messageInfo_ExportEntitiesPartialSuccess.DiscardUnknown(m)
}

var xxx_messageInfo_ExportEntitiesPartialSuccess proto.InternalMessageInfo

func (m *ExportEntitiesPartialSuccess) GetRejectedEntities() int64 {
	if m != nil {
		return m.RejectedEntities
	}
	return 0
}

func (m *ExportEntitiesPartialSuccess) GetErrorMessage() string {
	if m != nil {
		return m.ErrorMessage
	}
	return ""
}

func init() {
	proto.RegisterType((*ExportEntitiesServiceRequest)(nil), "opentelemetry.proto.collector.entities.v1.ExportEntitiesServiceRequest")
	proto.RegisterType((*ExportEntitiesServiceResponse)(nil), "opentelemetry.proto.collector.entities.v1.ExportEntitiesServiceResponse")
	proto.RegisterType((*ExportEntitiesPartialSuccess)(nil), "opentelemetry.proto.collector.entities.v1.ExportEntitiesPartialSuccess")
}

func init() {
	proto.RegisterFile("opentelemetry/proto/collector/entities/v1/entities_service.proto", fileDescriptor_fbd01640b6ae8848)
}

var fileDescriptor_fbd01640b6ae8848 = []byte{
	// 424 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x53, 0xcd, 0xaa, 0xd3, 0x40,
	0x14, 0xce, 0xb4, 0x52, 0x70, 0xaa, 0x56, 0x43, 0x17, 0xa5, 0x68, 0x2c, 0x71, 0x93, 0xa2, 0x4c,
	0x48, 0x7d, 0x01, 0xa9, 0x14, 0xdd, 0x88, 0x21, 0x15, 0x17, 0x2e, 0x0c, 0x31, 0x3d, 0x84, 0x48,
	0x9a, 0x19, 0x67, 0xa6, 0x41, 0x37, 0x3e, 0x83, 0x3b, 0x57, 0xbe, 0x80, 0x3b, 0xdf, 0xa2, 0xcb,
	0x2e, 0x5d, 0x88, 0x5c, 0xda, 0x17, 0xb9, 0x24, 0xd3, 0x84, 0x26, 0xe4, 0x42, 0xb9, 0x77, 0x37,
	0xf9, 0x72, 0xbe, 0x9f, 0xf9, 0x86, 0x83, 0x5f, 0x50, 0x06, 0xa9, 0x84, 0x04, 0xd6, 0x20, 0xf9,
	0x37, 0x9b, 0x71, 0x2a, 0xa9, 0x1d, 0xd2, 0x24, 0x81, 0x50, 0x52, 0x6e, 0x43, 0x2a, 0x63, 0x19,
	0x83, 0xb0, 0x33, 0xa7, 0x3a, 0xfb, 0x02, 0x78, 0x16, 0x87, 0x40, 0x8a, 0x61, 0x7d, 0x5a, 0x53,
	0x50, 0x20, 0xa9, 0x14, 0x48, 0xc9, 0x22, 0x99, 0x33, 0x1e, 0x46, 0x34, 0xa2, 0xca, 0x22, 0x3f,
	0xa9, 0xd9, 0x31, 0x69, 0x8b, 0xd0, 0x66, 0xac, 0xe6, 0xcd, 0xef, 0xf8, 0xe1, 0xe2, 0x2b, 0xa3,
	0x5c, 0x2e, 0x8e, 0xf8, 0x52, 0xe5, 0xf1, 0xe0, 0xcb, 0x06, 0x84, 0xd4, 0x3f, 0xe2, 0x07, 0x1c,
	0x04, 0xdd, 0xf0, 0x10, 0xfc, 0x92, 0x3a, 0x42, 0x93, 0xae, 0xd5, 0x9f, 0x39, 0xa4, 0x2d, 0xec,
	0x49, 0x44, 0xe2, 0x1d, 0x99, 0xa5, 0xb6, 0x77, 0x9f, 0x37, 0x10, 0xf3, 0x27, 0xc2, 0x8f, 0xae,
	0x08, 0x20, 0x18, 0x4d, 0x05, 0xe8, 0x19, 0x1e, 0xb0, 0x80, 0xcb, 0x38, 0x48, 0x7c, 0xb1, 0x09,
	0x43, 0x10, 0xb9, 0x3f, 0xb2, 0xfa, 0xb3, 0x57, 0xe4, 0xec, 0xb2, 0x48, 0xdd, 0xc2, 0x55, 0x7a,
	0x4b, 0x25, 0x37, 0xbf, 0xb5, 0xfd, 0xff, 0x58, 0xf3, 0xee, 0xb1, 0x1a, 0x6a, 0xb2, 0x66, 0x33,
	0x75, 0x96, 0xfe, 0x34, 0x6f, 0xe6, 0x33, 0x84, 0x12, 0x56, 0xa7, 0xcd, 0x20, 0xab, 0x9b, 0x5f,
	0x53, 0xfd, 0x28, 0xa9, 0xfa, 0x13, 0x7c, 0x17, 0x38, 0xa7, 0xdc, 0x5f, 0x83, 0x10, 0x41, 0x04,
	0xa3, 0xce, 0x04, 0x59, 0xb7, 0xbd, 0x3b, 0x05, 0xf8, 0x46, 0x61, 0xb3, 0x3f, 0x08, 0x0f, 0x1a,
	0x2d, 0xe8, 0xbf, 0x10, 0xee, 0xa9, 0x18, 0xfa, 0xf5, 0xef, 0x5b, 0x7f, 0xd3, 0xf1, 0xeb, 0x9b,
	0x0b, 0xa9, 0xb7, 0x31, 0xb5, 0xf9, 0x3f, 0xb4, 0xdd, 0x1b, 0x68, 0xb7, 0x37, 0xd0, 0xc5, 0xde,
	0x40, 0x3f, 0x0e, 0x86, 0xb6, 0x3b, 0x18, 0xda, 0xdf, 0x83, 0xa1, 0xe1, 0x67, 0x31, 0x3d, 0xdf,
	0x68, 0x3e, 0x6c, 0x78, 0xb8, 0xf9, 0xac, 0x8b, 0x3e, 0xb8, 0x51, 0x53, 0x25, 0x3e, 0xdd, 0x29,
	0xb6, 0x0a, 0x64, 0x60, 0xc7, 0xa9, 0x04, 0x9e, 0x06, 0x89, 0x5d, 0x7c, 0x15, 0x36, 0x11, 0xa4,
	0xed, 0xab, 0xf7, 0xbb, 0x33, 0x7d, 0xcb, 0x20, 0x7d, 0x57, 0xe9, 0x15, 0x4e, 0xe4, 0x65, 0x95,
	0xaa, 0x0c, 0x42, 0xde, 0x3b, 0x9f, 0x7a, 0x85, 0xd6, 0xf3, 0xcb, 0x00, 0x00, 0x00, 0xff, 0xff,
	0xc4, 0xc3, 0xd2, 0xe0, 0xda, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// EntitiesServiceClient is the client API for EntitiesService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type EntitiesServiceClient interface {
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(ctx context.Context, in *ExportEntitiesServiceRequest, opts ...grpc.CallOption) (*ExportEntitiesServiceResponse, error)
}

type entitiesServiceClient struct {
	cc *grpc.ClientConn
}

func NewEntitiesServiceClient(cc *grpc.ClientConn) EntitiesServiceClient {
	return &entitiesServiceClient{cc}
}

func (c *entitiesServiceClient) Export(ctx context.Context, in *ExportEntitiesServiceRequest, opts ...grpc.CallOption) (*ExportEntitiesServiceResponse, error) {
	out := new(ExportEntitiesServiceResponse)
	err := c.cc.Invoke(ctx, "/opentelemetry.proto.collector.entities.v1.EntitiesService/Export", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EntitiesServiceServer is the server API for EntitiesService service.
type EntitiesServiceServer interface {
	// For performance reasons, it is recommended to keep this RPC
	// alive for the entire life of the application.
	Export(context.Context, *ExportEntitiesServiceRequest) (*ExportEntitiesServiceResponse, error)
}

// UnimplementedEntitiesServiceServer can be embedded to have forward compatible implementations.
type UnimplementedEntitiesServiceServer struct {
}

func (*UnimplementedEntitiesServiceServer) Export(ctx context.Context, req *ExportEntitiesServiceRequest) (*ExportEntitiesServiceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Export not implemented")
}

func RegisterEntitiesServiceServer(s *grpc.Server, srv EntitiesServiceServer) {
	s.RegisterService(&_EntitiesService_serviceDesc, srv)
}

func _EntitiesService_Export_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExportEntitiesServiceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EntitiesServiceServer).Export(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/opentelemetry.proto.collector.entities.v1.EntitiesService/Export",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EntitiesServiceServer).Export(ctx, req.(*ExportEntitiesServiceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _EntitiesService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "opentelemetry.proto.collector.entities.v1.EntitiesService",
	HandlerType: (*EntitiesServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Export",
			Handler:    _EntitiesService_Export_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "opentelemetry/proto/collector/entities/v1/entities_service.proto",
}

func (m *ExportEntitiesServiceRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ExportEntitiesServiceRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ExportEntitiesServiceRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ResourceEntities) > 0 {
		for iNdEx := len(m.ResourceEntities) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.ResourceEntities[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintEntitiesService(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func (m *ExportEntitiesServiceResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ExportEntitiesServiceResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ExportEntitiesServiceResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.PartialSuccess.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintEntitiesService(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *ExportEntitiesPartialSuccess) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ExportEntitiesPartialSuccess) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ExportEntitiesPartialSuccess) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ErrorMessage) > 0 {
		i -= len(m.ErrorMessage)
		copy(dAtA[i:], m.ErrorMessage)
		i = encodeVarintEntitiesService(dAtA, i, uint64(len(m.ErrorMessage)))
		i--
		dAtA[i] = 0x12
	}
	if m.RejectedEntities != 0 {
		i = encodeVarintEntitiesService(dAtA, i, uint64(m.RejectedEntities))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintEntitiesService(dAtA []byte, offset int, v uint64) int {
	offset -= sovEntitiesService(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ExportEntitiesServiceRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.ResourceEntities) > 0 {
		for _, e := range m.ResourceEntities {
			l = e.Size()
			n += 1 + l + sovEntitiesService(uint64(l))
		}
	}
	return n
}

func (m *ExportEntitiesServiceResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.PartialSuccess.Size()
	n += 1 + l + sovEntitiesService(uint64(l))
	return n
}

func (m *ExportEntitiesPartialSuccess) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.RejectedEntities != 0 {
		n += 1 + sovEntitiesService(uint64(m.RejectedEntities))
	}
	l = len(m.ErrorMessage)
	if l > 0 {
		n += 1 + l + sovEntitiesService(uint64(l))
	}
	return n
}

func sovEntitiesService(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozEntitiesService(x uint64) (n int) {
	return sovEntitiesService(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ExportEntitiesServiceRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEntitiesService
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ExportEntitiesServiceRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ExportEntitiesServiceRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ResourceEntities", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEntitiesService
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEntitiesService
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEntitiesService
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ResourceEntities = append(m.ResourceEntities, &v1.ResourceEntities{})
			if err := m.ResourceEntities[len(m.ResourceEntities)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEntitiesService(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEntitiesService
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ExportEntitiesServiceResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEntitiesService
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ExportEntitiesServiceResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ExportEntitiesServiceResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PartialSuccess", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEntitiesService
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEntitiesService
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEntitiesService
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.PartialSuccess.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEntitiesService(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEntitiesService
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ExportEntitiesPartialSuccess) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEntitiesService
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ExportEntitiesPartialSuccess: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ExportEntitiesPartialSuccess: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RejectedEntities", wireType)
			}
			m.RejectedEntities = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEntitiesService
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RejectedEntities |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ErrorMessage", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEntitiesService
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEntitiesService
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEntitiesService
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ErrorMessage = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEntitiesService(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEntitiesService
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipEntitiesService(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowEntitiesService
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEntitiesService
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEntitiesService
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthEntitiesService
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupEntitiesService
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthEntitiesService
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthEntitiesService        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowEntitiesService          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupEntitiesService = fmt.Errorf("proto: unexpected end of group")
)
