// Copyright (c) Mysten Labs, Inc.
// SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: sui/rpc/v2beta/signature_scheme.proto

package v2beta

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Flag use to disambiguate the signature schemes supported by Sui.
//
// Note: the enum values defined by this proto message exactly match their
// expected BCS serialized values when serialized as a u8. See
// [enum.SignatureScheme](https://mystenlabs.github.io/sui-rust-sdk/sui_sdk_types/enum.SignatureScheme.html)
// for more information about signature schemes.
type SignatureScheme int32

const (
	SignatureScheme_ED25519   SignatureScheme = 0
	SignatureScheme_SECP256K1 SignatureScheme = 1
	SignatureScheme_SECP256R1 SignatureScheme = 2
	SignatureScheme_MULTISIG  SignatureScheme = 3
	SignatureScheme_BLS12381  SignatureScheme = 4
	SignatureScheme_ZKLOGIN   SignatureScheme = 5
	SignatureScheme_PASSKEY   SignatureScheme = 6
)

// Enum value maps for SignatureScheme.
var (
	SignatureScheme_name = map[int32]string{
		0: "ED25519",
		1: "SECP256K1",
		2: "SECP256R1",
		3: "MULTISIG",
		4: "BLS12381",
		5: "ZKLOGIN",
		6: "PASSKEY",
	}
	SignatureScheme_value = map[string]int32{
		"ED25519":   0,
		"SECP256K1": 1,
		"SECP256R1": 2,
		"MULTISIG":  3,
		"BLS12381":  4,
		"ZKLOGIN":   5,
		"PASSKEY":   6,
	}
)

func (x SignatureScheme) Enum() *SignatureScheme {
	p := new(SignatureScheme)
	*p = x
	return p
}

func (x SignatureScheme) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SignatureScheme) Descriptor() protoreflect.EnumDescriptor {
	return file_sui_rpc_v2beta_signature_scheme_proto_enumTypes[0].Descriptor()
}

func (SignatureScheme) Type() protoreflect.EnumType {
	return &file_sui_rpc_v2beta_signature_scheme_proto_enumTypes[0]
}

func (x SignatureScheme) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use SignatureScheme.Descriptor instead.
func (SignatureScheme) EnumDescriptor() ([]byte, []int) {
	return file_sui_rpc_v2beta_signature_scheme_proto_rawDescGZIP(), []int{0}
}

var File_sui_rpc_v2beta_signature_scheme_proto protoreflect.FileDescriptor

const file_sui_rpc_v2beta_signature_scheme_proto_rawDesc = "" +
	"\n" +
	"%sui/rpc/v2beta/signature_scheme.proto\x12\x0esui.rpc.v2beta*r\n" +
	"\x0fSignatureScheme\x12\v\n" +
	"\aED25519\x10\x00\x12\r\n" +
	"\tSECP256K1\x10\x01\x12\r\n" +
	"\tSECP256R1\x10\x02\x12\f\n" +
	"\bMULTISIG\x10\x03\x12\f\n" +
	"\bBLS12381\x10\x04\x12\v\n" +
	"\aZKLOGIN\x10\x05\x12\v\n" +
	"\aPASSKEY\x10\x06B\x10Z\x0esui/rpc/v2betab\x06proto3"

var (
	file_sui_rpc_v2beta_signature_scheme_proto_rawDescOnce sync.Once
	file_sui_rpc_v2beta_signature_scheme_proto_rawDescData []byte
)

func file_sui_rpc_v2beta_signature_scheme_proto_rawDescGZIP() []byte {
	file_sui_rpc_v2beta_signature_scheme_proto_rawDescOnce.Do(func() {
		file_sui_rpc_v2beta_signature_scheme_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_sui_rpc_v2beta_signature_scheme_proto_rawDesc), len(file_sui_rpc_v2beta_signature_scheme_proto_rawDesc)))
	})
	return file_sui_rpc_v2beta_signature_scheme_proto_rawDescData
}

var file_sui_rpc_v2beta_signature_scheme_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_sui_rpc_v2beta_signature_scheme_proto_goTypes = []any{
	(SignatureScheme)(0), // 0: sui.rpc.v2beta.SignatureScheme
}
var file_sui_rpc_v2beta_signature_scheme_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_sui_rpc_v2beta_signature_scheme_proto_init() }
func file_sui_rpc_v2beta_signature_scheme_proto_init() {
	if File_sui_rpc_v2beta_signature_scheme_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_sui_rpc_v2beta_signature_scheme_proto_rawDesc), len(file_sui_rpc_v2beta_signature_scheme_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_sui_rpc_v2beta_signature_scheme_proto_goTypes,
		DependencyIndexes: file_sui_rpc_v2beta_signature_scheme_proto_depIdxs,
		EnumInfos:         file_sui_rpc_v2beta_signature_scheme_proto_enumTypes,
	}.Build()
	File_sui_rpc_v2beta_signature_scheme_proto = out.File
	file_sui_rpc_v2beta_signature_scheme_proto_goTypes = nil
	file_sui_rpc_v2beta_signature_scheme_proto_depIdxs = nil
}
