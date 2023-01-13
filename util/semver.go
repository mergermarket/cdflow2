package util

import (
	"strconv"
)

type Semver struct {
	Major int
	Minor int
	Patch int
}

func ParseSemver(v string) (s Semver, ok bool) {
	if v == "" {
		return
	}

	if v[0] == 'v' {
		v = v[1:]
	}

	s.Major, v, ok = parseInt(v)
	if !ok {
		return
	}

	if v == "" {
		s.Minor = 0
		s.Patch = 0
		return
	}

	if v[0] != '.' {
		ok = false
		return
	}

	s.Minor, v, ok = parseInt(v[1:])
	if !ok {
		return
	}

	if v == "" {
		s.Patch = 0
		return
	}

	if v[0] != '.' {
		ok = false
		return
	}

	s.Patch, v, ok = parseInt(v[1:])
	if !ok {
		return
	}

	return
}

func parseInt(v string) (t int, rest string, ok bool) {
	if v == "" {
		return
	}

	if v[0] < '0' || '9' < v[0] {
		return
	}

	i := 1
	for i < len(v) && '0' <= v[i] && v[i] <= '9' {
		i++
	}

	if v[0] == '0' && i != 1 {
		return
	}

	asInt, err := strconv.Atoi(v[:i])
	if err != nil {
		return
	}

	return asInt, v[i:], true
}
