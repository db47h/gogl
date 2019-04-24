//go:generate go run .. -o internal/gl -v 3.3 -profile core

// +build demo

package main

import (
	"log"
	"runtime"

	"github.com/db47h/gogl/demo/internal/gl"

	"github.com/go-gl/glfw/v3.2/glfw"
)

func init() {
	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	apiVer := gl.APIVersion()
	switch apiVer.API {
	case gl.OpenGL:
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI)
	case gl.OpenGLES:
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
	default:
		panic("unsupported API")
	}
	glfw.WindowHint(glfw.ContextVersionMajor, apiVer.Major)
	glfw.WindowHint(glfw.ContextVersionMinor, apiVer.Minor)
	if gl.CoreProfile {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	}
	glfw.WindowHint(glfw.Samples, 4)

	window, err := glfw.CreateWindow(800, 600, "Testing", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	gl.InitGo(glfw.GetProcAddress)

	log.Print(glfw.GetVersionString())
	ver := gl.RuntimeVersion()
	log.Printf("Runtime API: %s %s %d.%d", gl.GetGoString(gl.GL_VENDOR), ver.API.String(), ver.Major, ver.Minor)

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		switch key {
		case glfw.KeyEscape:
			if action == glfw.Press {
				w.SetShouldClose(true)
			}
		}
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	for !window.ShouldClose() {
		gl.CustomClear(gl.GL_COLOR_BUFFER_BIT, 0, 0, 0.5)
		// C.doClear()
		// Do OpenGL stuff.
		window.SwapBuffers()
		glfw.PollEvents()
	}
}
