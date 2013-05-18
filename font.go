package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/banthar/gl"
	"image"
	"image/png"
	"io"
	"io/ioutil"
)

func uploadTexture_NRGBA32(img *image.NRGBA) gl.Texture {
	b := img.Bounds()
	data := make([]uint8, b.Max.X*b.Max.Y*4)
	for y := 0; y < b.Max.Y; y++ {
		for x := 0; x < b.Max.X; x++ {
			p := img.At(x, y)
			offset := y*b.Max.X*4 + x*4
			r, g, b, a := p.RGBA()
			data[offset+0] = uint8(r)
			data[offset+1] = uint8(g)
			data[offset+2] = uint8(b)
			data[offset+3] = uint8(a)
		}
	}

	id := gl.GenTexture()
	id.Bind(gl.TEXTURE_2D)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, b.Max.X, b.Max.Y, 0, gl.RGBA, gl.UNSIGNED_BYTE, data)

	if gl.GetError() != gl.NO_ERROR {
		id.Delete()
		panic(errors.New("Failed to load a texture"))
		return 0
	}
	return id
}

//-------------------------------------------------------------------------
// FontGlyph
//-------------------------------------------------------------------------

type FontGlyph struct {
	OffsetX int32
	OffsetY int32
	Width   uint32
	Height  uint32

	// texture coords
	TX  float32
	TY  float32
	TX2 float32
	TY2 float32

	XAdvance uint32
}

type FontEncoding struct {
	Unicode uint32
	Index   uint32
}

type Font struct {
	Glyphs []FontGlyph

	// I'm keeping it here because original font implementation
	// uses binary search lookups in that array, but here in Go I will
	// simply use a map for that
	Encoding []FontEncoding
	Texture  gl.Texture
	YAdvance uint32

	EncodingMap map[rune]int
}

func LoadFontFromFile(filename string) (*Font, error) {
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

func LoadFont(data []byte) (fontOut *Font, errOut error) {
	defer func() {
		if err := recover(); err != nil {
			var ok bool
			fontOut = nil
			errOut, ok = err.(error)
			if !ok {
				panic(err)
			}
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
	font.EncodingMap = make(map[rune]int, glyphsNum)
	for i := 0; i < int(glyphsNum); i++ {
		readLittleEndian(buf, &font.Encoding[i].Unicode)
		readLittleEndian(buf, &font.Encoding[i].Index)

		font.EncodingMap[rune(font.Encoding[i].Unicode)] =
			int(font.Encoding[i].Index)
	}

	img, err := png.Decode(buf)
	if err != nil {
		return nil, err
	}

	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		return nil, errors.New("Wrong image format")
	}

	font.Texture = uploadTexture_NRGBA32(nrgba)
	return font, nil
}

func drawQuad(x, y, w, h int, u, v, u2, v2 float32) {
	gl.Begin(gl.QUADS)

	gl.TexCoord2f(float32(u), float32(v))
	gl.Vertex2i(int(x), int(y))

	gl.TexCoord2f(float32(u2), float32(v))
	gl.Vertex2i(int(x+w), int(y))

	gl.TexCoord2f(float32(u2), float32(v2))
	gl.Vertex2i(int(x+w), int(y+h))

	gl.TexCoord2f(float32(u), float32(v2))
	gl.Vertex2i(int(x), int(y+h))

	gl.End()
}

func drawGlyph(x, y int, g *FontGlyph) {
	drawQuad(x+int(g.OffsetX), y+int(g.OffsetY), int(g.Width), int(g.Height),
		float32(g.TX), float32(g.TY), float32(g.TX2), float32(g.TY2))
}

func (self *Font) Draw(x, y int, text string) {
	gl.BindTexture(gl.TEXTURE_2D, uint(self.Texture))
	for _, rune := range text {
		index, ok := self.EncodingMap[rune]
		if !ok {
			continue
		}

		g := &self.Glyphs[index-1]
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

		x += int(self.Glyphs[index-1].XAdvance)
	}
	return x
}
