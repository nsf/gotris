package opengl_util

// #include "opengl.h"
import "C"
import "unsafe"

func LoadTexture_PNG_ARGB32(data []byte) *Texture {
	tex := new(Texture)
	ptr := unsafe.Pointer(&data[0])
	result := bool(C.load_texture_png_argb32((*C.texture_t)(unsafe.Pointer(tex)), &ptr))
	if result != true {
		return nil
	}
	return tex
}

func (tex *Texture) Free() {
	C.free_texture((*C.texture_t)(unsafe.Pointer(tex)))
}
