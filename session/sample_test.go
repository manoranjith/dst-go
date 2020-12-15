package session_test

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/hyperledger-labs/perun-node/session"
)

func TestEditSample(t *testing.T) {
	s := session.NewSample()
	fmt.Printf("\nInitial value: %+v\nField a:%s\n", s, GetUnexportedField(s, "a"))
	SetUnexportedField(s, "a", "new")
	fmt.Printf("\nModified value: %+v\nField a:%s\n", s, GetUnexportedField(s, "a"))
}

// GetUnexportedField returns the string corresponding to the field name in the given data.
func GetUnexportedField(data interface{}, name string) interface{} {
	value := reflect.ValueOf(data)
	field := value.Elem().FieldByName(name)
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func SetUnexportedField(data interface{}, name string, value interface{}) {
	field := reflect.ValueOf(data).Elem().FieldByName(name)
	reflect.NewAt(field.Type(),
		unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}
