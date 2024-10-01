// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package functional

func MapFn[T any, U any](fn func(T) U, a []T) []U {
	s := make([]U, len(a))
	for i := 0; i < len(a); i++ {
		s[i] = fn(a[i])
	}
	return s
}

func RejectKVs[T comparable, U any](fn func(T, U) bool, a map[T]U) map[T]U {
	s := map[T]U{}
	for k, v := range a {
		if !fn(k, v) {
			s[k] = v
		}
	}
	return s
}

func MapKV[T comparable, U, V any](fn func(T, U) V, a map[T]U) []V {
	s := []V{}
	for k, v := range a {
		s = append(s, fn(k, v))
	}
	return s
}

func Any[T any](fn func(T) bool, a []T) bool {
	for _, v := range a {
		if fn(v) {
			return true
		}
	}
	return false
}

func All[T any](fn func(T) bool, a []T) bool {
	for _, v := range a {
		if !fn(v) {
			return false
		}
	}
	return true
}

func Compact[T any](a []*T) []*T {
	s := []*T{}
	for _, v := range a {
		if v == nil {
			continue
		}
		s = append(s, v)
	}
	return s
}
