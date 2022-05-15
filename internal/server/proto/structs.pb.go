// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.12.0
// source: internal/server/proto/structs.proto

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

type ListDutiesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ListDutiesRequest) Reset() {
	*x = ListDutiesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_server_proto_structs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListDutiesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDutiesRequest) ProtoMessage() {}

func (x *ListDutiesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_server_proto_structs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDutiesRequest.ProtoReflect.Descriptor instead.
func (*ListDutiesRequest) Descriptor() ([]byte, []int) {
	return file_internal_server_proto_structs_proto_rawDescGZIP(), []int{0}
}

type ListDutiesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Duties []*Duty `protobuf:"bytes,1,rep,name=duties,proto3" json:"duties,omitempty"`
}

func (x *ListDutiesResponse) Reset() {
	*x = ListDutiesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_server_proto_structs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListDutiesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDutiesResponse) ProtoMessage() {}

func (x *ListDutiesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_server_proto_structs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDutiesResponse.ProtoReflect.Descriptor instead.
func (*ListDutiesResponse) Descriptor() ([]byte, []int) {
	return file_internal_server_proto_structs_proto_rawDescGZIP(), []int{1}
}

func (x *ListDutiesResponse) GetDuties() []*Duty {
	if x != nil {
		return x.Duties
	}
	return nil
}

type Duty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// pubKey is the public key of the validator that signs the job
	PubKey string `protobuf:"bytes,1,opt,name=pubKey,proto3" json:"pubKey,omitempty"`
	// slot is the slot at which this job was created
	Slot uint64 `protobuf:"varint,2,opt,name=slot,proto3" json:"slot,omitempty"`
	// Types that are assignable to Job:
	//	*Duty_BlockProposal
	//	*Duty_Attestation
	Job isDuty_Job `protobuf_oneof:"job"`
}

func (x *Duty) Reset() {
	*x = Duty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_server_proto_structs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Duty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Duty) ProtoMessage() {}

func (x *Duty) ProtoReflect() protoreflect.Message {
	mi := &file_internal_server_proto_structs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Duty.ProtoReflect.Descriptor instead.
func (*Duty) Descriptor() ([]byte, []int) {
	return file_internal_server_proto_structs_proto_rawDescGZIP(), []int{2}
}

func (x *Duty) GetPubKey() string {
	if x != nil {
		return x.PubKey
	}
	return ""
}

func (x *Duty) GetSlot() uint64 {
	if x != nil {
		return x.Slot
	}
	return 0
}

func (m *Duty) GetJob() isDuty_Job {
	if m != nil {
		return m.Job
	}
	return nil
}

func (x *Duty) GetBlockProposal() *BlockProposal {
	if x, ok := x.GetJob().(*Duty_BlockProposal); ok {
		return x.BlockProposal
	}
	return nil
}

func (x *Duty) GetAttestation() *Attestation {
	if x, ok := x.GetJob().(*Duty_Attestation); ok {
		return x.Attestation
	}
	return nil
}

type isDuty_Job interface {
	isDuty_Job()
}

type Duty_BlockProposal struct {
	BlockProposal *BlockProposal `protobuf:"bytes,3,opt,name=BlockProposal,proto3,oneof"`
}

type Duty_Attestation struct {
	Attestation *Attestation `protobuf:"bytes,4,opt,name=Attestation,proto3,oneof"`
}

func (*Duty_BlockProposal) isDuty_Job() {}

func (*Duty_Attestation) isDuty_Job() {}

type BlockProposal struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Root string `protobuf:"bytes,2,opt,name=root,proto3" json:"root,omitempty"`
}

func (x *BlockProposal) Reset() {
	*x = BlockProposal{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_server_proto_structs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BlockProposal) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlockProposal) ProtoMessage() {}

func (x *BlockProposal) ProtoReflect() protoreflect.Message {
	mi := &file_internal_server_proto_structs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlockProposal.ProtoReflect.Descriptor instead.
func (*BlockProposal) Descriptor() ([]byte, []int) {
	return file_internal_server_proto_structs_proto_rawDescGZIP(), []int{3}
}

func (x *BlockProposal) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

type Attestation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Root   string                  `protobuf:"bytes,2,opt,name=root,proto3" json:"root,omitempty"`
	Source *Attestation_Checkpoint `protobuf:"bytes,3,opt,name=source,proto3" json:"source,omitempty"`
	Target *Attestation_Checkpoint `protobuf:"bytes,4,opt,name=target,proto3" json:"target,omitempty"`
}

func (x *Attestation) Reset() {
	*x = Attestation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_server_proto_structs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Attestation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Attestation) ProtoMessage() {}

func (x *Attestation) ProtoReflect() protoreflect.Message {
	mi := &file_internal_server_proto_structs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Attestation.ProtoReflect.Descriptor instead.
func (*Attestation) Descriptor() ([]byte, []int) {
	return file_internal_server_proto_structs_proto_rawDescGZIP(), []int{4}
}

func (x *Attestation) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

func (x *Attestation) GetSource() *Attestation_Checkpoint {
	if x != nil {
		return x.Source
	}
	return nil
}

func (x *Attestation) GetTarget() *Attestation_Checkpoint {
	if x != nil {
		return x.Target
	}
	return nil
}

type Attestation_Checkpoint struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Root  string `protobuf:"bytes,1,opt,name=root,proto3" json:"root,omitempty"`
	Epoch uint64 `protobuf:"varint,2,opt,name=epoch,proto3" json:"epoch,omitempty"`
}

func (x *Attestation_Checkpoint) Reset() {
	*x = Attestation_Checkpoint{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_server_proto_structs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Attestation_Checkpoint) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Attestation_Checkpoint) ProtoMessage() {}

