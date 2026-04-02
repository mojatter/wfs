package wfs

import (
	"errors"
	"io/fs"
	"os"
	"reflect"
	"testing"
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
	dest.CreateFileFunc = func(name string, mode fs.FileMode) (WriterFile, error) {
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
