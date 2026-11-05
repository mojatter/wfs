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

func TestOpenFSDelegator_TestFS(t *testing.T) {
	d := DelegateOpenFS(os.DirFS("osfs/testdata"))
	if err := fstest.TestFS(d, "dir0/file01.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestOpenFSDelegator_ErrNotImplemented(t *testing.T) {
	d := &OpenFSDelegator{}
	var err error
	_, err = d.Open("")
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("unexpected %v; want %v", err, ErrNotImplemented)
	}
}

func TestFSDelegator_TestFS(t *testing.T) {
	d := DelegateFS(os.DirFS("osfs/testdata"))
	if err := fstest.TestFS(d, "dir0/file01.txt"); err != nil {
		t.Fatal(err)
	}
}

func testFSDelegatorErrors(t *testing.T, d *FSDelegator, wantErr error) {
	var err error
	if _, err = d.Open(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.ReadDir(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.ReadFile(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.Glob(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.Stat(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.Lstat(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.ReadLink(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.Sub(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if err = d.MkdirAll("", fs.ModePerm); err != nil {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.CreateFile("", fs.ModePerm); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.WriteFile("", []byte{}, fs.ModePerm); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if err = d.RemoveFile(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if err = d.RemoveAll(""); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
}

func TestFSDelegator_ErrNotImplemented(t *testing.T) {
	testFSDelegatorErrors(t, &FSDelegator{}, ErrNotImplemented)
}

func TestFSDelegator(t *testing.T) {
	wantErr := errors.New("test")

	testFSDelegatorErrors(t, &FSDelegator{
		OpenFunc: func(_ string) (fs.File, error) {
			return nil, wantErr
		},
		ReadDirFunc: func(_ string) ([]fs.DirEntry, error) {
			return nil, wantErr
		},
		ReadFileFunc: func(_ string) ([]byte, error) {
			return nil, wantErr
		},
		GlobFunc: func(_ string) ([]string, error) {
			return nil, wantErr
		},
		StatFunc: func(_ string) (fs.FileInfo, error) {
			return nil, wantErr
		},
		LstatFunc: func(_ string) (fs.FileInfo, error) {
			return nil, wantErr
		},
		ReadLinkFunc: func(_ string) (string, error) {
			return "", wantErr
		},
		SubFunc: func(_ string) (fs.FS, error) {
			return nil, wantErr
		},
		MkdirAllFunc: func(_ string, _ fs.FileMode) error {
			return nil
		},
		CreateFileFunc: func(_ string, _ fs.FileMode) (WriterFile, error) {
			return nil, wantErr
		},
		WriteFileFunc: func(_ string, _ []byte, _ fs.FileMode) (int, error) {
			return 0, wantErr
		},
		RemoveFileFunc: func(_ string) error {
			return wantErr
		},
		RemoveAllFunc: func(_ string) error {
			return wantErr
		},
	}, wantErr)
}

func TestDelegateFS(t *testing.T) {
	DelegateFS(&FSDelegator{})
}

func TestDelegateFS_ReadDir(t *testing.T) {
	fsys := os.DirFS("osfs/testdata")
	path := "dir0"
	want, err := fs.ReadDir(fsys, path)
	if err != nil {
		t.Fatal(err)
	}
	d := DelegateFS(fsys)
	got, err := d.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestDelegateFS_ReadFile(t *testing.T) {
	fsys := os.DirFS("osfs/testdata")
	path := "dir0/file01.txt"
	want, err := fs.ReadFile(fsys, path)
	if err != nil {
		t.Fatal(err)
	}
	d := DelegateFS(fsys)
	got, err := d.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestDelegateFS_Glob(t *testing.T) {
	fsys := os.DirFS("osfs/testdata")
	pattern := "dir0/*.txt"
	want, err := fs.Glob(fsys, pattern)
	if err != nil {
		t.Fatal(err)
	}
	d := DelegateFS(fsys)
	got, err := d.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestDelegateFS_Stat(t *testing.T) {
	fsys := os.DirFS("osfs/testdata")
	path := "dir0/file01.txt"
	want, err := fs.Stat(fsys, path)
	if err != nil {
		t.Fatal(err)
	}
	d := DelegateFS(fsys)
	got, err := d.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestFSDelegator_Lstat(t *testing.T) {
	lstatInfo, statInfo := &FileInfoDelegator{}, &FileInfoDelegator{}
	lstatFunc := func(_ string) (fs.FileInfo, error) {
		return lstatInfo, nil
	}
	statFunc := func(_ string) (fs.FileInfo, error) {
		return statInfo, nil
	}
	testCases := []struct {
		caseName string
		d        *FSDelegator
		want     fs.FileInfo
		wantErr  bool
	}{
		{caseName: "LstatFunc", d: &FSDelegator{LstatFunc: lstatFunc}, want: lstatInfo},
		{caseName: "LstatFunc over StatFunc", d: &FSDelegator{LstatFunc: lstatFunc, StatFunc: statFunc}, want: lstatInfo},
		{caseName: "StatFunc fallback", d: &FSDelegator{StatFunc: statFunc}, want: statInfo},
		{caseName: "no funcs", d: &FSDelegator{}, wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := fs.Lstat(tc.d, "name")
			if tc.wantErr {
				var pathErr *fs.PathError
				if !errors.Is(err, ErrNotImplemented) || !errors.As(err, &pathErr) || pathErr.Op != "Lstat" {
					t.Errorf("unexpected %#v; want *fs.PathError{Op: Lstat} with %v", err, ErrNotImplemented)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("unexpected %v; want %v", got, tc.want)
			}
		})
	}
}

func TestDelegateFS_ReadLinkFS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "target.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	testCases := []struct {
		caseName   string
		fsys       fs.FS
		wantLink   bool
		wantTarget string
	}{
		{caseName: "ReadLinkFS", fsys: os.DirFS(dir), wantLink: true, wantTarget: "target.txt"},
		{caseName: "without ReadLinkFS", fsys: DelegateOpenFS(os.DirFS(dir))},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			d := DelegateFS(tc.fsys)
			if got := d.LstatFunc != nil && d.ReadLinkFunc != nil; got != tc.wantLink {
				t.Errorf("funcs set got %v; want %v", got, tc.wantLink)
			}

			info, err := fs.Lstat(d, "link")
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode()&fs.ModeSymlink != 0; got != tc.wantLink {
				t.Errorf("Lstat symlink got %v; want %v", got, tc.wantLink)
			}
			target, err := fs.ReadLink(d, "link")
			if tc.wantLink {
				if err != nil {
					t.Fatal(err)
				}
				if target != tc.wantTarget {
					t.Errorf("ReadLink got %q; want %q", target, tc.wantTarget)
				}
			} else if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("ReadLink error got %v; want %v", err, ErrNotImplemented)
			}
		})
	}
}

func TestDelegateFS_Sub(t *testing.T) {
	fsys := os.DirFS("osfs/testdata")
	path := "dir0"
	want, err := fs.Sub(fsys, path)
	if err != nil {
		t.Fatal(err)
	}
	d := DelegateFS(fsys)
	got, err := d.Sub(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %v; want %v", got, want)
	}
}

func TestDelegateFile(t *testing.T) {
	DelegateFile(&FileDelegator{})
}

func testFileDelegatorErrors(t *testing.T, d *FileDelegator, wantErr error) {
	var err error
	if _, err = d.Stat(); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.Read([]byte{}); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if err = d.Close(); err != nil {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.ReadDir(-1); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
	if _, err = d.Write([]byte{}); !errors.Is(err, wantErr) {
		t.Errorf("unexpected %v", err)
	}
}

func TestFileDelegator_ErrNotImplemented(t *testing.T) {
	testFileDelegatorErrors(t, &FileDelegator{}, ErrNotImplemented)
}

func TestFileDelegator(t *testing.T) {
	wantErr := errors.New("test")

	testFileDelegatorErrors(t, &FileDelegator{
		StatFunc: func() (fs.FileInfo, error) {
			return nil, wantErr
		},
		ReadFunc: func(p []byte) (int, error) {
			return 0, wantErr
		},
		CloseFunc: func() error {
			return nil
		},
		ReadDirFunc: func(n int) ([]fs.DirEntry, error) {
			return nil, wantErr
		},
		WriteFunc: func(p []byte) (int, error) {
			return 0, wantErr
		},
	}, wantErr)
}

func TestDirEntryDelegator(t *testing.T) {
	fsys := os.DirFS("osfs/testdata")
	ds, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) == 0 {
		t.Fatal("unexpected empty dir entries")
	}

	want := ds[0]
	got := DelegateDirEntry(want)
	if got.Name() != want.Name() {
		t.Errorf("unexpected %s; want %s", got.Name(), want.Name())
	}
	if got.IsDir() != want.IsDir() {
		t.Errorf("unexpected %v; want %v", got.IsDir(), want.IsDir())
	}
	if got.Type() != want.Type() {
		t.Errorf("unexpected %v; want %v", got.Type(), want.Type())
	}

	wantInfo, err := want.Info()
	if err != nil {
		t.Fatal(err)
	}
	gotInfo, err := got.Info()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotInfo, wantInfo) {
		t.Errorf("unexpected %v; want %v", gotInfo, wantInfo)
	}

	got.InfoFunc = nil
	gotInfo, err = got.Info()
	if err != nil {
		t.Fatal(err)
	}
	if gotInfo != nil {
		t.Errorf("unexpected %v; want nil", gotInfo)
	}
}

func TestInfoInfoDelegator(t *testing.T) {
	fsys := os.DirFS("osfs/testdata")
	want, err := fs.Stat(fsys, "dir0/file01.txt")
	if err != nil {
		t.Fatal(err)
	}

	got := DelegateFileInfo(want)
	if got.Name() != want.Name() {
		t.Errorf("unexpected %s; want %s", got.Name(), want.Name())
	}
	if got.Size() != want.Size() {
		t.Errorf("unexpected %d; want %d", got.Size(), want.Size())
	}
	if got.Mode() != want.Mode() {
		t.Errorf("unexpected %v; want %v", got.Mode(), want.Mode())
	}
	if got.ModTime() != want.ModTime() {
		t.Errorf("unexpected %v; want %v", got.ModTime(), want.ModTime())
	}
	if got.IsDir() != want.IsDir() {
		t.Errorf("unexpected %v; want %v", got.IsDir(), want.IsDir())
	}
	if got.Sys() != want.Sys() {
		t.Errorf("unexpected %v; want %v", got.Sys(), want.Sys())
	}
}
