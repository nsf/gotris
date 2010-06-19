package main

import (
	"gl"
	"opengl_util"
	"os"
	"io/ioutil"
	"bytes"
	"io"
	"encoding/binary"
)

//-------------------------------------------------------------------------
// FontGlyph
//-------------------------------------------------------------------------

type FontGlyph struct {
	OffsetX int32
	OffsetY int32
	Width uint32
	Height uint32

	// texture coords
	TX float32
	TY float32
	TX2 float32
	TY2 float32

	XAdvance uint32
}

type FontEncoding struct {
	Unicode uint32
	Index uint32
}

type Font struct {
	Glyphs []FontGlyph

	// I'm keeping it here because original font implementation
	// uses binary search lookups in that array, but here in Go I will 
	// simply use a map for that
	Encoding []FontEncoding
	Texture *opengl_util.Texture
	YAdvance uint32

	EncodingMap map[int]int
}

func LoadFontFromFile(filename string) (*Font, os.Error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return LoadFont(data)
}

func readLittleEndian(r io.Reader, data interface{}) {
	err := binary.Read(r, binary.LittleEndian, data)
	if err != nil {
		panic(err)
	}
}

func LoadFont(data []byte) (fontOut *Font, errOut os.Error) {
	defer func() {
		if err := recover(); err != nil {
			fontOut = nil
			errOut = err.(os.Error)
		}
	}()

	font := new(Font)
	buf := bytes.NewBuffer(data[4:]) // skip magic

	var glyphsNum uint32
	readLittleEndian(buf, &glyphsNum)
	readLittleEndian(buf, &font.YAdvance)

	font.Glyphs = make([]FontGlyph, glyphsNum)
	for i := 0; i < int(glyphsNum); i++ {
		readLittleEndian(buf, &font.Glyphs[i].OffsetX)
		readLittleEndian(buf, &font.Glyphs[i].OffsetY)
		readLittleEndian(buf, &font.Glyphs[i].Width)
		readLittleEndian(buf, &font.Glyphs[i].Height)
		readLittleEndian(buf, &font.Glyphs[i].TX)
		readLittleEndian(buf, &font.Glyphs[i].TY)
		readLittleEndian(buf, &font.Glyphs[i].TX2)
		readLittleEndian(buf, &font.Glyphs[i].TY2)
		readLittleEndian(buf, &font.Glyphs[i].XAdvance)
	}

	font.Encoding = make([]FontEncoding, glyphsNum)
	font.EncodingMap = make(map[int]int, glyphsNum)
	for i := 0; i < int(glyphsNum); i++ {
		readLittleEndian(buf, &font.Encoding[i].Unicode)
		readLittleEndian(buf, &font.Encoding[i].Index)

		font.EncodingMap[int(font.Encoding[i].Unicode)] =
			int(font.Encoding[i].Index)
	}

	data = buf.Bytes()
	font.Texture = opengl_util.LoadTexture_PNG_ARGB32(data)
	if font.Texture == nil {
		return nil, os.NewError("Failed to load texture")
	}

	return font, nil
}

func drawQuad(x, y, w, h int, u, v, u2, v2 float) {
	gl.Begin(gl.QUADS)

	gl.TexCoord2f(gl.GLfloat(u), gl.GLfloat(v))
	gl.Vertex2i(gl.GLint(x), gl.GLint(y))

	gl.TexCoord2f(gl.GLfloat(u2), gl.GLfloat(v))
	gl.Vertex2i(gl.GLint(x+w), gl.GLint(y))

	gl.TexCoord2f(gl.GLfloat(u2), gl.GLfloat(v2))
	gl.Vertex2i(gl.GLint(x+w), gl.GLint(y+h))

	gl.TexCoord2f(gl.GLfloat(u), gl.GLfloat(v2))
	gl.Vertex2i(gl.GLint(x), gl.GLint(y+h))

	gl.End()
}

func drawGlyph(x, y int, g *FontGlyph) {
	drawQuad(x + int(g.OffsetX), y + int(g.OffsetY), int(g.Width), int(g.Height),
		 float(g.TX), float(g.TY), float(g.TX2), float(g.TY2))
}

func (self *Font) Draw(x, y int, text string) {
	gl.BindTexture(gl.TEXTURE_2D, gl.GLuint(self.Texture.Id))
	for _, rune := range text {
		index, ok := self.EncodingMap[rune]
		if !ok {
			continue
		}

		g := &self.Glyphs[index - 1]
		drawGlyph(x, y, g)
		x += int(g.XAdvance)
	}
	gl.BindTexture(gl.TEXTURE_2D, 0)
}

func (self *Font) Width(text string) int {
	x := 0
	for _, rune := range text {
		index, ok := self.EncodingMap[rune]
		if !ok {
			continue
		}

		x += int(self.Glyphs[index - 1].XAdvance)
	}
	return x
}
