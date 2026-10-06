package wfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestMkdirAll(t *testing.T) {
	got := ""
	fsys := &FSDelegator{
		MkdirAllFunc: func(dir string, _ fs.FileMode) error {
			got = dir
			return nil
		},
	}

	want := "path/to/dir"
	err := MkdirAll(fsys, want, fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("unexpected %s; want %s", got, want)
	}
}

func TestMkdirAll_ErrNotImplemented(t *testing.T) {
	fsys := &OpenFSDelegator{}

	dir := "path/to/dir"
	wantErr := &fs.PathError{Op: "MkdirAll", Path: dir, Err: ErrNotImplemented}

	err := MkdirAll(fsys, dir, fs.ModePerm)
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok {
		t.Errorf("unexpected %v", err)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", gotErr, wantErr)
	}
}

func TestCreateFile(t *testing.T) {
	want := &FileDelegator{}
	called := false
	fsys := &FSDelegator{
		CreateFileFunc: func(_ string, _ fs.FileMode) (WriterFile, error) {
			called = true
			return want, nil
		},
	}

	got, err := CreateFile(fsys, "test.txt", fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("not called CreateFile")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestCreateFile_ErrNotImplemented(t *testing.T) {
	fsys := &OpenFSDelegator{}

	name := "test.txt"
	wantErr := &fs.PathError{Op: "CreateFile", Path: name, Err: ErrNotImplemented}

	var err error
	_, err = CreateFile(fsys, name, fs.ModePerm)
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok {
		t.Errorf("unexpected %v", err)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", gotErr, wantErr)
	}
}

func TestWriteFile(t *testing.T) {
	want := 1
	called := false
	fsys := &FSDelegator{
		WriteFileFunc: func(_ string, _ []byte, _ fs.FileMode) (int, error) {
			called = true
			return want, nil
		},
	}

	got, err := WriteFile(fsys, "", []byte{}, fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("not called WriteFile")
	}
	if got != want {
		t.Errorf("unexpected %d; want %d", got, want)
	}
}

func TestWriteFile_ErrNotImplemented(t *testing.T) {
	fsys := &OpenFSDelegator{}

	name := "test.txt"
	wantErr := &fs.PathError{Op: "WriteFile", Path: name, Err: ErrNotImplemented}

	var err error
	_, err = WriteFile(fsys, name, []byte{}, fs.ModePerm)
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok {
		t.Errorf("unexpected %v", err)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", gotErr, wantErr)
	}
}

func TestRemoveFile(t *testing.T) {
	called := false
	fsys := &FSDelegator{
		RemoveFileFunc: func(name string) error {
			called = true
			return nil
		},
	}

	err := RemoveFile(fsys, "")
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("not called RemoveFile")
	}
}

func TestRemoveAll(t *testing.T) {
	called := false
	fsys := &FSDelegator{
		RemoveAllFunc: func(name string) error {
			called = true
			return nil
		},
	}

	err := RemoveAll(fsys, "")
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("not called RemoveAll")
	}
}

func TestRemoveFile_ErrNotImplemented(t *testing.T) {
	fsys := &OpenFSDelegator{}

	name := "test.txt"
	wantErr := &fs.PathError{Op: "RemoveFile", Path: name, Err: ErrNotImplemented}

	err := RemoveFile(fsys, name)
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok {
		t.Errorf("unexpected %v", err)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", gotErr, wantErr)
	}
}

func TestRemoveAll_ErrNotImplemented(t *testing.T) {
	fsys := &OpenFSDelegator{}

	path := "path/to/dir"
	wantErr := &fs.PathError{Op: "RemoveAll", Path: path, Err: ErrNotImplemented}

	err := RemoveAll(fsys, path)
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok {
		t.Errorf("unexpected %v", err)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", gotErr, wantErr)
	}
}

func TestRename(t *testing.T) {
	var gotOld, gotNew string
	fsys := &FSDelegator{
		RenameFunc: func(oldpath, newpath string) error {
			gotOld, gotNew = oldpath, newpath
			return nil
		},
	}
	if err := Rename(fsys, "a", "b"); err != nil {
		t.Fatal(err)
	}
	if gotOld != "a" || gotNew != "b" {
		t.Errorf("unexpected %s -> %s", gotOld, gotNew)
	}
}

func TestRename_ErrNotImplemented(t *testing.T) {
	fsys := &OpenFSDelegator{}
	wantErr := &fs.PathError{Op: "Rename", Path: "a", Err: ErrNotImplemented}
	err := Rename(fsys, "a", "b")
	if err == nil {
		t.Fatal("no error")
	}
	gotErr, ok := err.(*fs.PathError)
	if !ok || gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", err, wantErr)
	}
}

func TestCopyFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	root := "dir0"
	want := map[string][]byte{}
	src := os.DirFS("osfs/testdata")
	err = fs.WalkDir(src, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		p, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		want[path] = p
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	got := map[string][]byte{}
	dest := DelegateFS(os.DirFS(tmpDir))
	dest.MkdirAllFunc = func(dir string, mode fs.FileMode) error {
		if mode != fs.ModePerm {
			t.Errorf("%s: mode = %v; want %v", dir, mode, fs.ModePerm)
		}
		return nil
	}
	dest.CreateFileFunc = func(name string, mode fs.FileMode) (WriterFile, error) {
		if mode != 0o666 {
			t.Errorf("%s: mode = %v; want %v", name, mode, fs.FileMode(0o666))
		}
		return &FileDelegator{
			WriteFunc: func(p []byte) (int, error) {
				got[name] = p
				return len(p), nil
			},
		}, nil
	}

	err = CopyFS(dest, src, root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestCopyFS_StatError(t *testing.T) {
	wantErr := errors.New("test")

	src := DelegateFS(os.DirFS("osfs/testdata"))
	src.StatFunc = func(name string) (fs.FileInfo, error) {
		return nil, wantErr
	}

	gotErr := CopyFS(&FSDelegator{}, src, ".")
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", gotErr, wantErr)
	}
}

func TestCopyFS_OpenError(t *testing.T) {
	wantErr := errors.New("test")

	src := DelegateFS(os.DirFS("osfs/testdata"))
	src.OpenFunc = func(name string) (fs.File, error) {
		return nil, wantErr
	}

	gotErr := CopyFS(&FSDelegator{}, src, ".")
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %+v; want %v", gotErr, wantErr)
	}
}

