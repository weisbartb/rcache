# Reflection Cache

## Purpose

This library provides caching functionality along with parsing instructions for reflected structs.
The goal is to allow for parse once, call multiple times object caching for tag metadata.
Any metadata instruction can be captured, interpreted, and recalled as needed.
This was inspired after writing multiple object caches to pull tag data and cache their tag structure.

## Usage

```go

var cache = NewCache[InstructionSet](md InstructionSet)
md := cache.GetTypeDataFor(reflect.TypeOf(myStruct{}))
```

## InstructionSet interface

```go
type InstructionSet interface {
FieldName(tag string) string
TagNamespace() string
Skip(tag string) bool
GetMetadata(fieldType reflect.Type, tag string) InstructionSet
}
```

### FieldName

This allows for the field name to be overridden for a given struct. Examples of this would be the `json:"myFieldName"`.

### TagNamespace

This allows the tag to be set `json:"someField"`, `json` is the tag namespace.

### Skip

Allows this field to be skipped by the parser, by default all non-exported fields are automatically skipped.

### GetMetadata

This creates a new instance of the InstructionSet with any values set based on the tags.

## Notes

### Not every field is in the cache

Skipped fields do not appear in the `Fields()` call.
`Field()[x].Idx` will give you the struct index for the field. This is the value to use
with `reflection.ValueOf(x).Field(Idx)`. If your tag is going to cause field mutation, it is advised
to memoize the mutation function with any static parsing done when the `GetMetadata` call is made.

### Slices, Arrays, Maps are supported, but only operate on elements

Only the element for the given collections is cacheable for the collection.
Keys for maps (regardless of types) are not cachable and will not show up.