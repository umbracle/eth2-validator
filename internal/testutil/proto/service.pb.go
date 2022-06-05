// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.12.0
// source: internal/testutil/proto/service.proto

package proto

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type NodeType int32

const (
	NodeType_Other     NodeType = 0
	NodeType_Beacon    NodeType = 1
	NodeType_Validator NodeType = 2
	NodeType_Bootnode  NodeType = 3
)

// Enum value maps for NodeType.
var (
	NodeType_name = map[int32]string{
		0: "Other",
		1: "Beacon",
		2: "Validator",
		3: "Bootnode",
	}
	NodeType_value = map[string]int32{
		"Other":     0,
		"Beacon":    1,
		"Validator": 2,
		"Bootnode":  3,
	}
)

func (x NodeType) Enum() *NodeType {
	p := new(NodeType)
	*p = x
	return p
}

func (x NodeType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (NodeType) Descriptor() protoreflect.EnumDescriptor {
	return file_internal_testutil_proto_service_proto_enumTypes[0].Descriptor()
}

func (NodeType) Type() protoreflect.EnumType {
	return &file_internal_testutil_proto_service_proto_enumTypes[0]
}

func (x NodeType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use NodeType.Descriptor instead.
func (NodeType) EnumDescriptor() ([]byte, []int) {
	return file_internal_testutil_proto_service_proto_rawDescGZIP(), []int{0}
}

type NodeClient int32

const (
	NodeClient_Prysm      NodeClient = 0
	NodeClient_Teku       NodeClient = 1
	NodeClient_Lighthouse NodeClient = 2
)

// Enum value maps for NodeClient.
var (
	NodeClient_name = map[int32]string{
		0: "Prysm",
		1: "Teku",
		2: "Lighthouse",
	}
	NodeClient_value = map[string]int32{
		"Prysm":      0,
		"Teku":       1,
		"Lighthouse": 2,
	}
)

func (x NodeClient) Enum() *NodeClient {
	p := new(NodeClient)
	*p = x
	return p
}

func (x NodeClient) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (NodeClient) Descriptor() protoreflect.EnumDescriptor {
	return file_internal_testutil_proto_service_proto_enumTypes[1].Descriptor()
}

func (NodeClient) Type() protoreflect.EnumType {
	return &file_internal_testutil_proto_service_proto_enumTypes[1]
}

func (x NodeClient) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use NodeClient.Descriptor instead.
func (NodeClient) EnumDescriptor() ([]byte, []int) {
	return file_internal_testutil_proto_service_proto_rawDescGZIP(), []int{1}
}

type DeployNodeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeClient NodeClient `protobuf:"varint,1,opt,name=nodeClient,proto3,enum=proto.NodeClient" json:"nodeClient,omitempty"`
}

func (x *DeployNodeRequest) Reset() {
	*x = DeployNodeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_testutil_proto_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeployNodeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeployNodeRequest) ProtoMessage() {}

func (x *DeployNodeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_testutil_proto_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeployNodeRequest.ProtoReflect.Descriptor instead.
func (*DeployNodeRequest) Descriptor() ([]byte, []int) {
	return file_internal_testutil_proto_service_proto_rawDescGZIP(), []int{0}
}

func (x *DeployNodeRequest) GetNodeClient() NodeClient {
	if x != nil {
		return x.NodeClient
	}
	return NodeClient_Prysm
}

type DeployNodeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DeployNodeResponse) Reset() {
	*x = DeployNodeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_testutil_proto_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeployNodeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeployNodeResponse) ProtoMessage() {}

func (x *DeployNodeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_testutil_proto_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeployNodeResponse.ProtoReflect.Descriptor instead.
func (*DeployNodeResponse) Descriptor() ([]byte, []int) {
	return file_internal_testutil_proto_service_proto_rawDescGZIP(), []int{1}
}

type DeployValidatorRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NumValidators uint64     `protobuf:"varint,1,opt,name=numValidators,proto3" json:"numValidators,omitempty"`
	NodeClient    NodeClient `protobuf:"varint,2,opt,name=nodeClient,proto3,enum=proto.NodeClient" json:"nodeClient,omitempty"`
}

func (x *DeployValidatorRequest) Reset() {
	*x = DeployValidatorRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_testutil_proto_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeployValidatorRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeployValidatorRequest) ProtoMessage() {}

func (x *DeployValidatorRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_testutil_proto_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeployValidatorRequest.ProtoReflect.Descriptor instead.
func (*DeployValidatorRequest) Descriptor() ([]byte, []int) {
	return file_internal_testutil_proto_service_proto_rawDescGZIP(), []int{2}
}

func (x *DeployValidatorRequest) GetNumValidators() uint64 {
	if x != nil {
		return x.NumValidators
	}
	return 0
}

func (x *DeployValidatorRequest) GetNodeClient() NodeClient {
	if x != nil {
		return x.NodeClient
	}
	return NodeClient_Prysm
}

type DeployValidatorResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DeployValidatorResponse) Reset() {
	*x = DeployValidatorResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_testutil_proto_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeployValidatorResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeployValidatorResponse) ProtoMessage() {}

func (x *DeployValidatorResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_testutil_proto_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeployValidatorResponse.ProtoReflect.Descriptor instead.
func (*DeployValidatorResponse) Descriptor() ([]byte, []int) {
	return file_internal_testutil_proto_service_proto_rawDescGZIP(), []int{3}
}

var File_internal_testutil_proto_service_proto protoreflect.FileDescriptor

var file_internal_testutil_proto_service_proto_rawDesc = []byte{
	0x0a, 0x25, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x75,
	0x74, 0x69, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x46,
	0x0a, 0x11, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x31, 0x0a, 0x0a, 0x6e, 0x6f, 0x64, 0x65, 0x43, 0x6c, 0x69, 0x65, 0x6e,
	0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x4e, 0x6f, 0x64, 0x65, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x52, 0x0a, 0x6e, 0x6f, 0x64, 0x65,
	0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x22, 0x14, 0x0a, 0x12, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x71, 0x0a, 0x16,
	0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x24, 0x0a, 0x0d, 0x6e, 0x75, 0x6d, 0x56, 0x61, 0x6c,
	0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0d, 0x6e,
	0x75, 0x6d, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73, 0x12, 0x31, 0x0a, 0x0a,
	0x6e, 0x6f, 0x64, 0x65, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4e, 0x6f, 0x64, 0x65, 0x43, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x52, 0x0a, 0x6e, 0x6f, 0x64, 0x65, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x22,
	0x19, 0x0a, 0x17, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x6f, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2a, 0x3e, 0x0a, 0x08, 0x4e, 0x6f,
	0x64, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x09, 0x0a, 0x05, 0x4f, 0x74, 0x68, 0x65, 0x72, 0x10,
	0x00, 0x12, 0x0a, 0x0a, 0x06, 0x42, 0x65, 0x61, 0x63, 0x6f, 0x6e, 0x10, 0x01, 0x12, 0x0d, 0x0a,
	0x09, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x10, 0x02, 0x12, 0x0c, 0x0a, 0x08,
	0x42, 0x6f, 0x6f, 0x74, 0x6e, 0x6f, 0x64, 0x65, 0x10, 0x03, 0x2a, 0x31, 0x0a, 0x0a, 0x4e, 0x6f,
	0x64, 0x65, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x12, 0x09, 0x0a, 0x05, 0x50, 0x72, 0x79, 0x73,
	0x6d, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x54, 0x65, 0x6b, 0x75, 0x10, 0x01, 0x12, 0x0e, 0x0a,
	0x0a, 0x4c, 0x69, 0x67, 0x68, 0x74, 0x68, 0x6f, 0x75, 0x73, 0x65, 0x10, 0x02, 0x32, 0xa1, 0x01,
	0x0a, 0x0a, 0x45, 0x32, 0x45, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x41, 0x0a, 0x0a,
	0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x4e, 0x6f, 0x64, 0x65, 0x12, 0x18, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x44, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x50, 0x0a, 0x0f, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x6f, 0x72, 0x12, 0x1d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x42, 0x1a, 0x5a, 0x18, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74,
	0x65, 0x73, 0x74, 0x75, 0x74, 0x69, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_testutil_proto_service_proto_rawDescOnce sync.Once
	file_internal_testutil_proto_service_proto_rawDescData = file_internal_testutil_proto_service_proto_rawDesc
)

func file_internal_testutil_proto_service_proto_rawDescGZIP() []byte {
	file_internal_testutil_proto_service_proto_rawDescOnce.Do(func() {
		file_internal_testutil_proto_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_testutil_proto_service_proto_rawDescData)
	})
	return file_internal_testutil_proto_service_proto_rawDescData
}

var file_internal_testutil_proto_service_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_internal_testutil_proto_service_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_internal_testutil_proto_service_proto_goTypes = []interface{}{
	(NodeType)(0),                   // 0: proto.NodeType
	(NodeClient)(0),                 // 1: proto.NodeClient
	(*DeployNodeRequest)(nil),       // 2: proto.DeployNodeRequest
	(*DeployNodeResponse)(nil),      // 3: proto.DeployNodeResponse
	(*DeployValidatorRequest)(nil),  // 4: proto.DeployValidatorRequest
	(*DeployValidatorResponse)(nil), // 5: proto.DeployValidatorResponse
}
var file_internal_testutil_proto_service_proto_depIdxs = []int32{
	1, // 0: proto.DeployNodeRequest.nodeClient:type_name -> proto.NodeClient
	1, // 1: proto.DeployValidatorRequest.nodeClient:type_name -> proto.NodeClient
	2, // 2: proto.E2EService.DeployNode:input_type -> proto.DeployNodeRequest
	4, // 3: proto.E2EService.DeployValidator:input_type -> proto.DeployValidatorRequest
	3, // 4: proto.E2EService.DeployNode:output_type -> proto.DeployNodeResponse
	5, // 5: proto.E2EService.DeployValidator:output_type -> proto.DeployValidatorResponse
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_internal_testutil_proto_service_proto_init() }
func file_internal_testutil_proto_service_proto_init() {
	if File_internal_testutil_proto_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_testutil_proto_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeployNodeRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_testutil_proto_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeployNodeResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_testutil_proto_service_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeployValidatorRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_testutil_proto_service_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeployValidatorResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_testutil_proto_service_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_internal_testutil_proto_service_proto_goTypes,
		DependencyIndexes: file_internal_testutil_proto_service_proto_depIdxs,
		EnumInfos:         file_internal_testutil_proto_service_proto_enumTypes,
		MessageInfos:      file_internal_testutil_proto_service_proto_msgTypes,
	}.Build()
	File_internal_testutil_proto_service_proto = out.File
	file_internal_testutil_proto_service_proto_rawDesc = nil
	file_internal_testutil_proto_service_proto_goTypes = nil
	file_internal_testutil_proto_service_proto_depIdxs = nil
}
