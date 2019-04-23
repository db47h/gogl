# gogl

gogl generates OpenGL and OpenGLES bindings for Go programs.

## Design goals

For simple use cases, [glow] does the job well.

- Dual API support in a single package: OpenGL and OpenGLES. The generated
  package provides constants and functions to identify the available API at
  runtime.
- Customizable API: allow writing C functions that aggregate multiple OpenGL calls into a single cgo call. Currently [glow] puts OpenGL function pointers into Go variables, which makes custom C code more complex than it should be.

## Usage

### Installation

```bash
git clone https://github.com/db47h/gogl
cd gogl
go install
```

gogl was intended to generate internal packages. For example, to compile the demo program:

```bash
cd demo
gogl -v 3.3 -profile core -o internal/gl
go run -tags demo .
```

Or better:

```bash
cd demo
go generate -tags demo
go run -tags demo .
```

Note that the `-tags demo` is only here to prevent `go get` and the likes to try and compile the demo program.

### Customizing the generated packages

See [demo/internal/gl/custom.go](demo/internal/gl/custom.go) for an example.

In order to call OpenGL functions from Go, you need to call the corresponding generated function, for example:

```go
func GetGoString(name uint32) string {
	return C.GoString((*C.char)(unsafe.Pointer(GetString(name))))
}
```

or, write a C wrapper (this is because cgo cannot call C function pointers):

```go
/*
#include "gl.h"

static const char *clear(GLbitfield mask, GLclampf r, GLclampf g, GLclampf b) {
	glClearColor(r, g, b, 1.0);
	glClear(mask);
}
*/
import "C"

func CustomClear(mask uint32, r, g, b float32) {
	C.clear(C.GLbitfield(mask), C.GLclampf(r), C.GLclampf(g), C.GLclampf(b))
}
```

This is not strictly necessary with GLES2 where function pointers are resolved
at link time (except for extensions). The above example is however portable
between both APIs and demonstrates how to aggregate multiple OpenGL calls into a
single cgo call.

## TODO

- [ ] Parse and generate C types from the XML
- [ ] (may be) create appropriate Go types with a C() function that converts to
  the proper C type (note that strings are tricky to handle automatically in a
  proper and efficient way).
- [ ] Option for GLES3.
- [ ] Handle extensions.
- [ ] Compile flags for Windows, macOS, Android and iOS.
- [ ] Get rid of `khrplatform.h`.

## License

The generated code is subject to the (MIT) license in the generated `khrplatform.h` file. Other generated files
are not licensed.

The gogl program itself is released under the MIT license (with an exception for the generated code as mentioned above). See the [LICENSE](LICENSE) file.


[glow]: https://github.com/go-gl/glow