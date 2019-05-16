// Copyright 2019 Denis Bernard <db047h@gmail.com>
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
}

func NewVersion(major, minor int) *Version {
	return &Version{major, minor}
}

func (v *Version) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor)
}

func (v *Version) Get() interface{} {
	return *v
}

func (v *Version) Set(s string) error {
	end := strings.IndexRune(s, '.')
	if end < 0 {
		end = len(s)
	}

	n, err := strconv.ParseInt(s[:end], 10, 32)
	if err != nil {
		return err
	}
	v.Major = int(n)
	if end >= len(s) {
		v.Minor = 0
		return nil
	}
	n, err = strconv.ParseInt(s[end+1:], 10, 32)
	if err != nil {
		return err
	}
	v.Minor = int(n)
	return nil
}

// Less returns true if v < rhs
func (v *Version) Less(rhs *Version) bool {
	if v.Major == rhs.Major {
		return v.Minor < rhs.Minor
	}
	return v.Major < rhs.Major
}
