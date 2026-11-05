package osfs

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/mojatter/wfs"
	"github.com/mojatter/wfs/wfstest"
)

func TestFS(t *testing.T) {
	fsys := New("testdata")
	if err := fstest.TestFS(fsys, "dir0", "dir0/file01.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestWriteFileFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := New(filepath.Dir(tmpDir))
	if err := wfstest.TestWriteFileFS(fsys, filepath.Base(tmpDir)); err != nil {
		t.Fatal(err)
	}
}

func TestRenameFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := New(filepath.Dir(tmpDir))
	if err := wfstest.TestRenameFS(fsys, filepath.Base(tmpDir)); err != nil {
		t.Fatal(err)
	}
}

func TestFileHandle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := New(filepath.Dir(tmpDir))
	if err := wfstest.TestFileHandle(fsys, filepath.Base(tmpDir)); err != nil {
		t.Fatal(err)
	}
}

func TestCopyFS(t *testing.T) {
	fsys := New(t.TempDir())
	src := os.DirFS("testdata")
	if err := wfs.CopyFS(fsys, src, "."); err != nil {
		t.Fatal(err)
	}

	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			info, err := fs.Stat(fsys, path)
			if err != nil {
				return err
			}
			if info.Mode().Perm()&0o700 != 0o700 {
				t.Errorf("%s mode = %v; want a directory with owner rwx", path, info.Mode())
			}
			return nil
		}
		want, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		got, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %q; want %q", path, got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSyncWriterFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := New(tmpDir)
	f, err := fsys.CreateFile("sync.txt", fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write([]byte("data")); err != nil {
		t.Fatal(err)
	}
	sf, ok := f.(wfs.SyncWriterFile)
	if !ok {
		t.Fatalf("osfs CreateFile result does not implement SyncWriterFile")
	}
	if err := sf.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
}

func TestMkdirAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := NewOSFS(tmpDir)
	err = fsys.MkdirAll("dir", fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	err = fsys.MkdirAll("../invalid", fs.ModePerm)
	if err == nil {
		t.Fatal(err)
	}
}

func TestCreateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := DirFS(tmpDir)
	got, err := wfs.CreateFile(fsys, "test.txt", fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
}

func TestCreateFile_Mode(t *testing.T) {
	testCases := []struct {
		caseName string
		existing fs.FileMode // mode of a file written first, if any
		mode     fs.FileMode
		want     fs.FileMode
	}{
		{caseName: "new file in missing parents", mode: 0o600, want: 0o600},
		{caseName: "existing file keeps its mode", existing: 0o600, mode: 0o644, want: 0o600},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			fsys := New(t.TempDir())
			name := "a/b/c.txt"
			if tc.existing != 0 {
				if _, err := fsys.WriteFile(name, []byte("old"), tc.existing); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := fsys.WriteFile(name, []byte("new"), tc.mode); err != nil {
				t.Fatal(err)
			}

			info, err := fs.Stat(fsys, name)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != tc.want {
				t.Errorf("file mode = %v; want %v", got, tc.want)
			}
			for _, dir := range []string{"a", "a/b"} {
				info, err := fs.Stat(fsys, dir)
				if err != nil {
					t.Fatal(err)
				}
				if !info.IsDir() || info.Mode().Perm()&0o700 != 0o700 {
					t.Errorf("%s mode = %v; want a directory with owner rwx", dir, info.Mode())
				}
			}
		})
	}
}

func TestCreateFile_MkdirAllError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	orgMkdirAllFunc := osMkdirAllFunc
	defer func() { osMkdirAllFunc = orgMkdirAllFunc }()

	wantErr := errors.New("test")
	osMkdirAllFunc = func(dir string, perm os.FileMode) error {
		return wantErr
	}

	fsys := DirFS(tmpDir)
	var gotErr error
	_, gotErr = wfs.CreateFile(fsys, "name.txt", fs.ModePerm)

	if gotErr.Error() != wantErr.Error() {
		t.Errorf("unexpected %v; want %v", gotErr, wantErr)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	name := "test.txt"
	want := []byte("test")

	fsys := DirFS(tmpDir)
	n, err := wfs.WriteFile(fsys, name, want, fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(want) {
		t.Errorf("unexpected %d; want %d", n, len(want))
	}

	got, err := os.ReadFile(tmpDir + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %s; want %s", got, want)
	}
}

func TestWriteFile_InvalidError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := DirFS(tmpDir)
	_, err = wfs.WriteFile(fsys, "../invalid.txt", []byte{}, fs.ModePerm)
	if err == nil {
		t.Fatal("no error")
	}
}

