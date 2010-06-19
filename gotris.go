package main

import (
	"sdl"
	"gl"
	"rand"
)

const blockSize = 15
const smallBlockSize = 9
const smallBlockOffset = (blockSize - smallBlockSize) / 2

//-------------------------------------------------------------------------
// TetrisFigure
//-------------------------------------------------------------------------

// ah, the source code is utf-8, let's use some UNICODE box-drawing here:

// ████
//   ████
const specN = `
0110
1200
0000
0000
`
//   ████
// ████
const specNMirrored = `
1100
0210
0000
0000
`
//   ██
// ██████
const specT = `
0100
1210
0000
0000
`
// ████████
const specI = `
0100
0200
0100
0100
`
// ████
// ████
const specB = `
1100
1100
0000
0000
`
// ██████
// ██
const specL = `
0100
0200
0110
0000
`
// ██████
//     ██
const specLMirrored = `
0100
0200
1100
0000
`

type TetrisFigure struct {
	// center of the figure (valid range: 0..3 0..3)
	CenterX, CenterY int

	// position in blocks relative to top left tetris field block
	X, Y int
	Blocks [16]TetrisBlock
}

// build figure out of spec
func NewTetrisFigure(spec string, color TetrisBlockColor) *TetrisFigure {
	figure := new(TetrisFigure)
	figure.CenterX = -1
	figure.CenterY = -1

	i := 0
	for _, c := range spec {
		switch c {
		case '2':
			figure.CenterX = i % 4
			figure.CenterY = i / 4
			fallthrough
		case '1':
			figure.Blocks[i].Filled = true
			figure.Blocks[i].Color = color
			fallthrough
		case '0':
			i++
		}
	}
	return figure
}

func (self *TetrisFigure) SetColor(color TetrisBlockColor) {
	for i := 0; i < 16; i++ {
		if !self.Blocks[i].Filled {
			continue
		}

		self.Blocks[i].Color = color
	}
}

func rotateCWBlock(x, y int) (ox, oy int) {
	ox, oy = -y, x
	return
}

func rotateCCWBlock(x, y int) (ox, oy int) {
	ox, oy = y, -x
	return
}

type RotateFunc func(int, int) (int, int)

func (self *TetrisFigure) GetRotationsNum(rotateBlock RotateFunc) int {
	const (
		Rotate1 uint = 1 << iota
		Rotate2
		Rotate3
		Rotate4
	)
	validRotations := ^uint(0)
	// first we rotate each visible block four times around the center
	// and checking whether each rotation is valid, then we make a list
	// of valid rotation counts (like: [3, 4] or [1, 2, 3, 4])
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			blockMask := uint(0)
			if !self.Blocks[y * 4 + x].Filled {
				continue
			}
			blockX, blockY := x - self.CenterX, y - self.CenterY
			for i := 0; i < 4; i++ {
				blockX, blockY = rotateBlock(blockX, blockY)
				rbx, rby := self.CenterX + blockX, self.CenterY + blockY

				// check whether a rotation is valid an record it
				if rbx >= 0 && rbx <= 4 && rby >= 0 && rby <= 4 {
					blockMask |= 1 << uint(i)
				}
			}

			// apply mask to global mask
			validRotations &= blockMask
		}
	}

	// at this point we have valid rotations list
	rotationsNum := 0
	// determine number of rotations
	switch {
	case validRotations & Rotate1 > 0: rotationsNum = 1
	case validRotations & Rotate2 > 0: rotationsNum = 2
	case validRotations & Rotate3 > 0: rotationsNum = 3
	case validRotations & Rotate4 > 0: rotationsNum = 4
	}

	return rotationsNum
}

func (self *TetrisFigure) Rotate(rotateBlock RotateFunc) {
	// if there is no center, then the figure cannot be rotated
	if self.CenterX == -1 {
		return
	}

	rotationsNum := self.GetRotationsNum(rotateBlock)

	var newBlocks [16]TetrisBlock
	for i := 0; i < 16; i++ {
		if !self.Blocks[i].Filled {
			continue
		}
		x := i % 4
		y := i / 4
		x, y = x - self.CenterX, y - self.CenterY

		for j := 0; j < rotationsNum; j++ {
			x, y = rotateBlock(x, y)
		}

		x, y = x + self.CenterX, y + self.CenterY
		newBlocks[y * 4 + x] = self.Blocks[i]
	}
	self.Blocks = newBlocks
}

