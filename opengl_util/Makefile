include $(GOROOT)/src/Make.$(GOARCH)

OPENGL_TEXTURE_O=opengl_texture.o
CGO_LDFLAGS:=$(OPENGL_TEXTURE_O) $(shell pkg-config --libs libpng)

GOFILES:=\
	structures.$(O).go

TARG=opengl_util
CGOFILES=\
	 opengl_util.go

CLEANFILES+=structures.$(O).go

include $(GOROOT)/src/Make.pkg

$(OPENGL_TEXTURE_O): opengl_texture.c opengl.h
	gcc -fPIC -c $(shell pkg-config --cflags libpng) -o $(OPENGL_TEXTURE_O) opengl_texture.c

_cgo_.so: $(OPENGL_TEXTURE_O)

structures.$(O).go: structures.c
	godefs -g opengl_util structures.c > structures.$(O).go
	gofmt -w structures.$(O).go
