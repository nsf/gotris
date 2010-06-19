all: gotris dejavu.font

include $(GOROOT)/src/Make.$(GOARCH)

TARG=gotris
GOFILES=gotris.go font.go

include $(GOROOT)/src/Make.cmd

dejavu.font: Tools/Makefile
	make -C Tools dejavu.font
	cp Tools/dejavu.font .

#CLEANFILES+=dejavu.font