func (self *TetrisFigure) Draw(ox, oy int) {
	ox += (self.X + 1) * blockSize // skip tetris field wall also
	oy += self.Y * blockSize
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			offset := y * 4 + x
			self.Blocks[offset].Draw(ox + x * blockSize, oy + y * blockSize)
		}
	}
}

//-------------------------------------------------------------------------
// TetrisBlockColor
//-------------------------------------------------------------------------

type TetrisBlockColor struct {
	R, G, B byte
}

//-------------------------------------------------------------------------
// TetrisBlock
//-------------------------------------------------------------------------

type TetrisBlock struct {
	Filled bool
	Color TetrisBlockColor
}

func (self *TetrisBlock) Draw(x, y int) {
	if self.Filled {
		drawBlock(x, y, self.Color)
	}
}

//-------------------------------------------------------------------------
// TetrisField
//-------------------------------------------------------------------------

type TetrisField struct {
	Width int
	Height int
	Blocks []TetrisBlock
}

func NewTetrisField(w, h int) *TetrisField {
	return &TetrisField{w, h, make([]TetrisBlock, w*h)}
}

func (self *TetrisField) Draw(ox, oy int) {
	leftWallX := self.PixelsWidth() - blockSize
	grey := TetrisBlockColor{80, 80, 80}
	for y := 0; y < self.Height + 1; y++ {
		drawBlock(ox, oy + y * blockSize, grey)
		drawBlock(ox + leftWallX, oy + y * blockSize, grey)
	}
	bottomWallY := self.PixelsHeight() - blockSize
	for x := 0; x < self.Width; x++ {
		drawBlock(ox + (x + 1) * blockSize, oy + bottomWallY, grey)
	}

	ox += blockSize
	for y := 0; y < self.Height; y++ {
		for x := 0; x < self.Width; x++ {
			offset := y * self.Width + x
			self.Blocks[offset].Draw(ox + x * blockSize, oy + y * blockSize)
		}
	}
}

func (self *TetrisField) Collide(figure *TetrisFigure) bool {
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			offset := y * 4 + x
			if !figure.Blocks[offset].Filled {
				continue
			}

			fx, fy := figure.X + x, figure.Y + y
			if fx < 0 || fy < 0 || fx >= self.Width || fy >= self.Height {
				return true
			}
			fieldOffset := fy * self.Width + fx
			if self.Blocks[fieldOffset].Filled {
				return true
			}
		}
	}
	return false
}

func (self *TetrisField) StepCollideAndMerge(figure *TetrisFigure) bool {
	figure.Y++
	if !self.Collide(figure) {
		return false
	}
	figure.Y--

	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			offset := y * 4 + x
			if !figure.Blocks[offset].Filled {
				continue
			}
			fx, fy := figure.X + x, figure.Y + y
			fieldOffset := fy * self.Width + fx
			self.Blocks[fieldOffset] = figure.Blocks[offset]
		}
	}
	return true
}

// check if there are any complete lines on the field and remove them
// returns the number of lines removed
func (self *TetrisField) CheckForLines() int {
	lines := 0
	for y := 0; y < self.Height; y++ {
		full := true
		for x := 0; x < self.Width; x++ {
			offset := y * self.Width + x
			if !self.Blocks[offset].Filled {
				full = false
				break
			}
		}

		if !full {
			continue
		}
		// if the line is full, increment counter and move all those
		// that are above this line one line down
		lines++

		for y2 := y - 1; y2 >= 0; y2-- {
			for x := 0; x < self.Width; x++ {
				offset := y2 * self.Width + x
				self.Blocks[offset + self.Width] = self.Blocks[offset]
			}
		}
	}
	return lines
}

func (self *TetrisField) PixelsWidth() int {
	return (self.Width + 2) * blockSize
}

