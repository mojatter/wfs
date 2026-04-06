package memfs

import (
	"fmt"
	"io/fs"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestValue(t *testing.T) {
	v := &value{
		name:    "path/to/name",
		data:    []byte(`content`),
		mode:    fs.ModePerm,
		modTime: time.Now(),
	}
	if name := v.Name(); name != "name" {
		t.Errorf(`Name returns %s; want name`, name)
	}
	if size := v.Size(); size != int64(len(v.data)) {
		t.Errorf(`Size returns %d; want %d`, size, len(v.data))
	}
	v.isDir = true
	if size := v.Size(); size != 0 {
		t.Errorf(`dir Size returns %d; want 0`, size)
	}
	if mode := v.Mode(); mode != v.mode {
		t.Errorf(`Mode returns %v; want %v`, mode, v.mode)
	}
	if modTime := v.ModTime(); modTime != v.modTime {
		t.Errorf(`ModTime returns %v; want %v`, modTime, v.modTime)
	}
	if isDir := v.IsDir(); isDir != v.isDir {
		t.Errorf(`IsDir returns %v; want %v`, isDir, v.isDir)
	}
	if sys := v.Sys(); sys != nil {
		t.Errorf(`Sys returns %v; want nil`, sys)
	}
	if typ := v.Type(); typ != v.mode&fs.ModeType {
		t.Errorf(`Type returns %v; want %v`, typ, v.mode)
	}
	info, err := v.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info != v {
		t.Errorf(`Info returns %v; want %v`, info, v)
	}
}

var testStoreSrc = map[string]*value{
	"/":                {name: ".", mode: fs.ModePerm, isDir: true},
	"/dir0":            {name: "dir0", mode: fs.ModePerm, isDir: true},
	"/dir0/file01.txt": {name: "dir0/file01.txt", mode: fs.ModePerm, isDir: false},
	"/dir0/file02.txt": {name: "dir0/file02.txt", mode: fs.ModePerm, isDir: false},
	"/dir1":            {name: "dir0", mode: fs.ModePerm, isDir: true},
	"/dir1/file11.txt": {name: "dir1/file11.txt", mode: fs.ModePerm, isDir: false},
	"/dir1/file12.txt": {name: "dir1/file12.txt", mode: fs.ModePerm, isDir: false},
	"/file1.txt":       {name: "file1.txt", mode: fs.ModePerm, isDir: false},
	"/file2.txt":       {name: "file2.txt", mode: fs.ModePerm, isDir: false},
}

func newStoreTest() *store {
	s := newStore()
	for k, v := range testStoreSrc {
		s.put(k, v)
	}
	return s
}

func TestStore(t *testing.T) {
	s := newStoreTest()

	var wantKeys []string
	for k := range testStoreSrc {
		wantKeys = append(wantKeys, k)
	}
	sort.Strings(wantKeys)

	if !reflect.DeepEqual(s.keys, wantKeys) {
		t.Errorf(`Error store.keys is %v; want %v`, s.keys, wantKeys)
	}

	key := "/dir0/file02.txt"
	wantValue := testStoreSrc[key]
	gotValue := s.get(key)
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf(`Error store.get("%s") returns %v; want %v`, key, gotValue, wantValue)
	}
}

func TestStore_remove(t *testing.T) {
	s := newStoreTest()

	key := "/dir1/file11.txt"
	v := s.get(key)
	if v == nil {
		t.Errorf(`Error not found %s`, key)
	}

	s.remove(key)
	v = s.get(key)
	if v != nil {
		t.Errorf(`Error found %s: %v`, key, v)
	}

	for _, k := range s.keys {
		if k == key {
			t.Errorf(`Error found %s`, key)
		}
	}

	v = s.remove(key)
	if v != nil {
		t.Errorf(`Error found %s: %v`, key, v)
	}
}

func BenchmarkStore_put(b *testing.B) {
	// Pre-generate keys in a randomized order so that put hits the
	// realistic insertion-into-sorted-slice path rather than always
	// appending at the end.
	const n = 5000
	rng := rand.New(rand.NewSource(1))
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("/dir%05d/file%05d.txt", rng.Intn(n), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := newStore()
		for _, k := range keys {
			s.put(k, &value{name: k, mode: fs.ModePerm})
		}
	}
}

func TestStore_removeAll(t *testing.T) {
	s := newStoreTest()

	prefix := "/dir0"
	s.removeAll(prefix)

	for _, key := range s.keys {
		if strings.HasPrefix(key, prefix) {
			t.Errorf(`Error found %s`, key)
		}
		v := s.get(key)
		if v == nil {
			t.Errorf(`Error not found %s: %v`, key, v)
		}
	}

	want := len(s.keys)
	s.removeAll(prefix)
	got := len(s.keys)

	if got != want {
		t.Errorf(`Error keys length %d; want %d`, got, want)
	}
}

func TestStore_prefixKeys(t *testing.T) {
	testCases := []struct {
		want   []string
		prefix string
	}{
		{
			want: []string{
				"/dir0",
				"/dir1",
				"/file1.txt",
				"/file2.txt",
			},
			prefix: "/",
		}, {
			want: []string{
				"/dir0/file01.txt",
				"/dir0/file02.txt",
			},
			prefix: "/dir0",
		}, {
			prefix: "/not-found",
		},
	}

	s := newStoreTest()
	for _, tc := range testCases {
		got := s.prefixKeys(tc.prefix)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf(`Error prefixKeys("%s") got %v; want %v`, tc.prefix, got, tc.want)
		}
	}
}

func TestStore_prefixGlobKeys(t *testing.T) {
	testCases := []struct {
		want    []string
		prefix  string
		pattern string
		errStr  string
	}{
		{
			want: []string{
				"/dir0/file01.txt",
				"/dir1/file11.txt",
			},
			prefix:  "/",
			pattern: "*/*1.txt",
		}, {
			want: []string{
				"/dir0/file01.txt",
				"/dir0/file02.txt",
			},
			prefix:  "/dir0",
			pattern: "*.txt",
		}, {
			prefix:  "/not-found",
			pattern: "*.*",
		}, {
			prefix:  "/",
			pattern: "*.go",
		}, {
			prefix:  "/",
			pattern: "[[",
			errStr:  "syntax error in pattern",
		},
	}

	s := newStoreTest()
	for _, tc := range testCases {
		got, err := s.prefixGlobKeys(tc.prefix, tc.pattern)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.errStr {
			t.Errorf(`Error prefixGlobKeys("%s", "%s") error got "%s"; want "%s"`,
				tc.prefix, tc.pattern, errStr, tc.errStr)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf(`Error prefixGlobKeys("%s", "%s") got %v; want %v`,
				tc.prefix, tc.pattern, got, tc.want)
		}
	}
}
