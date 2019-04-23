// +build demo

package gl

/*
#include "gl.h"

static const char *clear(GLbitfield mask, GLclampf r, GLclampf g, GLclampf b) {
	glClearColor(r, g, b, 1.0);
	glClear(mask);
}
*/
import "C"
import "unsafe"

func GetGoString(name uint32) string {
	return C.GoString((*C.char)(unsafe.Pointer(GetString(name))))
}

func CustomClear(mask uint32, r, g, b float32) {
	C.clear(C.GLbitfield(mask), C.GLclampf(r), C.GLclampf(g), C.GLclampf(b))
}
