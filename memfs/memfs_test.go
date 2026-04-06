package memfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/mojatter/wfs"
	"github.com/mojatter/wfs/wfstest"
)

func newMemFSTest(t *testing.T) *MemFS {
	fsys := New()
	err := wfs.CopyFS(fsys, os.DirFS("../osfs/testdata"), ".")
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

func TestFS(t *testing.T) {
	fsys := newMemFSTest(t)
	if err := fstest.TestFS(fsys, "dir0", "dir0/file01.txt"); err != nil {
		t.Errorf(`Error testing/fstest: %+v`, err)
	}
}

func TestWriteFileFS(t *testing.T) {
	fsys := New()
	tmpdir := "tmpdir"
	if err := fsys.mkdirAll(tmpdir, fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := wfstest.TestWriteFileFS(fsys, tmpdir); err != nil {
		t.Errorf(`Error wfs/wfstest: %+v`, err)
	}
}

func TestRenameFS(t *testing.T) {
	fsys := New()
	tmpdir := "tmpdir"
	if err := fsys.MkdirAll(tmpdir, fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := wfstest.TestRenameFS(fsys, tmpdir); err != nil {
		t.Errorf(`Error wfs/wfstest: %+v`, err)
	}
}

func TestSubSharesMutex(t *testing.T) {
	fsys := New()
	if err := fsys.MkdirAll("sub", fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	subFS, err := fsys.Sub("sub")
	if err != nil {
		t.Fatal(err)
	}
	sub, ok := subFS.(*MemFS)
	if !ok {
		t.Fatalf("Sub returned %T; want *MemFS", subFS)
	}
	if sub.mutex != fsys.mutex {
		t.Errorf("Sub MemFS does not share mutex with parent")
	}

	// Smoke test concurrent access through parent and sub. With a shared
	// mutex this is race-free; -race will catch a regression.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = wfs.WriteFile(fsys, "sub/a.txt", []byte("a"), fs.ModePerm)
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		_, _ = wfs.WriteFile(sub, "b.txt", []byte("b"), fs.ModePerm)
	}
	<-done
}

func TestMemFile_Sync(t *testing.T) {
	fsys := New()
	f, err := fsys.CreateFile("sync.txt", fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sf, ok := f.(wfs.SyncWriterFile)
	if !ok {
		t.Fatalf("memfs CreateFile result does not implement SyncWriterFile")
	}
	if err := sf.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
}

func TestCreateFile(t *testing.T) {
	testCases := []struct {
		name   string
		errStr string
	}{
		{
			name: "file.txt",
		}, {
			name: "newDir/file.txt",
		}, {
			name:   "newDir",
			errStr: "Create newDir: invalid argument",
		}, {
			name:   "newDir/file.txt/invalid",
			errStr: "MkdirAll newDir/file.txt: invalid argument",
		}, {
			name:   "../invalid",
			errStr: "Create ../invalid: invalid argument",
		}, {
			name: "dir0/file01.txt",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		_, err := fsys.CreateFile(tc.name, fs.ModePerm)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.errStr {
			t.Errorf(`Error Create("%s") error got "%s"; want "%s"`, tc.name, errStr, tc.errStr)
		}
		if err != nil {
			continue
		}
		info, err := fsys.Stat(tc.name)
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			t.Errorf(`Error %s IsDir() returns true; want false`, tc.name)
		}
	}
}

func TestMkdirAll(t *testing.T) {
	testCases := []struct {
		dir    string
		errStr string
	}{
		{
			dir: "test0",
		}, {
			dir: "test0/test1",
		}, {
			dir: "test2/test3",
		}, {
			dir:    "../invalid",
			errStr: "MkdirAll ../invalid: invalid argument",
		}, {
			dir:    "dir0/file01.txt",
			errStr: "MkdirAll dir0/file01.txt: invalid argument",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		err := fsys.MkdirAll(tc.dir, fs.ModePerm)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.errStr {
			t.Errorf(`Error MkdirAll("%s") error got "%s"; want "%s"`, tc.dir, errStr, tc.errStr)
		}
		if err != nil {
			continue
		}
		info, err := fsys.Stat(tc.dir)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			t.Errorf(`Error %s IsDir() returns false; want true`, tc.dir)
		}
	}
}

func TestGlob(t *testing.T) {
	testCases := []struct {
		want    []string
		pattern string
		errStr  string
	}{
		{
			want: []string{
				"dir0/file01.txt",
			},
			pattern: "*/*1.txt",
		}, {
			want: []string{
				"dir0/file01.txt",
				"dir0/file02.txt",
			},
			pattern: "dir0/*.txt",
		}, {
			pattern: "no-match",
		}, {
			pattern: "[[",
			errStr:  "syntax error in pattern",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		got, err := fsys.Glob(tc.pattern)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.errStr {
			t.Errorf(`Error Glob("%s") error got "%s"; want "%s"`, tc.pattern, errStr, tc.errStr)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf(`Error Glob("%s") got %v; want %v`, tc.pattern, got, tc.want)
		}
	}
}

func TestReadDir(t *testing.T) {
	testCases := []struct {
		want   []string
		dir    string
		errStr string
	}{
		{
			want: []string{
				"dir0",
			},
			dir: ".",
		}, {
			want: []string{
				"file01.txt",
				"file02.txt",
			},
			dir: "dir0",
		}, {
			dir:    "not-found",
			errStr: "Open not-found: file does not exist",
		}, {
			dir:    "dir0/file01.txt",
			errStr: "ReadDir dir0/file01.txt: not a directory",
		}, {
			dir:    "../invalid",
			errStr: "Open ../invalid: invalid argument",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		entries, err := fsys.ReadDir(tc.dir)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.errStr {
			t.Errorf(`Error ReadDir("%s") error got "%s"; want "%s"`, tc.dir, errStr, tc.errStr)
		}
		var got []string
		for _, entry := range entries {
			got = append(got, entry.Name())
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf(`Error ReadDir("%s") got %v; want %v`, tc.dir, got, tc.want)
		}
	}
}

func TestReadFile(t *testing.T) {
	testCases := []struct {
		want   []byte
		name   string
		errStr string
	}{
		{
			want: []byte("content01\n"),
			name: "dir0/file01.txt",
		}, {
			name:   "not-found",
			errStr: "Open not-found: file does not exist",
		}, {
			name:   "dir0",
			errStr: "ReadFile dir0: invalid argument",
		}, {
			name:   "../invalid.txt",
			errStr: "Open ../invalid.txt: invalid argument",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		got, err := fsys.ReadFile(tc.name)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.errStr {
			t.Errorf(`Error ReadFile("%s") error got "%s"; want "%s"`, tc.name, errStr, tc.errStr)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf(`Error ReadFile("%s") got "%s"; want "%s"`, tc.name, got, tc.want)
		}
	}
}

func TestSub(t *testing.T) {
	fsys := newMemFSTest(t)
	dir0, err := fsys.Sub("dir0")
	if err != nil {
		t.Fatal(err)
	}
	memfsDir0 := dir0.(*MemFS)

	// NOTE: Write to sub filesystem.
	name := "test.txt"
	want := []byte(`test`)
	_, err = memfsDir0.WriteFile(name, want, fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	// NOTE: Read from parent filesystem.
	got, err := fsys.ReadFile("dir0/" + name)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf(`Error ReadFile("%s") got "%s"; want "%s"`, name, got, want)
	}
}

func TestSub_Errors(t *testing.T) {
	testCases := []struct {
		dir    string
		errStr string
	}{
		{
			dir:    "../invalid",
			errStr: "Sub ../invalid: invalid argument",
		}, {
			dir:    "not-found",
			errStr: "Open not-found: file does not exist",
		}, {
			dir:    "dir0/file01.txt",
			errStr: "Sub dir0/file01.txt: invalid argument",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		var err error
		_, err = fsys.Sub(tc.dir)
		if err == nil {
			t.Fatalf(`Fatal Sub("%s") return no error`, tc.dir)
		}
		if err.Error() != tc.errStr {
			t.Errorf(`Error Sub("%s") error got "%v"; want "%s"`, tc.dir, err, tc.errStr)
		}
	}
}

func TestWriteFile(t *testing.T) {
	data := []byte(`testdata`)
	testCases := []struct {
		name   string
		errStr string
	}{
		{
			name: "new.txt",
		}, {
			name: "dir0/file01.txt",
		}, {
			name:   "dir0",
			errStr: "Create dir0: invalid argument",
		}, {
			name:   "../invalid.txt",
			errStr: "Create ../invalid.txt: invalid argument",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		n, err := fsys.WriteFile(tc.name, data, fs.ModePerm)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.errStr {
			t.Errorf(`Error WriteFile("%s") error got "%s"; want "%s"`, tc.name, errStr, tc.errStr)
		}
		if errStr == "" && n != len(data) {
			t.Errorf(`Error WriteFile("%s") returns %d; want %d`, tc.name, n, len(data))
		}
	}
}

func TestRemoveFile(t *testing.T) {
	fsys := newMemFSTest(t)
	name := "dir0/file01.txt"

	// NOTE: Check exists.
	var err error
	_, err = fsys.Stat(name)
	if err != nil {
		t.Fatal(err)
	}

	err = fsys.RemoveFile(name)
	if err != nil {
		t.Fatal(err)
	}

	info, err := fsys.Stat(name)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf(`Error RemoveFile("%s") after Stat returns %v`, name, info)
	}
}

func TestRemoveFile_Errors(t *testing.T) {
	fsys := newMemFSTest(t)
	name := "../invalid"

	want := &fs.PathError{Op: "RemoveFile", Path: name, Err: fs.ErrInvalid}
	got := fsys.RemoveFile(name)

	if !reflect.DeepEqual(got, want) {
		t.Errorf(`Error RemoveFile("%s") returns %v; want %v`, name, got, want)
	}
}

func TestRemoveAll(t *testing.T) {
	fsys := newMemFSTest(t)
	dir := "dir0"

	var want []string
	for _, k := range fsys.store.keys {
		if !strings.HasPrefix(k, "/"+dir) {
			want = append(want, k)
		}
	}

	err := fsys.RemoveAll("dir0")
	if err != nil {
		t.Fatal(err)
	}

	got := fsys.store.keys
	if !reflect.DeepEqual(got, want) {
		t.Errorf(`Error RemoveAll("%s") after keys %v; want %v`, dir, got, want)
	}
}

func TestRemoveAll_Errors(t *testing.T) {
	fsys := newMemFSTest(t)
	name := "../invalid"

	want := &fs.PathError{Op: "RemoveAll", Path: name, Err: fs.ErrInvalid}
	got := fsys.RemoveAll(name)

	if !reflect.DeepEqual(got, want) {
		t.Errorf(`Error RemoveAll("%s") returns %v; want %v`, name, got, want)
	}
}

func TestMemFile_Read_Errors(t *testing.T) {
	fsys := newMemFSTest(t)
	name := "dir0"

	f, err := fsys.Open(name)
	if err != nil {
		t.Fatal(err)
	}

	memf, ok := f.(*MemFile)
	if !ok {
		t.Fatalf(`Fatal not MemFile: %#v`, f)
	}

	_, err = memf.Read([]byte{})
	if err == nil {
		t.Fatalf(`Fatal Read(1) returns no error`)
	}
}

func TestMemFile_ReadDir(t *testing.T) {
	fsys := newMemFSTest(t)
	dir := "dir0"

	f, err := fsys.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	memf, ok := f.(*MemFile)
	if !ok {
		t.Fatalf(`Fatal not MemFile: %#v`, f)
	}

	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "file01.txt",
		}, {
			name: "file02.txt",
		}, {
			err: io.EOF,
		},
	}

	for _, tc := range testCases {
		entries, err := memf.ReadDir(1)
		if tc.err != nil {
			if !errors.Is(err, tc.err) {
				t.Errorf(`Error ReadDir(1) error %v; want %v`, err, tc.err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Errorf(`Error ReadDir(1) returns %d entries; want 1`, len(entries))
		}
		if entries[0].Name() != tc.name {
			t.Errorf(`Error ReadDir(1) returns unknown entries %v`, entries)
		}
	}
}

func TestMemFile_ReadDir_Errors(t *testing.T) {
	fsys := newMemFSTest(t)
	dir := "dir0"

	f, err := fsys.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	memf, ok := f.(*MemFile)
	if !ok {
		t.Fatalf(`Fatal not MemFile: %#v`, f)
	}

	memf.name = "../invalid"
	_, err = memf.ReadDir(1)
	if err == nil {
		t.Fatalf(`Fatal ReadDir(1) returns no error`)
	}
}
