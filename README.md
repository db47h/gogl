# gogl

gogl generates OpenGL and OpenGLES bindings for Go programs.

## Design goals

For simple use cases, [glow] does the job well. It is however not usable in the
following situations:

- Dual API support in a single package: OpenGL and OpenGLES. The generated
  package provides constants and functions to identify the available API at
  runtime.
- Enable writing part of the rendering code in C for better performance while
  still allowing calls to individual OpenGL or OpenGLES functions from Go
  (currently [glow] puts OpenGL function pointers into Go variables, which makes
  custom C code more complex than it should be).
- Most of the custom C code is written once, bundled in the generated package.

There is also the `gl` package from [gomobile]. It uses Go channels and a worker
goroutine that will batch multiple GL calls into a single cgo call. While it
enables OpenGL calls from any goroutine, there is no explicit control of the
batching mechanism and trading cgo calls for channel writes severely impacts
performance: on an AMD FX6300 a cgo call incurs a 80ns overhead vs. over 400ns
for a channel write (and 80ns vs 200ns on an i5 Skylake).

## Usage

### Installation

```bash
git clone https://github.com/db47h/gogl
cd gogl
go install
```

gogl was intended to generate internal packages. For example, to compile the
demo program:

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

Note that the `demo` build tag is only here to prevent `go get` and the likes to
try and compile the demo program.

By default on desktop, it will use the OpenGL API. You can however force the
OpenGLES 2 API by compiling with the `gles2` tag:

```bash
go run -tags "demo gles2"
```

### Using the generated API

In addition to Go wrappers for OpenGL and OpenGLES functions (same function name
minus the `gl` prefix), the generated package provides the following constants
and functions.

```go
// CoreProfile is true if the API was configured for the OpenGL core profile.
// The actual constant value depends on the requested API version and profile when
// generating the package.
//
// This is always false if API is GLES2.
//
const CoreProfile = true

// API type: OpenGL or OpenGLES.
//
type API int

// API Values.
//
const (
    OpenGL API = iota
    OpenGLES
)

func (a API) String() string

// Version represents an API version.
//
type Version struct {
    API   API
    Major int
    Minor int
}

// GE returns true if version v is greater or equal to Version{major, minor}.
//
// The following example shows how to use it in compatibility checks:
//
//  ver := gl.RuntimeVersion()
//  switch ver {
//  case ver.GE(OpenGL, 4, 0) || ver.GE(OpenGLES, 3, 1):
//      // call glDrawArraysIndirect
//  case ver.GE(OpenGL, 3, 1) || ver.GE(OpenGLES, 3, 0):
//      // call glDrawArraysInstanced
//  default:
//      // fallback
//  }
//
func (v Version) GE(api API, major, minor int) bool

// APIVersion returns the OpenGL or OpenGLES version supported by the package.
//
func APIVersion() Version

// RuntimeVersion returns the OpenGL or OpenGLES version available at runtime,
// which may differ from APIVersion.
//
func RuntimeVersion() Version

// InitC initializes OpenGL. loader is a function pointer to a C function of type
//
//  typedef void *(*loader) (const char *funcName)
//
// If API is GLES2, it is safe to pass a nil pointer to this function.
//
func InitC(loader unsafe.Pointer) error

// InitGo initializes OpenGL. The recommended value for loader is glfw.GetProcAddress.
// The loader function must panic on error.
//
// If API is GLES2, it is safe to pass a nil pointer to this function.
//
func InitGo(loader func(string) unsafe.Pointer)

```

After setting up an OpenGL context (for example after calling
`window.MakeContextCurrent()`), client code must call either `InitC` or `InitGo`
with an appropriate loader function. The difference between the two functions is
that for `InitC`, the loader function is a C function, while it is a Go function
for `InitGo`.

When using the OpenGL API, the `InitC` and `InitGo` functions will lookup
functions only for the API available at runtime. For example, if the package was
generated for OpenGL 4.5 core profile but only version 4.5 is available at
runtime, the C function `glSpecializeShader` will not be looked up. The Go
function `SpecializeShader` will still be available (since it was generated at
compile time) but will end up calling a nil pointer. Client code must therefore
check API compatibility and act accordingly (either bail out or work around
unavailable API calls).

### Customizing the generated packages

See [demo/internal/gl/custom.go](demo/internal/gl/custom.go) for an example.

In order to call OpenGL functions from Go, you need to call the corresponding
generated function, for example:

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

C code can use the GLVersion struct in order to query the runtime OpenGL or
OpenGLES version along with the mutually exclusive `GOTAG_gl` and `GOTAG_gles`
defines for the API type.

## TODO

TODOs and issues in no particular order.

- [ ] (may be) Use a `context` wrapper structure and interface to check
  available functions in a more Go-ish way:

  ```go
  switch ctx = ctx.(type) {
    case gl.Context_ES3_3_1:
        // call glDrawArraysIndirect
    case gl.Context_ES3_3_0:
        // call glDrawArraysInstanced
    default:
        // fallback to the hard way...
  }
  ```

- [ ] Fix issue with type GLhandleARB that is of a different size on macOS vs.
  the rest of the world... Not an issue until extension support is added.
- [ ] Compile flags/tags for Windows, macOS, Android and iOS and proper
  automatic detection of the GL or GLES API at compile time.
- [ ] Extend the demo project to compile with gomobile.
- [ ] Add support for Raspberry Pi.
- [ ] Parse and generate C types from gl.xml.
- [ ] (may be) create appropriate Go types with a C() function that converts to
  the proper C type (note that strings are tricky to handle automatically in a
  proper and efficient way).
- [ ] Option for GLES3.
- [ ] Handle extensions.
- [ ] Get rid of `khrplatform.h`.
- [ ] Provide a loader function.

Do not hesitate to contribute! Especially if you can test Windows, macOS or iOS.

## License

The generated code is subject to the (MIT) license in the generated
`khrplatform.h` file. Other generated files are not licensed.

The gogl program itself is released under the MIT license (with an exception for
the generated code as mentioned above). See the [LICENSE](LICENSE) file.


[glow]: https://github.com/go-gl/glow
[gomobile]: https://godoc.org/golang.org/x/mobile