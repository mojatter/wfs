// Package wfstest implements support for testing implementations and users of file systems.
package wfstest

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing/iotest"

	"github.com/mojatter/wfs"
)

// TestWriteFileFS tests a wfs.WriteFileFS implementation.
//
// Typical usage inside a test is:
//
//	tmpDir, err := os.MkdirTemp("", "test")
//	if err != nil {
//	  t.Fatal(err)
//	}
//	defer os.RemoveAll(tmpDir)
//
//	fsys := osfs.New(filepath.Dir(tmpDir))
//	if err := wfstest.TestWriteFileFS(fsys, filepath.Base(tmpDir)); err != nil {
//	  t.Fatal(err)
//	}
func TestWriteFileFS(fsys fs.FS, tmpDir string) error {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "file.txt", // simple create file.
		}, {
			name: "dir/file.txt", // mkdir and create file.
		}, {
			name:    "dir", // dir is exists that is a directory.
			wantErr: true,
		}, {
			name:    "dir/file.txt/invalid", // dir/file.txt is exists that is a file.
			wantErr: true,
		}, {
			name:    "file.txt/.", // invalid path.
			wantErr: true,
		}, {
			name: "dir/file.txt", // update file.
		},
	}
	for _, test := range tests {
		name := tmpDir + "/" + test.name

		f, err := wfs.CreateFile(fsys, name, fs.ModePerm)
		if test.wantErr {
			if err == nil {
				_ = f.Close()
				return fmt.Errorf("%s: CreateFile returns no error", name)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("%s: CreateFile: %v", name, err)
		}

		if err := checkFileWrite(fsys, f, name); err != nil {
			return err
		}
	}
	if err := wfs.RemoveFile(fsys, tmpDir+"/file.txt"); err != nil {
		return fmt.Errorf("%s: RemoveFile: %v", "file.txt", err)
	}
	if err := wfs.RemoveAll(fsys, tmpDir+"/dir"); err != nil {
		return fmt.Errorf("%s: RemoveAll: %v", "dir", err)
	}
	return nil
}

// TestRenameFS tests a wfs.RenameFS implementation. It assumes the filesystem
// also implements wfs.WriteFileFS so it can stage source files. tmpDir is a
// directory the test may freely create and destroy entries under.
func TestRenameFS(fsys fs.FS, tmpDir string) error {
	src := tmpDir + "/rename_src.txt"
	dst := tmpDir + "/rename_dst.txt"
	data := []byte("payload")

	if _, err := wfs.WriteFile(fsys, src, data, fs.ModePerm); err != nil {
		return fmt.Errorf("WriteFile %s: %v", src, err)
	}
	if err := wfs.Rename(fsys, src, dst); err != nil {
		return fmt.Errorf("rename %s -> %s: %v", src, dst, err)
	}
	got, err := wfs.ReadFile(fsys, dst)
	if err != nil {
		return fmt.Errorf("read %s after rename: %v", dst, err)
	}
	if string(got) != string(data) {
		return fmt.Errorf("rename: content got %q; want %q", got, data)
	}
	if _, err := fsys.Open(src); err == nil {
		return fmt.Errorf("rename: source %s still exists", src)
	}

	// Rename over an existing file should replace it.
	other := tmpDir + "/rename_other.txt"
	other2 := []byte("other-payload")
	if _, err := wfs.WriteFile(fsys, other, other2, fs.ModePerm); err != nil {
		return fmt.Errorf("write %s: %v", other, err)
	}
	if err := wfs.Rename(fsys, other, dst); err != nil {
		return fmt.Errorf("rename overwrite %s -> %s: %v", other, dst, err)
	}
	got, err = wfs.ReadFile(fsys, dst)
	if err != nil {
		return fmt.Errorf("read %s after overwrite: %v", dst, err)
	}
	if string(got) != string(other2) {
		return fmt.Errorf("rename overwrite: content got %q; want %q", got, other2)
	}

	// Rename of a missing file should fail.
	if err := wfs.Rename(fsys, tmpDir+"/does_not_exist.txt", tmpDir+"/whatever.txt"); err == nil {
		return fmt.Errorf("rename missing source: expected error, got nil")
	}

	if err := wfs.RemoveFile(fsys, dst); err != nil {
		return fmt.Errorf("RemoveFile %s: %v", dst, err)
	}
	return nil
}

