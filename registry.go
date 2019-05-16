// Copyright 2019 Denis Bernard <db047h@gmail.com>
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

type Registry struct {
	API         string
	Version     Version
	Tags        string
	Package     string
	CoreProfile bool
	Typedefs    []string
	Enums       []Enum
	Commands    []*Command
}

func decodeRegistry(r io.Reader) (*Registry, error) {
	var reg registry
	d := xml.NewDecoder(r)
	err := d.Decode(&reg)
	if err != nil {
		return nil, err
	}

	return &Registry{
		API:         api,
		Version:     version,
		Tags:        tags,
		Package:     pkgname,
		CoreProfile: coreProfile,
		Typedefs:    reg.Typedefs,
		Enums:       sortEnums(reg.Enums),
		Commands:    sortCommands(reg.Commands),
	}, nil
}

type registry struct {
	All struct {
		Enums    map[string]string
		Commands map[string]*Command
	}
	Typedefs []string
	Enums    map[string]string
	Commands map[string]*Command
}

func (r *registry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	r.All.Enums = make(map[string]string)
	r.All.Commands = make(map[string]*Command)
	r.Enums = make(map[string]string)
	r.Commands = make(map[string]*Command)

	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch t := t.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "types":
				if err = r.decodeTypes(d, &t); err != nil {
					return err
				}
			case "enums":
				if err = r.decodeEnums(d, &t); err != nil {
					return err
				}
			case "commands":
				if err = r.decodeCommands(d, &t); err != nil {
					return err
				}
			case "feature":
				if err = r.decodeFeature(d, &t); err != nil {
					return err
				}
			}
		case xml.EndElement:
		case xml.CharData:
		case xml.Comment:
		default:
			return fmt.Errorf("unexpected token type %T", t)
		}
	}
}

func (r *registry) decodeTypes(d *xml.Decoder, start *xml.StartElement) error {
	var td []byte
	inType := false
L:
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "type":
				inType = true
				for _, a := range t.Attr {
					if a.Name.Local == "name" && a.Value == "khrplatform" {
						inType = false
					}
				}
			case "apientry":
				td = append(td, "APIENTRY"...)
			}
		case xml.CharData:
			if inType {
				td = append(td, t...)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "type":
				if inType {
					r.Typedefs = append(r.Typedefs, convertTypedef(td))
					td = td[:0]
					inType = false
				}
			case "types":
				break L
			}
		}
	}
	return nil
}

// Remove dependency on khrplatform.h
//
func convertTypedef(td []byte) string {
	k := bytes.Index(td, []byte("khronos_"))
	if k < 0 {
		return string(td)
	}
	n := bytes.IndexByte(td[k:], '_') + 1 + k
	e := bytes.IndexByte(td[n:], ' ') + n
	subst := ""
	switch s := string(td[n:e]); s {
	case "int8_t", "uint8_t", "int16_t", "uint16_t", "int32_t", "uint32_t", "int64_t", "uint64_t", "intptr_t", "uintptr_t":
		subst = s
	case "float_t":
		subst = "float"
	case "ssize_t":
		subst = "intptr_t"
	case "usize_t":
		subst = "uintptr_t"
	default:
		panic("invalid type khronos_" + s)
	}
	copy(td[k:], subst)
	copy(td[k+len(subst):], td[e:])
	td = td[:len(td)-(e-k)+len(subst)]
	return string(td)
}

func (r *registry) decodeFeature(d *xml.Decoder, start *xml.StartElement) error {
	var ft struct {
		Require struct {
			Enums []struct {
				Name string `xml:"name,attr"`
			} `xml:"enum"`
			Cmds []struct {
				Name string `xml:"name,attr"`
			} `xml:"command"`
		} `xml:"require"`
		Remove []struct {
			Profile string `xml:"profile,attr"`
			Enums   []struct {
				Name string `xml:"name,attr"`
			} `xml:"enum"`
			Cmds []struct {
				Name string `xml:"name,attr"`
			} `xml:"command"`
		} `xml:"remove"`
	}
	var fv Version
	for _, a := range start.Attr {
		if a.Name.Local == "api" && a.Value != api {
			return d.Skip()
		}
		if a.Name.Local == "number" {
			fv.Set(a.Value)
			if version.Less(&fv) {
				return d.Skip()
			}
		}
	}
	err := d.DecodeElement(&ft, start)
	if err != nil {
		return err
	}
	for _, e := range ft.Require.Enums {
		v, ok := r.All.Enums[e.Name]
		if !ok {
			return fmt.Errorf("unknown enum %s in feature", e.Name)
		}
		r.Enums[e.Name] = v
	}
	for _, c := range ft.Require.Cmds {
		v, ok := r.All.Commands[c.Name]
		if !ok {
			return fmt.Errorf("unknown command %s in feature", c.Name)
		}
		v.Version = fv
		r.Commands[c.Name] = v
	}
	for i := range ft.Remove {
		if coreProfile && ft.Remove[i].Profile != "core" {
			continue
		}
		for _, e := range ft.Remove[i].Enums {
			delete(r.Enums, e.Name)
		}
		for _, c := range ft.Remove[i].Cmds {
			delete(r.Commands, c.Name)
		}
	}

	return nil
}

func (r *registry) decodeCommands(d *xml.Decoder, start *xml.StartElement) error {
	var cmds struct {
		Commands []*Command `xml:"command"`
	}
	err := d.DecodeElement(&cmds, start)
	if err != nil {
		return err
	}
	for _, c := range cmds.Commands {
		r.All.Commands[c.Name] = c
	}
	return nil
}

func (r *registry) decodeEnums(d *xml.Decoder, start *xml.StartElement) error {
	var es struct {
		Enums []struct {
			Name  string `xml:"name,attr"`
			Value string `xml:"value,attr"`
			API   string `xml:"api,attr"`
		} `xml:"enum"`
	}
	err := d.DecodeElement(&es, start)
	if err != nil {
		return err
	}
	for _, e := range es.Enums {
		if e.API != "" && e.API != api {
			continue
		}
		if _, ok := r.All.Enums[e.Name]; ok {
			return fmt.Errorf("duplicate enum %s", e.Name)
		}
		r.All.Enums[e.Name] = e.Value
	}
	return nil
}
