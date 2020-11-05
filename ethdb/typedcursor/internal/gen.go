package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"text/template"
)

type TemplateVariables struct {
	Type         string
	BucketType   string
	NilValue     string
	OnlyPositive bool
}

var Types = []TemplateVariables{
	{
		Type:         "uint64",
		BucketType:   "Uint64",
		NilValue:     "0",
		OnlyPositive: true,
	}, {
		Type:         "int",
		BucketType:   "Int",
		NilValue:     "0",
		OnlyPositive: false,
	},
}

func main() {
	buf := bytes.NewBuffer(nil)

	fmt.Fprintf(buf, `// Code generated by go generate; DO NOT EDIT.
package typedcursor

import (
	"bytes"
	"errors"

	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/ethdb/cbor"
)
`)

	for _, el := range Types {
		if err := typedcursorTemplate.Execute(buf, el); err != nil {
			panic(err)
		}
	}

	b, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("typed_gen.go", b, 0644); err != nil {
		panic(err)
	}
}

var typedcursorTemplate = template.Must(template.New("").Parse(`
type {{.BucketType}} struct {
	ethdb.Cursor
}

func New{{.BucketType}}(b ethdb.Cursor) *{{.BucketType}} {
	return &{{.BucketType}}{b}
}

func (b *{{.BucketType}}) Get(key []byte) ({{.Type}}, bool) {
	_, value, _ := b.Cursor.SeekExact(key)
	if value == nil {
		return 0, false
	}

	var v {{.Type}}
	decoder := cbor.Decoder(bytes.NewReader(value))
	defer cbor.Return(decoder)

	decoder.MustDecode(&v)
	return v, true
}

func (b *{{.BucketType}}) Increment(key []byte) error {
	v, _ := b.Get(key)
	return b.Put(key, v+1)
}

func (b *{{.BucketType}}) Decrement(key []byte) error {
	v, ok := b.Get(key)
	if !ok {
		// return ethdb.ErrNotFound
		return errors.New("not found key")
	}

	{{if .OnlyPositive}}	
	if v == 0 {
		return errors.New("could not decrement zero")
	}
	{{end}}

	return b.Put(key, v-1)
}

func (b *{{.BucketType}}) DecrementIfExist(key []byte) error {
	v, ok := b.Get(key)
	if !ok {
		return nil
	}

	{{if .OnlyPositive}}	
	if v == 0 {
		return errors.New("could not decrement zero")
	}
	{{end}}

	return b.Put(key, v-1)
}

func (b *{{.BucketType}}) Put(key []byte, value {{.Type}}) error {
	var buf bytes.Buffer

	encoder := cbor.Encoder(&buf)
	defer cbor.Return(encoder)

	encoder.MustEncode(&value)
	return b.Cursor.Put(key, buf.Bytes())
}

func (b *{{.BucketType}}) ForEach(fn func([]byte, {{.Type}}) error) error {
	return ethdb.ForEach(b.Cursor, func(k, v []byte) (bool, error) {
		var value {{.Type}}
		decoder := cbor.Decoder(bytes.NewReader(v))
		defer cbor.Return(decoder)

		decoder.MustDecode(&value)
		return true, fn(k, value)
	})
}
`))
