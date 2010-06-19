#include <png.h>
#include <stdlib.h>
#include <stdio.h>
#include "opengl.h"

//-------------------------------------------------------------------------
// png loading
//-------------------------------------------------------------------------

struct argb32_data {
	void *bytes;
	unsigned int width;
	unsigned int height;
};

static void user_error(png_structp png_ptr, png_const_charp msg)
{
	longjmp(png_jmpbuf(png_ptr), 1);
}

static void PNGAPI user_read(png_structp png_ptr, png_bytep data, png_size_t length)
{
	uint8_t **ptr = (uint8_t**)png_get_io_ptr(png_ptr);
	memcpy(data, *ptr, length);
	*ptr = *ptr + length;
}

static bool load_png_argb32_data(void **u8ptr, struct argb32_data *td)
{
	png_byte *header = (png_byte*)*u8ptr;
	*u8ptr += 8;

	if (png_sig_cmp(header, 0, 8))
		return false;

	png_structp png_ptr = png_create_read_struct(PNG_LIBPNG_VER_STRING, 0, user_error, 0);

	if (!png_ptr)
		return false;

	png_infop info_ptr = png_create_info_struct(png_ptr);

	if (!info_ptr) {
		png_destroy_read_struct(&png_ptr, 0, 0);
		return false;
	}

	if (setjmp(png_jmpbuf(png_ptr))) {
		png_destroy_read_struct(&png_ptr, &info_ptr, 0);
		return false;
	}

	png_set_read_fn(png_ptr, u8ptr, user_read);
	png_set_sig_bytes(png_ptr, 8);
	png_read_info(png_ptr, info_ptr);

	png_uint_32 width = 0;
	png_uint_32 height = 0;
	int bpp = 0;
	int colortype = 0;

	png_get_IHDR(png_ptr, info_ptr, (png_uint_32*)&width, (png_uint_32*)&height, &bpp, &colortype, 0, 0, 0);

	// basically, we convert here every possible variant to the 32-bit RGBA.
	if (colortype == PNG_COLOR_TYPE_PALETTE)
		png_set_palette_to_rgb(png_ptr);

	if (bpp < 8) {
		if (colortype == PNG_COLOR_TYPE_GRAY || colortype == PNG_COLOR_TYPE_GRAY_ALPHA)
			png_set_expand_gray_1_2_4_to_8(png_ptr);
		else
			png_set_packing(png_ptr);
	}

	if (png_get_valid(png_ptr, info_ptr, PNG_INFO_tRNS))
		png_set_tRNS_to_alpha(png_ptr);

	if (bpp == 16)
		png_set_strip_16(png_ptr);

	if (colortype == PNG_COLOR_TYPE_GRAY || colortype == PNG_COLOR_TYPE_GRAY_ALPHA)
		png_set_gray_to_rgb(png_ptr);

	if (colortype != PNG_COLOR_TYPE_RGB_ALPHA)
		png_set_add_alpha(png_ptr, 0xFFFF, PNG_FILLER_AFTER);

	png_read_update_info(png_ptr, info_ptr);
	png_get_IHDR(png_ptr, info_ptr, (png_uint_32*)&width, (png_uint_32*)&height,&bpp, &colortype, 0, 0, 0);

	td->bytes = malloc(width * height * 4);
	td->width = width;
	td->height = height;

	uint8_t **row_pointers = malloc(sizeof(uint8_t*) * height);
	uint8_t *buffer = td->bytes;
	unsigned int i;

	for (i = 0; i < height; ++i) {
		row_pointers[i] = buffer;
		buffer += width * 4;
	}

	png_read_image(png_ptr, row_pointers);
	png_read_end(png_ptr, 0);

	png_destroy_read_struct(&png_ptr, &info_ptr, 0);
	free(row_pointers);
	return true;
}

bool load_texture_png_argb32(texture_t *out, void **u8ptr)
{
	struct argb32_data data;
	bool result = load_png_argb32_data(u8ptr, &data);
	if (!result)
		return false;

	GLuint id = 0;
	glGenTextures(1, &id);
	glBindTexture(GL_TEXTURE_2D, id);
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_CLAMP_TO_EDGE);
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_CLAMP_TO_EDGE);
	glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_R, GL_CLAMP_TO_EDGE);
	glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, data.width, data.height, 0, GL_RGBA,
		     GL_UNSIGNED_BYTE, data.bytes);

	free(data.bytes);

	if (glGetError() != GL_NO_ERROR) {
		glDeleteTextures(1, &id);
		return false;
	}

	out->id = id;
	out->width = data.width;
	out->height = data.height;

	return true;
}

void free_texture(texture_t *t)
{
	glDeleteTextures(1, &t->id);
	t->id = t->width = t->height = 0;
}
