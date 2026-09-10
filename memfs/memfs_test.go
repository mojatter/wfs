package memfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"slices"
	"strings"
	"syscall"
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
		for range 100 {
			_, _ = wfs.WriteFile(fsys, "sub/a.txt", []byte("a"), fs.ModePerm)
		}
		close(done)
	}()
	for range 100 {
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
			errStr: "Create newDir: is a directory",
		}, {
			name:   "newDir/file.txt/invalid",
			errStr: "MkdirAll newDir/file.txt: not a directory",
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
			errStr: "MkdirAll dir0/file01.txt: not a directory",
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

func TestGlob_Sub(t *testing.T) {
	fsys := newMemFSTest(t)
	sub, err := fsys.Sub("dir0")
	if err != nil {
		t.Fatal(err)
	}

	// NOTE: Names are relative to the sub, so they can be opened through it.
	got, err := sub.(*MemFS).Glob("*.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"file01.txt", "file02.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf(`Error Glob("*.txt") through a sub got %v; want %v`, got, want)
	}
	for _, name := range got {
		f, err := sub.Open(name)
		if err != nil {
			t.Errorf(`Error Open("%s") got "%v"; want no error`, name, err)
			continue
		}
		f.Close()
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
			errStr: "ReadFile dir0: is a directory",
		}, {
			name:   "../invalid.txt",
			errStr: "Open ../invalid.txt: invalid argument",
		}, {
			name:   "dir0/file01.txt/below",
			errStr: "Open dir0/file01.txt/below: not a directory",
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
			dir:    "/absolute",
			errStr: "Sub /absolute: invalid argument",
		}, {
			dir:    "",
			errStr: "Sub : invalid argument",
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

func TestSub_Lazy(t *testing.T) {
	fsys := newMemFSTest(t)

	// NOTE: Sub into a missing directory; the write creates it.
	sub, err := fsys.Sub("not-found")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`test`)
	_, err = sub.(*MemFS).WriteFile("test.txt", want, fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fsys.ReadFile("not-found/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf(`Error ReadFile("not-found/test.txt") got "%s"; want "%s"`, got, want)
	}

	// NOTE: The created directory is named after its own segment, and is
	// walkable from the parent.
	info, err := fsys.Stat("not-found")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "not-found" {
		t.Errorf(`Error Stat("not-found") name got "%s"; want "not-found"`, info.Name())
	}
	var walked []string
	err = fs.WalkDir(fsys, ".", func(name string, _ fs.DirEntry, err error) error {
		walked = append(walked, name)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(walked, "not-found/test.txt") {
		t.Errorf(`Error WalkDir did not visit "not-found/test.txt"; got %v`, walked)
	}

	// NOTE: A nested write through a sub creates the intermediate directory
	// under the sub, not under a duplicated root.
	if _, err := sub.(*MemFS).WriteFile("nested/test.txt", want, fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.(*MemFS).ReadDir("nested"); err != nil {
		t.Errorf(`Error ReadDir("nested") through a sub got "%v"; want no error`, err)
	}

	// NOTE: Sub into a file succeeds and fails on use.
	sub, err = fsys.Sub("dir0/file01.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sub.Open("test.txt"); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf(`Error Open("test.txt") through a file Sub got "%v"; want ENOTDIR`, err)
	}
	if _, err := sub.(*MemFS).WriteFile("test.txt", want, fs.ModePerm); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf(`Error WriteFile("test.txt") through a file Sub got "%v"; want ENOTDIR`, err)
	}
	// NOTE: The file itself is not reachable as the root of the sub.
	if _, err := sub.Open("."); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf(`Error Open(".") through a file Sub got "%v"; want ENOTDIR`, err)
	}
	if _, err := sub.(*MemFS).ReadFile("."); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf(`Error ReadFile(".") through a file Sub got "%v"; want ENOTDIR`, err)
	}
	if _, err := sub.(*MemFS).ReadDir("."); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf(`Error ReadDir(".") through a file Sub got "%v"; want ENOTDIR`, err)
	}
	if _, err := sub.(*MemFS).Stat("."); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf(`Error Stat(".") through a file Sub got "%v"; want ENOTDIR`, err)
	}
}

func TestRename_Errors(t *testing.T) {
	testCases := []struct {
		caseName string
		oldpath  string
		newpath  string
		errStr   string
	}{
		{
			caseName: "below an existing file",
			oldpath:  "dir0/file01.txt/below",
			newpath:  "moved.txt",
			errStr:   "Rename dir0/file01.txt/below: not a directory",
		}, {
			caseName: "missing source",
			oldpath:  "not-found",
			newpath:  "moved.txt",
			errStr:   "Rename not-found: file does not exist",
		}, {
			caseName: "directory source",
			oldpath:  "dir0",
			newpath:  "moved",
			errStr:   "Rename dir0: invalid argument",
		}, {
			caseName: "destination below an existing file",
			oldpath:  "dir0/file01.txt",
			newpath:  "dir0/file02.txt/below",
			errStr:   "MkdirAll dir0/file02.txt: not a directory",
		},
	}

	fsys := newMemFSTest(t)
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := fsys.Rename(tc.oldpath, tc.newpath)
			if err == nil {
				t.Fatalf(`Fatal Rename("%s", "%s") returned no error`, tc.oldpath, tc.newpath)
			}
			if err.Error() != tc.errStr {
				t.Errorf(`Error Rename("%s", "%s") error got "%v"; want "%s"`,
					tc.oldpath, tc.newpath, err, tc.errStr)
			}
		})
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
			errStr: "Create dir0: is a directory",
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

	wantErr := &fs.PathError{Op: "RemoveFile", Path: name, Err: fs.ErrInvalid}
	err := fsys.RemoveFile(name)
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok {
		t.Fatalf("unexpected %v", err)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf(`Error RemoveFile("%s") returns %v; want %v`, name, gotErr, wantErr)
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

	wantErr := &fs.PathError{Op: "RemoveAll", Path: name, Err: fs.ErrInvalid}
	err := fsys.RemoveAll(name)
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok {
		t.Fatalf("unexpected %v", err)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf(`Error RemoveAll("%s") returns %v; want %v`, name, gotErr, wantErr)
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
