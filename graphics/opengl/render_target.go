package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"fmt"
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/shader"
	"math"
)

func glMatrix(matrix [4][4]float64) [16]float32 {
	result := [16]float32{}
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			result[i+j*4] = float32(matrix[i][j])
		}
	}
	return result
}

type RenderTarget struct {
	framebuffer C.GLuint
	width       int
	height      int
	flipY       bool
}

func newRTWithCurrentFramebuffer(width, height int) *RenderTarget {
	framebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &framebuffer)
	return &RenderTarget{C.GLuint(framebuffer), width, height, true}
}

func createFramebuffer(nativeTexture C.GLuint) C.GLuint {
	framebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &framebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)

	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	defer C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))

	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, nativeTexture, 0)
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) !=
		C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	// Set this framebuffer opaque because alpha values on a target might be
	// confusing.
	C.glClearColor(0, 0, 0, 1)
	C.glClear(C.GL_COLOR_BUFFER_BIT)

	return framebuffer
}

func (r *RenderTarget) setAsViewport() {
	current := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &current)
	if C.GLuint(current) == r.framebuffer {
		return
	}

	C.glFlush()

	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(r.framebuffer))
	err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER)
	if err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}

	C.glBlendFuncSeparate(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA,
		C.GL_ZERO, C.GL_ONE)

	width := graphics.AdjustSizeForTexture(r.width)
	height := graphics.AdjustSizeForTexture(r.height)
	C.glViewport(0, 0, C.GLsizei(width), C.GLsizei(height))
}

func (r *RenderTarget) projectionMatrix() [4][4]float64 {
	width := graphics.AdjustSizeForTexture(r.width)
	height := graphics.AdjustSizeForTexture(r.height)
	matrix := graphics.OrthoProjectionMatrix(0, width, 0, height)
	if r.flipY {
		matrix[1][1] *= -1
		matrix[1][3] += float64(r.height) /
			float64(graphics.AdjustSizeForTexture(r.height)) * 2
	}
	return matrix
}

func (r *RenderTarget) dispose() {
	C.glDeleteFramebuffers(1, &r.framebuffer)
}

func (r *RenderTarget) fill(red, green, blue uint8) {
	r.setAsViewport()
	const max = float64(math.MaxUint8)
	C.glClearColor(
		C.GLclampf(float64(red)/max),
		C.GLclampf(float64(green)/max),
		C.GLclampf(float64(blue)/max),
		1)
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (r *RenderTarget) drawTexture(
	texture *Texture,
	parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) {
	r.setAsViewport()
	projectionMatrix := r.projectionMatrix()
	quads := graphics.TextureQuads(parts, texture.width, texture.height)
	shader.DrawTexture(
		shader.NativeTexture(texture.native),
		glMatrix(projectionMatrix),
		quads,
		geometryMatrix,
		colorMatrix)
}