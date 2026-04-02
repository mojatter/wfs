// Package wfstest implements support for testing implementations and users of file systems.
package wfstest

import (
	"fmt"
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
				f.Close()
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

func checkFileWrite(fsys fs.FS, f wfs.WriterFile, name string) error {
	ps := [][]byte{[]byte("hello"), []byte(",world")}
	data := append(ps[0], ps[1]...)

	nn := 0
	for _, p := range ps {
		n, err := f.Write(p)
		if err != nil {
			f.Close()
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
