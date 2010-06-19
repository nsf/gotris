#pragma once

#include <GL/glew.h>
#include <stdbool.h>

typedef struct {
	GLuint id;
	int width;
	int height;
} texture_t;

bool load_texture_png_argb32(texture_t *out, void **p);
void free_texture(texture_t *t);