// TestFileHandle tests that open files behave as on a POSIX OS: Write from
// Open, CreateFile truncation, writes across Rename and RemoveFile, and use
// after Close. It assumes wfs.WriteFileFS, wfs.RemoveFileFS and wfs.RenameFS;
// tmpDir is a directory it may freely create and destroy entries under.
func TestFileHandle(fsys fs.FS, tmpDir string) error {
	testCases := []struct {
		caseName string
		run      func(dir string) error
	}{
		{
			caseName: "Write on a file from Open fails",
			run: func(dir string) error {
				return checkOpenedWrite(fsys, dir+"/a.txt", false)
			},
		},
		{
			caseName: "Write on a directory from Open fails",
			run: func(dir string) error {
				return checkOpenedWrite(fsys, dir+"/d", true)
			},
		},
		{
			caseName: "Read and Stat on a file from CreateFile",
			run: func(dir string) error {
				f, err := createAndWrite(fsys, dir+"/a.txt", "hello")
				if err != nil {
					return err
				}
				defer f.Close()

				if n, err := f.Read(make([]byte, 2)); n != 0 || err != io.EOF {
					return fmt.Errorf("read = (%d, %v); want (0, EOF)", n, err)
				}
				info, err := f.Stat()
				if err != nil {
					return fmt.Errorf("stat: %v", err)
				}
				if info.Name() != "a.txt" || info.Size() != 5 {
					return fmt.Errorf("stat = (%q, %d); want (%q, 5)", info.Name(), info.Size(), "a.txt")
				}
				return nil
			},
		},
		{
			caseName: "CreateFile truncates an existing file",
			run: func(dir string) error {
				name := dir + "/a.txt"
				if _, err := wfs.WriteFile(fsys, name, []byte("old content"), fs.ModePerm); err != nil {
					return fmt.Errorf("write file: %v", err)
				}
				f, err := wfs.CreateFile(fsys, name, fs.ModePerm)
				if err != nil {
					return fmt.Errorf("create: %v", err)
				}
				if err := checkContent(fsys, name, ""); err != nil {
					_ = f.Close()
					return fmt.Errorf("before Close: %v", err)
				}
				if err := f.Close(); err != nil {
					return fmt.Errorf("close: %v", err)
				}
				return checkContent(fsys, name, "")
			},
		},
		{
			caseName: "Close without a write keeps later content",
			run: func(dir string) error {
				name := dir + "/a.txt"
				f, err := createAndWrite(fsys, name, "")
				if err != nil {
					return err
				}
				if _, err := wfs.WriteFile(fsys, name, []byte("new"), fs.ModePerm); err != nil {
					_ = f.Close()
					return fmt.Errorf("write file: %v", err)
				}
				if err := f.Close(); err != nil {
					return fmt.Errorf("close: %v", err)
				}
				return checkContent(fsys, name, "new")
			},
		},
		{
			caseName: "Write follows Rename",
			run: func(dir string) error {
				return checkWriteAcross(fsys, dir, func() error {
					return wfs.Rename(fsys, dir+"/a.txt", dir+"/c.txt")
				}, map[string]string{"c.txt": "hello"})
			},
		},
		{
			caseName: "Write vanishes after RemoveFile",
			run: func(dir string) error {
				return checkWriteAcross(fsys, dir, func() error {
					return wfs.RemoveFile(fsys, dir+"/a.txt")
				}, nil)
			},
		},
		{
			caseName: "Write vanishes after Rename onto its name",
			run: func(dir string) error {
				return checkWriteAcross(fsys, dir, func() error {
					if _, err := wfs.WriteFile(fsys, dir+"/b.txt", []byte("other"), fs.ModePerm); err != nil {
						return err
					}
					return wfs.Rename(fsys, dir+"/b.txt", dir+"/a.txt")
				}, map[string]string{"a.txt": "other"})
			},
		},
		{
			caseName: "Write vanishes after RemoveAll of its parent",
			run: func(dir string) error {
				sub := dir + "/sub"
				if err := checkWriteAcross(fsys, sub, func() error {
					return wfs.RemoveAll(fsys, sub)
				}, nil); err != nil {
					return err
				}
				if _, err := fs.Stat(fsys, sub); !errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf("stat %s: got %v; want ErrNotExist", sub, err)
				}
				return nil
			},
		},
		{
			caseName: "file from Open after RemoveFile",
			run: func(dir string) error {
				name := dir + "/a.txt"
				if _, err := wfs.WriteFile(fsys, name, []byte("hello"), fs.ModePerm); err != nil {
					return fmt.Errorf("write file: %v", err)
				}
				f, err := fsys.Open(name)
				if err != nil {
					return fmt.Errorf("open: %v", err)
				}
				defer f.Close()

				if err := wfs.RemoveFile(fsys, name); err != nil {
					return fmt.Errorf("remove: %v", err)
				}
				info, err := f.Stat()
				if err != nil || info.Size() != 5 {
					return fmt.Errorf("stat = (%v, %v); want size 5", info, err)
				}
				b, err := io.ReadAll(f)
				if err != nil || string(b) != "hello" {
					return fmt.Errorf("read all = (%q, %v); want %q", b, err, "hello")
				}
				return nil
			},
		},
		{
			caseName: "methods fail after Close",
			run: func(dir string) error {
				name := dir + "/a.txt"
				w, err := createAndWrite(fsys, name, "hello")
				if err != nil {
					return err
				}
				if err := w.Close(); err != nil {
					return fmt.Errorf("close: %v", err)
				}
				r, err := fsys.Open(name)
				if err != nil {
					return fmt.Errorf("open: %v", err)
				}
				if err := r.Close(); err != nil {
					return fmt.Errorf("close: %v", err)
				}
				if err := wfs.MkdirAll(fsys, dir+"/d", fs.ModePerm); err != nil {
					return fmt.Errorf("mkdir: %v", err)
				}
				d, err := fsys.Open(dir + "/d")
				if err != nil {
					return fmt.Errorf("open: %v", err)
				}
				if err := d.Close(); err != nil {
					return fmt.Errorf("close: %v", err)
				}

				_, writeErr := w.Write([]byte("x"))
				_, readErr := r.Read(make([]byte, 1))
				_, statErr := r.Stat()
				ops := map[string]error{
					"Write": writeErr, "Read": readErr, "Stat": statErr,
					"Close of a file from CreateFile": w.Close(), "Close of a file from Open": r.Close(),
				}
				if s, ok := w.(wfs.SyncWriterFile); ok {
					ops["Sync"] = s.Sync()
				}
				for op, err := range ops {
					if !errors.Is(err, fs.ErrClosed) {
						return fmt.Errorf("%s after Close = %v; want ErrClosed", op, err)
					}
				}
				if rd, ok := d.(fs.ReadDirFile); ok {
					// osfs reports "use of closed file" here rather than fs.ErrClosed.
					if _, err := rd.ReadDir(-1); err == nil {
						return fmt.Errorf("read dir after Close returns no error")
					}
				}
				return checkContent(fsys, name, "hello")
			},
		},
	}
	for i, tc := range testCases {
		dir := fmt.Sprintf("%s/handle%d", tmpDir, i)
		if err := wfs.MkdirAll(fsys, dir, fs.ModePerm); err != nil {
			return fmt.Errorf("%s: MkdirAll: %v", tc.caseName, err)
		}
		if err := tc.run(dir); err != nil {
			return fmt.Errorf("%s: %v", tc.caseName, err)
		}
		if err := wfs.RemoveAll(fsys, dir); err != nil {
			return fmt.Errorf("%s: RemoveAll: %v", tc.caseName, err)
		}
	}
	return nil
}

