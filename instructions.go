package rcache

import (
	"reflect"
	"sync"
)

type InstructionSet interface {
	// FieldName returns an override for the given field, the tag from the struct is passed as an argument.
	// An empty string return will use the field name from the struct.
	// Note: These are only used for the GetFieldByName call
	FieldName(tag string) string
	// TagNamespace returns the identifier of the tag, example would be `json:""`, "json" is the tag.
	TagNamespace() string
	// Skip instructs the parser to skip the field if its returned true, this will pass the tag through.
	// Example: if `json:"-"` was present, it would return true if this was a JSON parser.
	// This allows for custom logic to be implemented for skipping fields.
	// Note: all non-exported fields are skipped automatically.
	Skip(tag string) bool
	// GetMetadata is the constructor for the InstructionSet.
	// This is called for every field and allows you to programmatically set up the metadata for said field.
	GetMetadata(field reflect.StructField, tag string) InstructionSet
}

type FieldCache[Instructions InstructionSet] struct {
	mu       sync.RWMutex
	Idx      int
	fields   *[]*FieldCache[Instructions]
	fieldMap map[string]*FieldCache[Instructions]
	metadata Instructions
}

func (fc *FieldCache[Instructions]) Fields() []*FieldCache[Instructions] {
	if fc.fields == nil {
		return nil
	}
	return *fc.fields
}
func (fc *FieldCache[Instructions]) GetFieldByName(name string) *FieldCache[Instructions] {
	return fc.fieldMap[name]
}
func (fc *FieldCache[Instructions]) InstructionData() Instructions {
	return fc.metadata
}

type Cache[Metadata InstructionSet] struct {
	mu sync.RWMutex
	m  map[reflect.Type]*FieldCache[Metadata]
}

func NewCache[Metadata InstructionSet]() *Cache[Metadata] {
	return &Cache[Metadata]{
		m: make(map[reflect.Type]*FieldCache[Metadata]),
	}
}

func (rc *Cache[Metadata]) GetTypeDataFor(t reflect.Type) *FieldCache[Metadata] {
	var emptyMetadata Metadata
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	rc.mu.RLock()
	if cData, ok := rc.m[t]; ok {
		rc.mu.RUnlock()
		return cData
	}
	rc.mu.RUnlock()
	var tagNamespace = emptyMetadata.TagNamespace()
	var out = &FieldCache[Metadata]{
		fieldMap: map[string]*FieldCache[Metadata]{},
	}
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if out.fields == nil {
				tmp := make([]*FieldCache[Metadata], 0)
				out.fields = &tmp
			}
			f := t.Field(i)
			if len(f.Name) == 0 || (f.Name[0] >= 'a' && f.Name[0] <= 'z') {
				if !f.Anonymous {
					// not exported
					continue
				}
			}
			tag := f.Tag.Get(tagNamespace)
			if emptyMetadata.Skip(tag) {
				continue
			}
			var fieldName = emptyMetadata.FieldName(tag)
			if len(fieldName) == 0 {
				fieldName = f.Name
			}
			fType := f.Type
			if fType.Kind() == reflect.Ptr {
				fType = fType.Elem()
			}
			if fType == t {
				node := &FieldCache[Metadata]{
					Idx:    i,
					fields: out.fields,
				}
				*out.fields = append(*out.fields, node)
				out.fieldMap[fieldName] = node
				continue
			}
			// De-reference the return to not create side effects
			child := *rc.GetTypeDataFor(f.Type)
			child.Idx = i
			md := emptyMetadata.GetMetadata(f, tag)
			if md != nil {
				child.metadata = md.(Metadata)
			} else {
				child.metadata = emptyMetadata
			}
			*out.fields = append(*out.fields, &child)
			out.fieldMap[fieldName] = &child
		}
	case reflect.Array, reflect.Slice, reflect.Map:
		out = rc.GetTypeDataFor(t.Elem())
	default:
		out = &FieldCache[Metadata]{}
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.m[t] = out
	return out
}
