// Copyright (C) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package packer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	validatorsetsig "github.com/ava-labs/teleporter/abi-bindings/go/governance/ValidatorSetSig"
	teleportermessenger "github.com/ava-labs/teleporter/abi-bindings/go/teleporter/TeleporterMessenger"
	teleporterregistry "github.com/ava-labs/teleporter/abi-bindings/go/teleporter/registry/TeleporterRegistry"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

const interfaceName = "ABIPacker"

// Paths below are relative to the current package.
const packagePath = "../..."
const currentPackagePath = "."

var fs = token.NewFileSet()

var packerTypes = map[string]ABIPacker{
	"ValidatorSetSigMessage": &validatorsetsig.ValidatorSetSigMessage{},
	"TeleporterMessage":      &teleportermessenger.TeleporterMessage{},
	"ProtocolRegistryEntry":  &teleporterregistry.ProtocolRegistryEntry{},
}

func findAllImplementers(t *testing.T) []string {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Fset: fs,
	}
	interfacePkg, err := packages.Load(cfg, currentPackagePath)
	require.NoError(t, err)

	var iface *types.Interface
	// find the interface in the currnet package
	for _, pkg := range interfacePkg {
		for _, obj := range pkg.Types.Scope().Names() {
			if obj == interfaceName {
				ifaceObj := pkg.Types.Scope().Lookup(obj)
				if ifaceObj == nil {
					continue
				}
				if typ, ok := ifaceObj.Type().Underlying().(*types.Interface); ok {
					iface = typ
					break
				}
			}
		}
	}
	require.NotNilf(t, iface, "Interface %s not found in package %s", interfaceName, currentPackagePath)

	pkgs, err := packages.Load(cfg, packagePath)
	require.NoError(t, err)

	allImplementers := []string{}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			// Inspect types in the package
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range genDecl.Specs {
					// We have a Type spec
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					// Find object matching the TypeSpec name in the package scope
					obj := pkg.Types.Scope().Lookup(typeSpec.Name.Name)
					if obj == nil {
						continue
					}
					// Check if the typed object we have is a struct
					_, ok = obj.Type().Underlying().(*types.Struct)
					if !ok {
						continue
					}
					// Lookup the object by name and cast it to types.Named
					typeName := typeSpec.Name.Name
					typeObj := pkg.Types.Scope().Lookup(typeName)
					if typeObj == nil {
						continue
					}
					named, ok := typeObj.Type().(*types.Named)
					if !ok {
						continue
					}
					// Finally check if the named struct we have implements the interface
					if types.Implements(types.NewPointer(named), iface) {
						allImplementers = append(allImplementers, named.Obj().Name())
					}
				}
			}
		}
	}
	return allImplementers
}

func TestExhaustivePacking(t *testing.T) {
	implementers := findAllImplementers(t)
	for _, structName := range implementers {
		packerType, ok := packerTypes[structName]
		require.True(
			t,
			ok,
			fmt.Sprintf("Struct %s not found in packer_test.go packerTypes map. Add it to check for exhaustiveness", structName),
		)

		randomizedStruct, err := randomizeStruct(packerType)
		require.NoError(t, err)

		v1 := randomizedStruct.(ABIPacker)
		encoded, err := v1.Pack()
		require.NoError(t, err)
		v2 := reflect.New(reflect.TypeOf(v1).Elem()).Interface().(ABIPacker)
		err = v2.Unpack(encoded)
		require.NoError(t, err)
		require.Equal(t, v1, v2)
	}
}

func randomizeStruct(packerStruct interface{}) (interface{}, error) {
	v := reflect.ValueOf(packerStruct).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		err := randomizeField(field)
		if err != nil {
			return nil, err
		}
	}
	return packerStruct, nil
}

func randomizeField(field reflect.Value) error {
	fieldType := field.Type()
	switch fieldType.Kind() {
	// The list only includes types supported by abigen
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(rand.Int63())
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(rand.Uint64())
	case reflect.Bool:
		field.SetBool(rand.Intn(2) == 1)
	case reflect.String:
		field.SetString(randomString(rand.Intn(256)))
	case reflect.Slice:
		n := rand.Intn(256)
		field.Set(reflect.MakeSlice(fieldType, n, n))
		for i := 0; i < field.Len(); i++ {
			err := randomizeField(field.Index(i))
			if err != nil {
				return fmt.Errorf("Error randomizing slice: %v", err)
			}
		}
	case reflect.Struct:
		randomizeStruct(field.Addr().Interface())
	case reflect.Array:
		elemType := fieldType.Elem()
		// arrays other than byte arrays get converted to slices by abigen
		if elemType.Kind() != reflect.Uint8 {
			return fmt.Errorf("Unsupported array type %s. Only byte arrays are supported.", elemType.Kind().String())
		}
		for j := 0; j < field.Len(); j++ {
			field.Index(j).SetUint(uint64(rand.Intn(256)))
		}

	case reflect.Pointer:
		if fieldType.Elem() == reflect.TypeOf(big.Int{}) {
			newBigInt := big.NewInt(rand.Int63())
			field.Set(reflect.ValueOf(newBigInt))
		} else {
			return fmt.Errorf("Unsupported pointer type %s", fieldType.Elem().String())
		}
	default:
		return fmt.Errorf("Unsupported type %s", fieldType.Kind().String())
	}
	return nil
}

// randomString generates a random string of the given length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