func TestContainsDenyWin(t *testing.T) {
	testCases := []struct {
		name string
		want bool
	}{
		{
			name: "allow.txt",
			want: false,
		}, {
			name: "path/to/allow.txt",
			want: false,
		}, {
			name: "deny:txt",
			want: true,
		}, {
			name: "C:/deny.txt",
			want: true,
		}, {
			name: `path\to\deny.txt`,
			want: true,
		},
	}
	for i, testCase := range testCases {
		got := containsDenyWin(testCase.name)
		if got != testCase.want {
			t.Errorf("tests[%d] %s unexpected %v; want %v",
				i, testCase.name, got, testCase.want)
		}
	}
}

func TestSub_WriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dir := "sub"
	name := "test.txt"
	want := []byte("test")

	fsys, err := fs.Sub(DirFS(tmpDir), dir)
	if err != nil {
		t.Fatal(err)
	}
	n, err := wfs.WriteFile(fsys, name, want, fs.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(want) {
		t.Errorf("unexpected %d; want %d", n, len(want))
	}

	got, err := os.ReadFile(tmpDir + "/" + dir + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected %s; want %s", got, want)
	}
}

func TestSub_Errors(t *testing.T) {
	testCases := []struct {
		caseName string
		dir      string
	}{
		{caseName: "escapes the root", dir: "../outside"},
		{caseName: "absolute path", dir: "/absolute"},
		{caseName: "empty path", dir: ""},
	}

	fsys := New("testdata")
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			_, err := fsys.Sub(tc.dir)
			if !errors.Is(err, fs.ErrInvalid) {
				t.Errorf(`Error Sub("%s") error got %v; want %v`, tc.dir, err, fs.ErrInvalid)
			}
		})
	}
}

func TestRemoveFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := DirFS(tmpDir)
	name := "test.txt"

	if err = os.WriteFile(tmpDir+"/"+name, []byte{}, fs.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = wfs.RemoveFile(fsys, name)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveFile_InvalidError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := DirFS(tmpDir)
	err = wfs.RemoveFile(fsys, "../invalid-dir")
	if err == nil {
		t.Fatal("no error")
	}
}

func TestRemoveAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := DirFS(tmpDir)
	path := "dir"
	name := "test.txt"

	if err = os.Mkdir(tmpDir+"/"+path, fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(tmpDir+"/"+path+"/"+name, []byte{}, fs.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = wfs.RemoveAll(fsys, path)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveAll_InvalidError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fsys := DirFS(tmpDir)
	err = wfs.RemoveAll(fsys, "../invalid-dir")
	if err == nil {
		t.Fatal("no error")
	}
}

func TestReadLinkFS(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(filepath.Join(subDir, "dir"), fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"target.txt", "dir/inner.txt"} {
		if err := os.WriteFile(filepath.Join(subDir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("target.txt", filepath.Join(subDir, "file-link")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	if err := os.Symlink("dir", filepath.Join(subDir, "dir-link")); err != nil {
		t.Fatal(err)
	}
	absTarget := filepath.Join(subDir, "target.txt")
	if err := os.Symlink(absTarget, filepath.Join(subDir, "abs-link")); err != nil {
		t.Fatal(err)
	}

	root := New(tmpDir)
	// fstest.TestFS opens every entry, so it runs before the dangling link exists.
	if err := fstest.TestFS(root, "sub/file-link", "sub/dir-link"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(subDir, "dangling")); err != nil {
		t.Fatal(err)
	}
	sub, err := root.Sub("sub")
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		caseName   string
		fsys       fs.FS
		name       string
		wantLink   bool
		wantTarget string
		wantDir    bool
		wantErr    error
	}{
		{caseName: "link to file", fsys: root, name: "sub/file-link", wantLink: true, wantTarget: "target.txt"},
		{caseName: "link to directory", fsys: root, name: "sub/dir-link", wantLink: true, wantTarget: "dir", wantDir: true},
		{caseName: "dangling link", fsys: root, name: "sub/dangling", wantLink: true, wantTarget: "missing", wantErr: fs.ErrNotExist},
		{caseName: "absolute link", fsys: root, name: "sub/abs-link", wantLink: true, wantTarget: absTarget},
		{caseName: "regular file", fsys: root, name: "sub/target.txt"},
		{caseName: "file via directory link", fsys: root, name: "sub/dir-link/inner.txt"},
		{caseName: "directory", fsys: root, name: "sub/dir", wantDir: true},
		{caseName: "root", fsys: root, name: ".", wantDir: true},
		{caseName: "link to file via Sub", fsys: sub, name: "file-link", wantLink: true, wantTarget: "target.txt"},
		{caseName: "link to directory via Sub", fsys: sub, name: "dir-link", wantLink: true, wantTarget: "dir", wantDir: true},
		{caseName: "dangling link via Sub", fsys: sub, name: "dangling", wantLink: true, wantTarget: "missing", wantErr: fs.ErrNotExist},
		{caseName: "regular file via Sub", fsys: sub, name: "target.txt"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			info, err := fs.Lstat(tc.fsys, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode()&fs.ModeSymlink != 0; got != tc.wantLink {
				t.Errorf("Lstat(%q) symlink got %v; want %v", tc.name, got, tc.wantLink)
			}
			if want := path.Base(tc.name); info.Name() != want {
				t.Errorf("Lstat(%q) name got %q; want %q", tc.name, info.Name(), want)
			}

			target, err := fs.ReadLink(tc.fsys, tc.name)
			if tc.wantLink {
				if err != nil {
					t.Fatal(err)
				}
				if target != tc.wantTarget {
					t.Errorf("ReadLink(%q) got %q; want %q", tc.name, target, tc.wantTarget)
				}
			} else {
				var pathErr *fs.PathError
				if !errors.As(err, &pathErr) || pathErr.Path != tc.name {
					t.Errorf("ReadLink(%q) error got %#v; want *fs.PathError for %q", tc.name, err, tc.name)
				}
			}

			info, err = fs.Stat(tc.fsys, tc.name)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("Stat(%q) error got %v; want %v", tc.name, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				t.Errorf("Stat(%q) did not follow the link", tc.name)
			}
			if info.IsDir() != tc.wantDir {
				t.Errorf("Stat(%q) IsDir got %v; want %v", tc.name, info.IsDir(), tc.wantDir)
			}
		})
	}
}

func TestReadLinkFS_Error(t *testing.T) {
	testCases := []struct {
		caseName    string
		name        string
		wantErr     error
		windowsOnly bool
	}{
		{caseName: "missing", name: "missing", wantErr: fs.ErrNotExist},
		{caseName: "escapes the root", name: "../outside", wantErr: fs.ErrInvalid},
		{caseName: "absolute path", name: "/absolute", wantErr: fs.ErrInvalid},
		{caseName: "empty path", name: "", wantErr: fs.ErrInvalid},
		{caseName: "backslash", name: `a\b`, wantErr: fs.ErrInvalid, windowsOnly: true},
		{caseName: "colon", name: "c:x", wantErr: fs.ErrInvalid, windowsOnly: true},
	}

	fsys := New("testdata")
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if tc.windowsOnly && runtime.GOOS != "windows" {
				t.Skip("valid name outside windows")
			}
			var pathErr *fs.PathError

			_, err := fsys.Lstat(tc.name)
			if !errors.Is(err, tc.wantErr) || !errors.As(err, &pathErr) || pathErr.Path != tc.name {
				t.Errorf("Lstat(%q) error got %#v; want *fs.PathError for %q with %v", tc.name, err, tc.name, tc.wantErr)
			}
			_, err = fsys.ReadLink(tc.name)
			if !errors.Is(err, tc.wantErr) || !errors.As(err, &pathErr) || pathErr.Path != tc.name {
				t.Errorf("ReadLink(%q) error got %#v; want *fs.PathError for %q with %v", tc.name, err, tc.name, tc.wantErr)
			}
		})
	}
}