func (x *Attestation_Checkpoint) ProtoReflect() protoreflect.Message {
	mi := &file_internal_server_proto_structs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Attestation_Checkpoint.ProtoReflect.Descriptor instead.
func (*Attestation_Checkpoint) Descriptor() ([]byte, []int) {
	return file_internal_server_proto_structs_proto_rawDescGZIP(), []int{4, 0}
}

func (x *Attestation_Checkpoint) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

func (x *Attestation_Checkpoint) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

var File_internal_server_proto_structs_proto protoreflect.FileDescriptor

var file_internal_server_proto_structs_proto_rawDesc = []byte{
	0x0a, 0x23, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x13, 0x0a, 0x11,
	0x4c, 0x69, 0x73, 0x74, 0x44, 0x75, 0x74, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x22, 0x39, 0x0a, 0x12, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x75, 0x74, 0x69, 0x65, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x23, 0x0a, 0x06, 0x64, 0x75, 0x74, 0x69, 0x65,
	0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x44, 0x75, 0x74, 0x79, 0x52, 0x06, 0x64, 0x75, 0x74, 0x69, 0x65, 0x73, 0x22, 0xaf, 0x01, 0x0a,
	0x04, 0x44, 0x75, 0x74, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x75, 0x62, 0x4b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x75, 0x62, 0x4b, 0x65, 0x79, 0x12, 0x12, 0x0a,
	0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x04, 0x73, 0x6c, 0x6f,
	0x74, 0x12, 0x3c, 0x0a, 0x0d, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73,
	0x61, 0x6c, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x48, 0x00,
	0x52, 0x0d, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x12,
	0x36, 0x0a, 0x0b, 0x41, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x41, 0x74, 0x74,
	0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x0b, 0x41, 0x74, 0x74, 0x65,
	0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x05, 0x0a, 0x03, 0x6a, 0x6f, 0x62, 0x22, 0x23,
	0x0a, 0x0d, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x12,
	0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72,
	0x6f, 0x6f, 0x74, 0x22, 0xc7, 0x01, 0x0a, 0x0b, 0x41, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x12, 0x35, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x41, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x68, 0x65, 0x63,
	0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x35,
	0x0a, 0x06, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x41, 0x74, 0x74, 0x65, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2e, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x52, 0x06, 0x74,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x1a, 0x36, 0x0a, 0x0a, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f,
	0x69, 0x6e, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x32, 0x55, 0x0a,
	0x10, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x12, 0x41, 0x0a, 0x0a, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x75, 0x74, 0x69, 0x65, 0x73, 0x12,
	0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x75, 0x74, 0x69,
	0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x75, 0x74, 0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x42, 0x18, 0x5a, 0x16, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_server_proto_structs_proto_rawDescOnce sync.Once
	file_internal_server_proto_structs_proto_rawDescData = file_internal_server_proto_structs_proto_rawDesc
)

func file_internal_server_proto_structs_proto_rawDescGZIP() []byte {
	file_internal_server_proto_structs_proto_rawDescOnce.Do(func() {
		file_internal_server_proto_structs_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_server_proto_structs_proto_rawDescData)
	})
	return file_internal_server_proto_structs_proto_rawDescData
}

var file_internal_server_proto_structs_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_internal_server_proto_structs_proto_goTypes = []interface{}{
	(*ListDutiesRequest)(nil),      // 0: proto.ListDutiesRequest
	(*ListDutiesResponse)(nil),     // 1: proto.ListDutiesResponse
	(*Duty)(nil),                   // 2: proto.Duty
	(*BlockProposal)(nil),          // 3: proto.BlockProposal
	(*Attestation)(nil),            // 4: proto.Attestation
	(*Attestation_Checkpoint)(nil), // 5: proto.Attestation.Checkpoint
}
var file_internal_server_proto_structs_proto_depIdxs = []int32{
	2, // 0: proto.ListDutiesResponse.duties:type_name -> proto.Duty
	3, // 1: proto.Duty.BlockProposal:type_name -> proto.BlockProposal
	4, // 2: proto.Duty.Attestation:type_name -> proto.Attestation
	5, // 3: proto.Attestation.source:type_name -> proto.Attestation.Checkpoint
	5, // 4: proto.Attestation.target:type_name -> proto.Attestation.Checkpoint
	0, // 5: proto.ValidatorService.ListDuties:input_type -> proto.ListDutiesRequest
	1, // 6: proto.ValidatorService.ListDuties:output_type -> proto.ListDutiesResponse
	6, // [6:7] is the sub-list for method output_type
	5, // [5:6] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_internal_server_proto_structs_proto_init() }
func file_internal_server_proto_structs_proto_init() {
	if File_internal_server_proto_structs_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_server_proto_structs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListDutiesRequest); i {
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
		file_internal_server_proto_structs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListDutiesResponse); i {
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
		file_internal_server_proto_structs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Duty); i {
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
		file_internal_server_proto_structs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BlockProposal); i {
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
		file_internal_server_proto_structs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Attestation); i {
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
		file_internal_server_proto_structs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Attestation_Checkpoint); i {
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
	file_internal_server_proto_structs_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*Duty_BlockProposal)(nil),
		(*Duty_Attestation)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_server_proto_structs_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_internal_server_proto_structs_proto_goTypes,
		DependencyIndexes: file_internal_server_proto_structs_proto_depIdxs,
		MessageInfos:      file_internal_server_proto_structs_proto_msgTypes,
	}.Build()
	File_internal_server_proto_structs_proto = out.File
	file_internal_server_proto_structs_proto_rawDesc = nil
	file_internal_server_proto_structs_proto_goTypes = nil
	file_internal_server_proto_structs_proto_depIdxs = nil
}
