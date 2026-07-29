package entity

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"
)

type Field struct {
	GoName   string
	DBName   string
	Label    string
	Order    int
	List     bool
	Search   bool
	ReadOnly bool
}

type KeyKind int

const (
	KeyPrimary KeyKind = iota
	KeyExternal
)

type Key struct {
	Kind   KeyKind
	Fields []*Field
}

func (k Key) IsComposite() bool {
	return len(k.Fields) > 1
}

type Descriptor struct {
	Type   reflect.Type
	Fields []Field
	Keys   []Key
}

func (d *Descriptor) PrimaryKey() *Key {
	for i := range d.Keys {
		if d.Keys[i].Kind == KeyPrimary {
			return &d.Keys[i]
		}
	}
	return nil
}

func (d *Descriptor) ExternalKey() *Key {
	for i := range d.Keys {
		if d.Keys[i].Kind == KeyExternal {
			return &d.Keys[i]
		}
	}
	return nil
}

func (d *Descriptor) ListFields() []Field {
	var out []Field
	for _, f := range d.Fields {
		if f.List {
			out = append(out, f)
		}
	}
	return out
}

func (d *Descriptor) SearchFields() []Field {
	var out []Field
	for _, f := range d.Fields {
		if f.Search {
			out = append(out, f)
		}
	}
	return out
}

var (
	registry   = make(map[reflect.Type]*Descriptor)
	registryMu sync.Mutex
)

type keyDef struct {
	kind   KeyKind
	fields []string
}

type registerConfig struct {
	keys []keyDef
}

type RegisterOption func(cfg *registerConfig)

func PrimaryKey(field string) RegisterOption {
	return func(cfg *registerConfig) {
		cfg.keys = append(cfg.keys, keyDef{kind: KeyPrimary, fields: []string{field}})
	}
}

func ExternalKey(fields ...string) RegisterOption {
	return func(cfg *registerConfig) {
		cfg.keys = append(cfg.keys, keyDef{kind: KeyExternal, fields: fields})
	}
}

func Register[T any](opts ...RegisterOption) *Descriptor {
	var t T
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("entity.Register: %s is not a struct", typ))
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if cached, ok := registry[typ]; ok {
		return cached
	}

	var cfg registerConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var fields []Field
	seenOrders := make(map[int]string)
	fieldByName := make(map[string]int)

	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}

		dbTag := f.Tag.Get("db")
		readOnly := f.Tag.Get("readonly") == "true"

		if dbTag == "" && !readOnly {
			continue
		}

		label := f.Tag.Get("label")

		order := 999
		if orderStr := f.Tag.Get("order"); orderStr != "" {
			v, err := strconv.Atoi(orderStr)
			if err != nil {
				panic(fmt.Sprintf("entity: invalid order tag %q in %s.%s", orderStr, typ.Name(), f.Name))
			}
			order = v
		}

		if existing, ok := seenOrders[order]; ok {
			panic(fmt.Sprintf("entity: duplicate order value %d for fields %s and %s in %s", order, existing, f.Name, typ.Name()))
		}
		seenOrders[order] = f.Name

		fieldByName[f.Name] = len(fields)
		fields = append(fields, Field{
			GoName:   f.Name,
			DBName:   dbTag,
			Label:    label,
			Order:    order,
			List:     f.Tag.Get("list") == "true",
			Search:   f.Tag.Get("search") == "true",
			ReadOnly: readOnly,
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Order < fields[j].Order
	})

	// rebuild fieldByName after sort
	for i := range fields {
		fieldByName[fields[i].GoName] = i
	}

	// validate and resolve keys
	var primaryCount int
	var keys []Key

	for _, kd := range cfg.keys {
		if len(kd.fields) == 0 {
			panic(fmt.Sprintf("entity: key with empty fields in %s", typ.Name()))
		}

		switch kd.kind {
		case KeyPrimary:
			primaryCount++
			if len(kd.fields) != 1 {
				panic(fmt.Sprintf("entity: PrimaryKey must have exactly one field in %s", typ.Name()))
			}
		case KeyExternal:
			if len(kd.fields) < 1 {
				panic(fmt.Sprintf("entity: ExternalKey must have at least one field in %s", typ.Name()))
			}
		}

		var resolved []*Field
		for _, name := range kd.fields {
			idx, ok := fieldByName[name]
			if !ok {
				panic(fmt.Sprintf("entity: key field %q not found in %s", name, typ.Name()))
			}
			resolved = append(resolved, &fields[idx])
		}

		keys = append(keys, Key{
			Kind:   kd.kind,
			Fields: resolved,
		})
	}

	if primaryCount == 0 {
		panic(fmt.Sprintf("entity: %s has no PrimaryKey", typ.Name()))
	}
	if primaryCount > 1 {
		panic(fmt.Sprintf("entity: %s has multiple PrimaryKeys", typ.Name()))
	}

	desc := &Descriptor{
		Type:   typ,
		Fields: fields,
		Keys:   keys,
	}
	registry[typ] = desc
	return desc
}
