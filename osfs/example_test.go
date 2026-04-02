package osfs_test

import (
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/mojatter/wfs"
	"github.com/mojatter/wfs/osfs"
)

func ExampleDirFS() {
	tmpDir, err := os.MkdirTemp("", "example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	name := "example.txt"
	content := []byte(`Hello`)

	fsys := osfs.DirFS(tmpDir)
	_, err = wfs.WriteFile(fsys, name, content, fs.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	wrote, err := os.ReadFile(tmpDir + "/" + name)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", string(wrote))

	// Output: Hello
}