func (self *TetrisField) PixelsHeight() int {
	return (self.Height + 1) * blockSize
}

func drawBlock(x, y int, color TetrisBlockColor) {
	gl.Color3ub(gl.GLubyte(color.R/2),gl.GLubyte(color.G/2),gl.GLubyte(color.B/2))
	gl.Begin(gl.QUADS)
	gl.Vertex2i(gl.GLint(x            ), gl.GLint(y))
	gl.Vertex2i(gl.GLint(x + blockSize), gl.GLint(y))
	gl.Vertex2i(gl.GLint(x + blockSize), gl.GLint(y + blockSize))
	gl.Vertex2i(gl.GLint(x            ), gl.GLint(y + blockSize))
	gl.Color3ub(gl.GLubyte(color.R),gl.GLubyte(color.G),gl.GLubyte(color.B))
	gl.Vertex2i(gl.GLint(x + smallBlockOffset            ), gl.GLint(y + smallBlockOffset))
	gl.Vertex2i(gl.GLint(x + blockSize - smallBlockOffset), gl.GLint(y + smallBlockOffset))
	gl.Vertex2i(gl.GLint(x + blockSize - smallBlockOffset), gl.GLint(y + blockSize - smallBlockOffset))
	gl.Vertex2i(gl.GLint(x + smallBlockOffset            ), gl.GLint(y + blockSize - smallBlockOffset))
	gl.End()
}

func main() {
	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()

	sdl.GL_SetAttribute(sdl.GL_SWAP_CONTROL, 1)

	if sdl.SetVideoMode(640, 480, 32, sdl.OPENGL) == nil {
		panic("sdl error")
	}

	sdl.WM_SetCaption("Gotris", "Gotris")

	if gl.Init() != 0 {
		panic("glew error")
	}

	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Viewport(0, 0, 640, 480)
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Ortho(0, 640, 480, 0, -1, 1)

	gl.ClearColor(0.1, 0, 0, 0)

	//-----------------------------------------------------------------------------

	font, err := LoadFontFromFile("dejavu.font")
	if err != nil {
		panic(err)
	}

	field := NewTetrisField(10, 25)
	pw := field.PixelsWidth()
	ph := field.PixelsHeight()
	cx := (640 - pw) / 2
	cy := (480 - ph) / 2

	labelcx := (640 - font.Width("Tetris!")) / 2

	specs := [...]string{
		specN,
		specNMirrored,
		specT,
		specI,
		specB,
		specL,
		specLMirrored }

	f := NewTetrisFigure(specI, TetrisBlockColor{0,255,0})
	lastTime := sdl.GetTicks()
	timer := uint32(0)

	running := true
	for running {
		e := new(sdl.Event)
		for e.Poll() {
			switch e.Type {
			case sdl.QUIT:
				running = false
			case sdl.KEYDOWN:
				switch e.Keyboard().Keysym.Sym {
				case sdl.K_LEFT:
					f.X--
					if field.Collide(f) {
						f.X++
					}
				case sdl.K_RIGHT:
					f.X++
					if field.Collide(f) {
						f.X--
					}
				case sdl.K_UP:
					f.Rotate(rotateCWBlock)
					if field.Collide(f) {
						f.Rotate(rotateCCWBlock)
					}
				case sdl.K_DOWN:
					for {
						if field.Collide(f) {
							f.Y--
							break
						} else {
							f.Y++
						}
					}
				}
			}
		}

		now := sdl.GetTicks()
		delta := now - lastTime
		lastTime = now
		timer += delta
		if timer > 200 {
			timer -= 200
			if field.StepCollideAndMerge(f) {
				field.CheckForLines()
				f = NewTetrisFigure(specs[rand.Uint32() % uint32(len(specs))],
						    TetrisBlockColor{0,255,0})
			}
		}

		gl.Clear(gl.COLOR_BUFFER_BIT)
		font.Draw(labelcx, 5, "Tetris!")
		field.Draw(cx, cy)
		f.Draw(cx, cy)
		gl.Color3ub(255,255,255)
		sdl.GL_SwapBuffers()
	}
}
