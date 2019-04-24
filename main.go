// Copyright 2019 Denis Bernard <db047h@gmail.com>
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:generate mkdir -p registry
//go:generate curl -L --compressed -o registry/gl.xml https://raw.githubusercontent.com/KhronosGroup/OpenGL-Registry/master/xml/gl.xml
//go:generate curl -L --compressed -o registry/khrplatform.h https://raw.githubusercontent.com/KhronosGroup/EGL-Registry/master/api/KHR/khrplatform.h
//go:generate go-bindata registry templates

package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

var (
	api     = "gl"
	version Version
	profile string
	tags    string
	pkgname string
)

func main() {
	var (
		out string
		in  = "registry/gl.xml"
	)

	flag.Var(&version, "v", "OpenGL api `version` (default: 3.1)")
	flag.StringVar(&profile, "profile", "", "default profile")
	flag.StringVar(&pkgname, "p", "gl", "package `name`")
	flag.StringVar(&out, "o", "", "output `directory`")

	flag.Parse()

	if version.Major == 0 && version.Minor == 0 {
		version.Set("3.1")
	}

	regXML, err := Asset(in)
	if err != nil {
		panic(err)
	}
	r, err := decodeRegistry(bytes.NewReader(regXML))
	if err != nil {
		panic(err)
	}

	asset, err := Asset("registry/khrplatform.h")
	if err != nil {
		panic(err)
	}
	if err = ioutil.WriteFile(filepath.Join(out, "khrplatform.h"), asset, 0666); err != nil {
		panic(err)
	}

	fmap := template.FuncMap{"ToUpper": strings.ToUpper, "NewVersion": NewVersion}

	fname := "header.tmpl"
	asset, err = Asset("templates/" + fname)
	if err != nil {
		panic(err)
	}
	t, err := template.New(fname).Funcs(fmap).Parse(string(asset))
	if err != nil {
		panic(err)
	}
	o, err := os.Create(filepath.Join(out, "gl.h"))
	if err != nil {
		panic(err)
	}
	err = t.Execute(o, r)
	o.Close()
	if err != nil {
		panic(err)
	}

	fname = "gl.tmpl"
	asset, err = Asset("templates/" + fname)
	if err != nil {
		panic(err)
	}
	t, err = template.New(fname).Funcs(fmap).Parse(string(asset))
	if err != nil {
		panic(err)
	}
	o, err = os.Create(filepath.Join(out, "gl.go"))
	if err != nil {
		panic(err)
	}
	r.Tags = "!gles2 darwin"
	err = t.Execute(o, r)
	o.Close()
	if err != nil {
		panic(err)
	}

	api = "gles2"
	version.Set("2.0")
	profile = ""
	tags = "gles2,!darwin"
	r, err = decodeRegistry(bytes.NewReader(regXML))
	if err != nil {
		panic(err)
	}

	fname = "gles2.tmpl"
	asset, err = Asset("templates/" + fname)
	if err != nil {
		panic(err)
	}
	t, err = template.New(fname).Funcs(fmap).Parse(string(asset))
	if err != nil {
		panic(err)
	}
	o, err = os.Create(filepath.Join(out, "gles2.go"))
	if err != nil {
		panic(err)
	}
	err = t.Execute(o, r)
	o.Close()
	if err != nil {
		panic(err)
	}
}

type Command struct {
	Type    Type
	Name    string
	Params  []Param
	Version Version
}

type Param struct {
	Type Type
	Name string
}

func (c *Command) GoName() string {
	if strings.HasPrefix(c.Name, "gl") {
		return strings.ToUpper(c.Name[2:3]) + c.Name[3:]
	}
	return c.Name
}

func (c *Command) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var xc struct {
		Proto struct {
			Type string `xml:"ptype"`
			Ptr  string `xml:",chardata"`
			Name string `xml:"name"`
		} `xml:"proto"`
		Params []struct {
			Type string `xml:"ptype"`
			Ptr  string `xml:",chardata"`
			Name string `xml:"name"`
		} `xml:"param"`
	}
	err := d.DecodeElement(&xc, &start)
	if err != nil {
		return err
	}

	c.Type = MkType(xc.Proto.Type, xc.Proto.Ptr)
	c.Name = xc.Proto.Name
	c.Params = make([]Param, 0, len(xc.Params))
	for _, xp := range xc.Params {
		if xp.Type == "" {
			xp.Type = "void"
		}
		c.Params = append(c.Params, Param{
			Name: paramName(xp.Name),
			Type: MkType(xp.Type, xp.Ptr),
		})
	}
	return nil
}

var reserved = map[string]struct{}{
	"int":        {},
	"int8":       {},
	"int16":      {},
	"int32":      {},
	"uint32":     {},
	"uint":       {},
	"uint8":      {},
	"uint16":     {},
	"int64":      {},
	"uint64":     {},
	"uintptr":    {},
	"float32":    {},
	"float64":    {},
	"complex64":  {},
	"complex128": {},
	"string":     {},
	"byte":       {},
	"rune":       {},
	"bool":       {},

	"break":       {},
	"default":     {},
	"func":        {},
	"interface":   {},
	"select":      {},
	"case":        {},
	"defer":       {},
	"go":          {},
	"map":         {},
	"struct":      {},
	"chan":        {},
	"else":        {},
	"goto":        {},
	"package":     {},
	"switch":      {},
	"const":       {},
	"fallthrough": {},
	"if":          {},
	"range":       {},
	"type":        {},
	"continue":    {},
	"for":         {},
	"import":      {},
	"return":      {},
	"var":         {},
}

func paramName(n string) string {
	if _, ok := reserved[n]; ok {
		return n + "_"
	}
	return n
}

type Enum struct {
	Name  string
	Value string
}

func (e *Enum) GoName() string {
	if strings.HasPrefix(e.Name, "GL_") {
		return e.Name[3:]
	}
	return e.Name
}

func sortEnums(em map[string]string) []Enum {
	enums := make([]Enum, 0, len(em))
	for k, v := range em {
		enums = append(enums, Enum{k, v})
	}
	sort.Slice(enums, func(i, j int) bool { return enums[i].Name < enums[j].Name })
	return enums
}

func sortCommands(cm map[string]*Command) []*Command {
	cmds := make([]*Command, 0, len(cm))
	for _, v := range cm {
		cmds = append(cmds, v)
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Version.Less(&cmds[j].Version) || (!cmds[j].Version.Less(&cmds[i].Version)) && cmds[i].Name < cmds[j].Name
	})
	return cmds
}