func checkOpenedWrite(fsys fs.FS, name string, isDir bool) error {
	var err error
	if isDir {
		err = wfs.MkdirAll(fsys, name, fs.ModePerm)
	} else {
		_, err = wfs.WriteFile(fsys, name, []byte("hello"), fs.ModePerm)
	}
	if err != nil {
		return fmt.Errorf("create %s: %v", name, err)
	}
	f, err := fsys.Open(name)
	if err != nil {
		return fmt.Errorf("open: %v", err)
	}
	if w, ok := f.(io.Writer); ok {
		if _, err := w.Write([]byte("XY")); err == nil {
			_ = f.Close()
			return fmt.Errorf("write returns no error")
		}
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close: %v", err)
	}
	if isDir {
		return nil
	}
	return checkContent(fsys, name, "hello")
}

func createAndWrite(fsys fs.FS, name, data string) (wfs.WriterFile, error) {
	f, err := wfs.CreateFile(fsys, name, fs.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("create: %v", err)
	}
	if _, err := f.Write([]byte(data)); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("write: %v", err)
	}
	return f, nil
}

// checkWriteAcross writes dir/a.txt, runs change, closes it, and checks a.txt and c.txt against want.
func checkWriteAcross(fsys fs.FS, dir string, change func() error, want map[string]string) error {
	if err := wfs.MkdirAll(fsys, dir, fs.ModePerm); err != nil {
		return fmt.Errorf("mkdir: %v", err)
	}
	f, err := createAndWrite(fsys, dir+"/a.txt", "hello")
	if err != nil {
		return err
	}
	if err := change(); err != nil {
		_ = f.Close()
		return fmt.Errorf("change: %v", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close: %v", err)
	}
	for _, base := range []string{"a.txt", "c.txt"} {
		name := dir + "/" + base
		w, ok := want[base]
		if !ok {
			if _, err := fs.Stat(fsys, name); !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("stat %s: got %v; want ErrNotExist", name, err)
			}
			continue
		}
		if err := checkContent(fsys, name, w); err != nil {
			return err
		}
	}
	return nil
}

func checkContent(fsys fs.FS, name, want string) error {
	b, err := wfs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read %s: %v", name, err)
	}
	if string(b) != want {
		return fmt.Errorf("read %s: got %q; want %q", name, b, want)
	}
	return nil
}

func checkFileWrite(fsys fs.FS, f wfs.WriterFile, name string) error {
	ps := [][]byte{[]byte("hello"), []byte(",world")}
	data := append(ps[0], ps[1]...)

	nn := 0
	for _, p := range ps {
		n, err := f.Write(p)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("%s: WriterFile.Write: %v", name, err)
		}
		nn = nn + n
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("%s: WriterFile.Close: %v", name, err)
	}

	if nn != len(data) {
		return fmt.Errorf("%s: Write size got %d; want %d", name, nn, len(data))
	}

	r, err := fsys.Open(name)
	if err != nil {
		return fmt.Errorf("%s: Open: %v", name, err)
	}
	defer r.Close()
	if err := iotest.TestReader(r, data); err != nil {
		return fmt.Errorf("%s: failed TestReader:\n\t%s", name, strings.ReplaceAll(err.Error(), "\n", "\n\t"))
	}
	return nil
}
