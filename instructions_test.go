package rcache_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/weisbartb/rcache"
)

type embeddedStruct struct {
	a string
	B string `redact:"admin"`
	C string `redact:"-"`
}

type simpleStruct struct {
	a string
	B string `redact:"admin"`
	C string `redact:"-"`
	embeddedStruct
}

type complexStruct struct {
	embeddedStruct
	EM  embeddedStruct `redact:"admin"`
	EM2 embeddedStruct `redact:"-"`
}

type RecursiveStruct struct {
	R *RecursiveStruct `redact:"admin"`
	A string           `redact:"admin"`
}

type InstructionSet struct {
	AllowedGroups []string
}

func Map[V any, R any](a []V, f func(V) R) []R {
	r := make([]R, len(a))
	for k, v := range a {
		r[k] = f(v)
	}
	return r
}
func (i InstructionSet) TagNamespace() string {
	const tag = "redact"
	return tag
}

func (i InstructionSet) FieldName(tag string) string {
	return strings.SplitN(tag, ",", 2)[0]
}
func (i InstructionSet) Skip(tag string) bool {
	return tag == "-"
}

func (i InstructionSet) GetMetadata(fieldType reflect.StructField, tag string) rcache.InstructionSet {
	tagParts := strings.Split(tag, ",")
	if tagParts[0] == "-" {
		// skip
		return nil
	}
	var allowedGroups []string
	if len(tagParts[0]) > 0 {
		allowedGroups = Map(tagParts, func(v string) string {
			return strings.ToLower(strings.TrimSpace(v))
		})
	}
	return InstructionSet{AllowedGroups: allowedGroups}
}

func TestReflectionCache_GetTypeDataFor(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := rcache.NewCache(InstructionSet{})
		r := c.GetTypeDataFor(reflect.TypeOf(simpleStruct{}))
		require.NotEmpty(t, r)
		require.Len(t, r.Fields(), 2)
		require.Len(t, r.Fields()[0].InstructionData().AllowedGroups, 1)
		require.Len(t, r.Fields()[1].Fields(), 1)
		require.Len(t, r.Fields()[1].Fields()[0].InstructionData().AllowedGroups, 1)
	})
	t.Run("recursive", func(t *testing.T) {
		c := rcache.NewCache(InstructionSet{})
		r := c.GetTypeDataFor(reflect.TypeOf(RecursiveStruct{}))
		require.Equal(t, r.Fields(), r.Fields()[0].Fields())
		require.NotEmpty(t, r)
	})
	t.Run("complex", func(t *testing.T) {
		c := rcache.NewCache(InstructionSet{})
		r := c.GetTypeDataFor(reflect.TypeOf(complexStruct{}))
		require.NotEmpty(t, r)
		require.Len(t, r.Fields(), 2)
		require.Equal(t, 0, r.Fields()[0].Idx)
		require.Equal(t, 1, r.Fields()[1].Idx)
		require.Equal(t, r.Fields()[0].Fields(), r.Fields()[1].Fields())
	})

}