func TestCopyFS_CreateFileError(t *testing.T) {
	wantErr := errors.New("test")

	src := os.DirFS("osfs/testdata")
	dest := &FSDelegator{
		CreateFileFunc: func(_ string, _ fs.FileMode) (WriterFile, error) {
			return nil, wantErr
		},
	}

	gotErr := CopyFS(dest, src, ".")
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %+v; want %v", gotErr, wantErr)
	}
}

func TestGlob(t *testing.T) {
	fsys := os.DirFS(".")
	_, err := Glob(fsys, "*.md")
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadFile(t *testing.T) {
	fsys := os.DirFS(".")
	_, err := ReadFile(fsys, "README.md")
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidPath(t *testing.T) {
	want := true
	got := ValidPath(".")
	if got != want {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestWalkDir(t *testing.T) {
	fsys := os.DirFS(".")
	err := WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCopyFS_CloseError(t *testing.T) {
	writeErr := errors.New("write")
	closeErr := errors.New("close")
	testCases := []struct {
		caseName string
		writeErr error
		closeErr error
		wantErrs []error
	}{
		{caseName: "Close fails", closeErr: closeErr, wantErrs: []error{closeErr}},
		{caseName: "Write fails", writeErr: writeErr, wantErrs: []error{writeErr}},
		{caseName: "Write and Close fail", writeErr: writeErr, closeErr: closeErr, wantErrs: []error{writeErr, closeErr}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			closed := false
			dest := DelegateFS(fstest.MapFS{})
			dest.CreateFileFunc = func(string, fs.FileMode) (WriterFile, error) {
				return &FileDelegator{
					WriteFunc: func(p []byte) (int, error) { return len(p), tc.writeErr },
					CloseFunc: func() error { closed = true; return tc.closeErr },
				}, nil
			}

			err := CopyFS(dest, fstest.MapFS{"a.txt": {Data: []byte("x")}}, ".")
			for _, want := range tc.wantErrs {
				if !errors.Is(err, want) {
					t.Errorf("CopyFS = %v; want %v", err, want)
				}
			}
			// A Write error alone must come back unwrapped.
			if tc.closeErr == nil && err != tc.writeErr {
				t.Errorf("CopyFS = %#v; want %v unchanged", err, tc.writeErr)
			}
			if !closed {
				t.Error("Close not called")
			}
		})
	}
}

func TestCopyFS_NonRegular(t *testing.T) {
	testCases := []struct {
		caseName string
		src      fstest.MapFS
	}{
		{caseName: "named pipe", src: fstest.MapFS{"a": {Mode: fs.ModeNamedPipe}}},
		{caseName: "device", src: fstest.MapFS{"a": {Mode: fs.ModeDevice}}},
		{caseName: "char device", src: fstest.MapFS{"a": {Mode: fs.ModeDevice | fs.ModeCharDevice}}},
		{caseName: "socket", src: fstest.MapFS{"a": {Mode: fs.ModeSocket}}},
		{caseName: "irregular", src: fstest.MapFS{"a": {Mode: fs.ModeIrregular}}},
		{caseName: "symlink to dir", src: fstest.MapFS{"a": {Data: []byte("d"), Mode: fs.ModeSymlink}, "d/x": {Data: []byte("x")}}},
		{caseName: "symlink to named pipe", src: fstest.MapFS{"a": {Data: []byte("p"), Mode: fs.ModeSymlink}, "p": {Mode: fs.ModeNamedPipe}}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			created := false
			dest := DelegateFS(fstest.MapFS{})
			dest.CreateFileFunc = func(string, fs.FileMode) (WriterFile, error) {
				created = true
				return &FileDelegator{}, nil
			}

			err := CopyFS(dest, tc.src, ".")
			var pathErr *fs.PathError
			if !errors.As(err, &pathErr) || pathErr.Op != "CopyFS" || pathErr.Path != "a" || !errors.Is(err, fs.ErrInvalid) {
				t.Errorf("CopyFS = %#v; want PathError{Op: CopyFS, Path: a, Err: ErrInvalid}", err)
			}
			if created {
				t.Error("CreateFile called")
			}
		})
	}
}

func TestCopyFS_Symlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "target.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	got := map[string][]byte{}
	dest := DelegateFS(fstest.MapFS{})
	dest.CreateFileFunc = func(name string, mode fs.FileMode) (WriterFile, error) {
		return &FileDelegator{WriteFunc: func(p []byte) (int, error) {
			got[name] = append(got[name], p...)
			return len(p), nil
		}}, nil
	}
	if err := CopyFS(dest, os.DirFS(dir), "."); err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{"link": []byte("x"), "target.txt": []byte("x")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("copied %v; want %v", got, want)
	}

	if err := os.Symlink("missing", filepath.Join(dir, "broken")); err != nil {
		t.Fatal(err)
	}
	if err := CopyFS(dest, os.DirFS(dir), "."); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("CopyFS with broken symlink = %v; want ErrNotExist", err)
	}
}
