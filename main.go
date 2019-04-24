// Copyright 2019 Denis Bernard <db047h@gmail.com>
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:generate go-bindata templates

package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

var (
	api            = "gl"
	version        Version
	versionGLES    Version
	profile        string
	tags           string
	pkgname        string
	forceRegUpdate bool
	verbose        bool
)

func main() {
	var out string

	flag.Var(&version, "gl", "OpenGL api `version` (default: 3.1)")
	// flag.Var(&version, "gles", "OpenGLES api `version` (default: 2.0)")
	flag.StringVar(&profile, "profile", "", "default profile")
	flag.StringVar(&pkgname, "p", "gl", "package `name`")
	flag.StringVar(&out, "o", "", "output `directory`")
	flag.BoolVar(&forceRegUpdate, "f", false, "force update of gl.xml")
	flag.BoolVar(&verbose, "v", false, "verbose output")

	flag.Parse()

	if version.Major == 0 && version.Minor == 0 {
		version.Set("3.1")
	}
	if versionGLES.Major == 0 && versionGLES.Minor == 0 {
		versionGLES.Set("2.0")
	}

	regXML, err := getRegistry(forceRegUpdate)
	if err != nil {
		panic(err)
	}
	if verbose {
		log.Print("Parsing gl.xml (GL)")
	}
	r, err := decodeRegistry(bytes.NewReader(regXML))
	if err != nil {
		panic(err)
	}

	fmap := template.FuncMap{"ToUpper": strings.ToUpper, "NewVersion": NewVersion}

	fname := "header.tmpl"
	of := filepath.Join(out, "gl.h")
	if verbose {
		log.Printf("Generating %s", of)
	}
	asset, err := Asset("templates/" + fname)
	if err != nil {
		panic(err)
	}
	t, err := template.New(fname).Funcs(fmap).Parse(string(asset))
	if err != nil {
		panic(err)
	}
	o, err := os.Create(of)
	if err != nil {
		panic(err)
	}
	err = t.Execute(o, r)
	o.Close()
	if err != nil {
		panic(err)
	}

	fname = "gl.tmpl"
	of = filepath.Join(out, "gl.go")
	if verbose {
		log.Printf("Generating %s", of)
	}
	asset, err = Asset("templates/" + fname)
	if err != nil {
		panic(err)
	}
	t, err = template.New(fname).Funcs(fmap).Parse(string(asset))
	if err != nil {
		panic(err)
	}
	o, err = os.Create(of)
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
	version = versionGLES
	profile = ""
	tags = "gles2,!darwin"
	if verbose {
		log.Print("Parsing gl.xml (GLES)")
	}
	r, err = decodeRegistry(bytes.NewReader(regXML))
	if err != nil {
		panic(err)
	}

	fname = "gles2.tmpl"
	of = filepath.Join(out, "gles2.go")
	if verbose {
		log.Printf("Generating %s", of)
	}
	asset, err = Asset("templates/" + fname)
	if err != nil {
		panic(err)
	}
	t, err = template.New(fname).Funcs(fmap).Parse(string(asset))
	if err != nil {
		panic(err)
	}
	o, err = os.Create(of)
	if err != nil {
		panic(err)
	}
	err = t.Execute(o, r)
	o.Close()
	if err != nil {
		panic(err)
	}
}

const regUrl = "https://raw.githubusercontent.com/KhronosGroup/OpenGL-Registry/master/xml/gl.xml"

func getRegistry(forceFetch bool) ([]byte, error) {
	c, err := os.UserCacheDir()
	if err != nil {
		return fetchRegistry()
	}
	// create gogl dir
	c = filepath.Join(c, "gogl")
	err = os.MkdirAll(c, 0777)
	if err != nil && !os.IsExist(err) {
		return fetchRegistry()
	}
	c = filepath.Join(c, "gl.xml")
	fi, err := os.Stat(c)
	if err != nil && !os.IsNotExist(err) {
		return fetchRegistry()
	}
	if !forceFetch && err == nil {
		if verbose {
			log.Printf("Using cached registry file %s; last updated: %v", c, fi.ModTime().Format(time.RFC1123))
			log.Printf("Use the -f switch to force an update")
		}
		return ioutil.ReadFile(c)
	}
	// write cache
	data, err := fetchRegistry()
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(c, data, 0666)
	return data, nil
}

func fetchRegistry() ([]byte, error) {
	if verbose {
		log.Printf("Fetching OpenGL registry from %s", regUrl)
	}
	resp, err := http.Get(regUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
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
