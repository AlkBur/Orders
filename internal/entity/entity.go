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

type Descriptor struct {
	Type   reflect.Type
	Fields []Field
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

func Register[T any]() *Descriptor {
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

	var fields []Field
	seenOrders := make(map[int]string)

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

	desc := &Descriptor{
		Type:   typ,
		Fields: fields,
	}
	registry[typ] = desc
	return desc
}
