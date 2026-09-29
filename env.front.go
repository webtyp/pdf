//go:build wasm
// +build wasm

package pdf

import (
	"encoding/base64"
	"syscall/js"
	"webtyp.com/fetch"
	"webtyp.com/files"
	. "webtyp.com/fmt"
	"webtyp.com/fmt/lang"
)

// initIO inicializa las funciones de IO para entorno frontend (wasm)
func (d *Document) initIO() {
	// Inicializar logger para frontend usando console.log
	d.logger = func(message ...any) {
		console := js.Global().Get("console")
		if !console.IsUndefined() {
			console.Call("log", lang.Translate(message...))
		}
	}
}

// browserFiles is the browser implementation of files.ReadWriter: it reads static resources
// (fonts, images) with fetch, and writing a file stores a copy in localStorage and hands the
// user a download.
type browserFiles struct{}

// defaultFiles is where a Document reads fonts and images and writes its output unless
// WithFiles says otherwise.
func defaultFiles() files.ReadWriter { return browserFiles{} }

// WriteFile escribe un archivo en localStorage y dispara una descarga
func (browserFiles) WriteFile(filePath string, content []byte) error {
	// 1. Guardar en localStorage (como backup/persistencia simple)
	localStorage := js.Global().Get("localStorage")
	if !localStorage.IsUndefined() {
		encoded := base64.StdEncoding.EncodeToString(content)
		localStorage.Call("setItem", filePath, encoded)
	}

	// 2. Disparar descarga del navegador
	uint8Array := js.Global().Get("Uint8Array").New(len(content))
	js.CopyBytesToJS(uint8Array, content)

	blob := js.Global().Get("Blob").New([]any{uint8Array}, map[string]any{"type": "application/pdf"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)

	link := js.Global().Get("document").Call("createElement", "a")
	link.Set("href", url)
	link.Set("download", filePath)
	link.Call("click")

	return nil
}

// ReadFile lee un archivo usando fetch (para cargar recursos estáticos como fuentes e imágenes).
// Un 404 es files.ErrNotExist.
func (browserFiles) ReadFile(filePath string) ([]byte, error) {
	ch := make(chan struct {
		data []byte
		err  error
	}, 1)

	fetch.Get(filePath).Send(func(resp *fetch.Response, err error) {
		if err != nil {
			ch <- struct {
				data []byte
				err  error
			}{nil, err}
			return
		}
		if resp.Status == 404 {
			ch <- struct {
				data []byte
				err  error
			}{nil, files.ErrNotExist}
			return
		}
		if resp.Status != 200 {
			ch <- struct {
				data []byte
				err  error
			}{nil, Errf("error fetching file %s: status %d", filePath, resp.Status)}
			return
		}
		ch <- struct {
			data []byte
			err  error
		}{resp.Body(), nil}
	})

	res := <-ch
	return res.data, res.err
}
