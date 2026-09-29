//go:build !wasm
// +build !wasm

package pdf

import (
	"os"

	"webtyp.com/files"
	"webtyp.com/fmt"
)

// initIO inicializa las funciones de IO para entorno backend (no-wasm)
func (d *Document) initIO() {
	// Inicializar logger para backend usando fmt.Println
	d.logger = func(message ...any) {
		fmt.Println(message...)
	}
}

// diskFiles is the server implementation of files.ReadWriter: plain files on disk.
type diskFiles struct{}

func (diskFiles) ReadFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return nil, files.ErrNotExist
	}
	return data, err
}

func (diskFiles) WriteFile(filePath string, content []byte) error {
	return os.WriteFile(filePath, content, 0644)
}

// defaultFiles is where a Document reads fonts and images and writes its output unless
// WithFiles says otherwise.
func defaultFiles() files.ReadWriter { return diskFiles{} }
