package main

import (
	"sdl"
	"gl"
	"rand"
	"fmt"
	"flag"
)

const blockSize = 15
const smallBlockSize = 9
const smallBlockOffset = (blockSize - smallBlockSize) / 2
const grayifyingInterval = 100

var initLevel *int = flag.Int("level", 1, "set initial level to this value (1..9)")

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

var specColors = [7]TetrisBlockColor{
	TetrisBlockColor{255,0,0},
	TetrisBlockColor{0,255,0},
	TetrisBlockColor{100,100,255},
	TetrisBlockColor{255,255,255},
	TetrisBlockColor{255,0,255},
	TetrisBlockColor{255,255,0},
	TetrisBlockColor{0,255,255}}

var specs = [7]string{
	specN,
	specNMirrored,
	specT,
	specI,
	specB,
	specL,
	specLMirrored}

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


//-------------------------------------------------------------------------
// TetrisFigure
//-------------------------------------------------------------------------

type TetrisFigure struct {
	// center of the figure (valid range: 0..3 0..3)
	CenterX, CenterY int

	// position in blocks relative to top left tetris field block
	X, Y int
	Blocks [16]TetrisBlock
	Class uint32
}

// build figure out of spec
func NewTetrisFigure(spec string, color TetrisBlockColor) *TetrisFigure {
	figure := new(TetrisFigure)
	figure.X = 3
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

func NewRandomTetrisFigure() *TetrisFigure {
	ri := rand.Uint32() % uint32(len(specs))
	f := NewTetrisFigure(specs[ri], specColors[ri])
	f.Class = ri
	return f
}

func NewRandomTetrisFigureNot(figure *TetrisFigure) *TetrisFigure {
	var ri uint32
	for {
		ri = rand.Uint32() % uint32(len(specs))
		if ri != figure.Class {
			break
		}
	}
	f := NewTetrisFigure(specs[ri], specColors[ri])
	f.Class = ri
	return f
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

func (self *TetrisField) Clear() {
	for i := 0; i < self.Width * self.Height; i++ {
		self.Blocks[i].Filled = false
	}
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

func (self *TetrisField) Grayify() {
	for i := 0; i < self.Width * self.Height; i++ {
		if !self.Blocks[i].Filled {
			continue
		}

		c := &self.Blocks[i].Color
		if c.R != 80 {
			if c.R < 80 {
				c.R++
			} else {
				c.R--
			}
		}
		if c.G != 80 {
			if c.G < 80 {
				c.G++
			} else {
				c.G--
			}
		}
		if c.B != 80 {
			if c.B < 80 {
				c.B++
			} else {
				c.B--
			}
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

//-------------------------------------------------------------------------
// GameSession
//-------------------------------------------------------------------------

// Game state
const (
	GS_Playing = iota
	GS_Paused
	GS_GameOver
)

type GameSession struct {
	Field *TetrisField
	Figure *TetrisFigure
	NextFigure *TetrisFigure

	Score int
	Level int
	State int

	time uint32
	grayifyingTime uint32
	cx, cy int
	initLevel int
	gameOverCx int
	pauseCx int
	font *Font
}

func NewGameSession(initLevel int, font *Font) *GameSession {
	if initLevel > 9 {
		initLevel = 9
	}
	if initLevel < 1 {
		initLevel = 1
	}

	gs := new(GameSession)
	gs.Field = NewTetrisField(10, 25)
	gs.Figure = NewRandomTetrisFigure()
	gs.NextFigure = NewRandomTetrisFigureNot(gs.Figure)
	gs.Score = 0
	gs.Level = initLevel
	gs.State = GS_Playing
	gs.time = 0
	gs.grayifyingTime = 0

	gs.initLevel = initLevel
	gs.font = font
	gs.cx = (640 - gs.Field.PixelsWidth()) / 2
	gs.cy = (480 - gs.Field.PixelsHeight()) / 2
	gs.gameOverCx = (640 - font.Width("Game Over, restart? y/n")) / 2
	gs.pauseCx = (640 - font.Width("Game paused, press P to resume")) / 2
	return gs
}

func (self *GameSession) Reset() {
	self.Field.Clear()
	self.Figure = NewRandomTetrisFigure()
	self.NextFigure = NewRandomTetrisFigureNot(self.Figure)
	self.Score = 0
	self.Level = self.initLevel
	self.State = GS_Playing
	self.time = 0
	self.grayifyingTime = 0
}

func (self *GameSession) Speed() uint32 {
	return uint32(1000 / self.Level)
}

func (self *GameSession) AddScore(score int) {
	self.Score += score * self.Level
	if self.Score > self.Level * self.Level * 10000 && self.Level < 9 {
		self.Level++
	}
}

//-------------------------------------------------------------------------
// GameSession::Update
//-------------------------------------------------------------------------

func (self *GameSession) updatePlaying(delta uint32) {
	self.time += delta
	self.grayifyingTime += delta
	if self.grayifyingTime > grayifyingInterval {
		self.grayifyingTime -= grayifyingInterval
		self.Field.Grayify()
	}
	if self.time > self.Speed() {
		self.time -= self.Speed()
		if self.Field.StepCollideAndMerge(self.Figure) {
			lines := self.Field.CheckForLines()
			if lines > 0 {
				self.AddScore(lines * 1000)
			}
			self.Figure = self.NextFigure
			if self.Field.Collide(self.Figure) {
				self.State = GS_GameOver
				return
			}
			self.NextFigure = NewRandomTetrisFigureNot(self.Figure)
		}
	}
}

func (self *GameSession) updateGameOver(delta uint32) {
	self.grayifyingTime += delta
	if self.grayifyingTime > grayifyingInterval {
		self.grayifyingTime -= grayifyingInterval
		self.Field.Grayify()
	}
}

func (self *GameSession) updateGamePaused(delta uint32) {
	self.updateGameOver(delta)
}

func (self *GameSession) Update(delta uint32) {
	switch self.State {
	case GS_Playing:
		self.updatePlaying(delta)
	case GS_GameOver:
		self.updateGameOver(delta)
	case GS_Paused:
		self.updateGamePaused(delta)
	}
}

//-------------------------------------------------------------------------
// GameSession::HandleKey
//-------------------------------------------------------------------------

func (self *GameSession) handleKeyPlaying(key uint32) bool {
	switch key {
	case sdl.K_LEFT, sdl.K_a, sdl.K_j:
		self.Figure.X--
		if self.Field.Collide(self.Figure) {
			self.Figure.X++
		}
	case sdl.K_RIGHT, sdl.K_d, sdl.K_l:
		self.Figure.X++
		if self.Field.Collide(self.Figure) {
			self.Figure.X--
		}
	case sdl.K_UP, sdl.K_w, sdl.K_i:
		self.Figure.Rotate(rotateCWBlock)
		if self.Field.Collide(self.Figure) {
			self.Figure.Rotate(rotateCCWBlock)
		}
	case sdl.K_DOWN, sdl.K_s, sdl.K_k, sdl.K_SPACE:
		for {
			if self.Field.Collide(self.Figure) {
				self.Figure.Y--
				break
			} else {
				self.Figure.Y++
			}
		}
	case sdl.K_ESCAPE:
		return false
	case sdl.K_p:
		self.State = GS_Paused
	}
	return true
}

func (self *GameSession) handleKeyPaused(key uint32) bool {
	if key == sdl.K_p {
		self.State = GS_Playing
	}
	return true
}

func (self *GameSession) handleKeyGameOver(key uint32) bool {
	switch key {
	case sdl.K_y:
		self.Reset()
	case sdl.K_n, sdl.K_ESCAPE:
		return false
	}
	return true
}

func (self *GameSession) HandleKey(key uint32) bool {
	switch self.State {
	case GS_Playing:
		return self.handleKeyPlaying(key)
	case GS_GameOver:
		return self.handleKeyGameOver(key)
	case GS_Paused:
		return self.handleKeyPaused(key)
	}
	return true
}

//-------------------------------------------------------------------------
// GameSession::Draw
//-------------------------------------------------------------------------

func (self *GameSession) Draw() {
	switch self.State {
	case GS_Playing:
		self.drawPlaying()
	case GS_GameOver:
		self.drawGameOver()
	case GS_Paused:
		self.drawGamePaused()
	}
}

func (self *GameSession) drawPlaying() {
	self.Field.Draw(self.cx, self.cy)
	self.Figure.Draw(self.cx, self.cy)

	gl.Color3ub(255, 255, 255)
	self.font.Draw(self.cx + self.Field.PixelsWidth() + 50, self.cy + 5, "Next:")
	self.NextFigure.Draw(self.cx + self.Field.PixelsWidth(), self.cy + 50)
}

func (self *GameSession) drawGameOver() {
	self.drawPlaying()
	gl.Color3ub(200, 0, 0)
	self.font.Draw(self.gameOverCx, 5, "Game Over, restart? y/n")
}

func (self *GameSession) drawGamePaused() {
	self.drawPlaying()
	gl.Color3ub(200, 200, 0)
	self.font.Draw(self.pauseCx, 5, "Game paused, press P to resume")
}

//-------------------------------------------------------------------------
// main()
//-------------------------------------------------------------------------

func main() {
	flag.Parse()
	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()

	sdl.GL_SetAttribute(sdl.GL_SWAP_CONTROL, 1)

	if sdl.SetVideoMode(640, 480, 32, sdl.OPENGL) == nil {
		panic("sdl error")
	}

	sdl.WM_SetCaption("Gotris", "Gotris")
	sdl.EnableKeyRepeat(250, 45)

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

	gl.ClearColor(0, 0, 0, 0)

	//-----------------------------------------------------------------------------

	font, err := LoadFontFromFile("dejavu.font")
	if err != nil {
		panic(err)
	}

	rand.Seed(int64(sdl.GetTicks()))

	gs := NewGameSession(*initLevel, font)
	lastTime := sdl.GetTicks()

	running := true
	for running {
		e := new(sdl.Event)
		for e.Poll() {
			switch e.Type {
			case sdl.QUIT:
				running = false
			case sdl.KEYDOWN:
				running = gs.HandleKey(e.Keyboard().Keysym.Sym)
			}
		}

		now := sdl.GetTicks()
		delta := now - lastTime
		lastTime = now

		gs.Update(delta)

		gl.Clear(gl.COLOR_BUFFER_BIT)
		font.Draw(5, 5, fmt.Sprintf("Level: %d | Score: %d", gs.Level, gs.Score))
		gs.Draw()
		gl.Color3ub(255,255,255)
		sdl.GL_SwapBuffers()
	}
}
