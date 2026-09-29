//go:build !wasm
// +build !wasm

package pdf

import (
	"webtyp.com/disk"
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

// defaultFiles is where a Document reads fonts and images and writes its output unless
// WithFiles says otherwise: the disk.
func defaultFiles() files.ReadWriter { return disk.Files{} }
