#!/usr/bin/python
# -*- coding: utf-8 -*-

# actions order:
# 1. collect glyphs info for a parameter set
# 2. (optional) modify glyphs info TODO
# 3. calculate texture size
# 4. draw glyphs, collect drawn glyph info
# 5. save output

import cairo
import math
import optparse
from collections import namedtuple
from xml.sax.saxutils import escape, quoteattr

#-------------------------------------------------------------------------------
# helping data structures
#-------------------------------------------------------------------------------

default_symbols = """ `1234567890-=\~!@#$%^&*()_+|qwertyuiop[]QWERTYUIOP{}asdfghjkl;'ASDFGHJKL:"zxcvbnm,./ZXCVBNM<>?"""

default_parameters = {
	"font_name" : "DejaVu",
	"size" : 8,
	"filename" : "outfont.png",
	"slant" : "normal",
	"weight" : "normal",
	"symbols" : default_symbols,
	"antialias" : cairo.ANTIALIAS_DEFAULT,
	"hint_style" : "default",
	"subpixel_order" : cairo.SUBPIXEL_ORDER_DEFAULT
}

slant_map = {
	"normal" : cairo.FONT_SLANT_NORMAL,
	"italic" : cairo.FONT_SLANT_ITALIC,
	"oblique" : cairo.FONT_SLANT_OBLIQUE
}

weight_map = {
	"normal" : cairo.FONT_WEIGHT_NORMAL,
	"bold" : cairo.FONT_WEIGHT_BOLD
}

hint_style_map = {
	"default" : cairo.HINT_STYLE_DEFAULT,
	"none" : cairo.HINT_STYLE_NONE,
	"slight" : cairo.HINT_STYLE_SLIGHT,
	"medium" : cairo.HINT_STYLE_MEDIUM,
	"full" : cairo.HINT_STYLE_FULL,
}

#-------------------------------------------------------------------------------
# utility functions
#-------------------------------------------------------------------------------

def next_power_of_2(v):
	v -= 1
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	return v + 1

def convert_xywh_to_texcoords(x, y, w, h, imagew, imageh):
	xs = 1.0 / imagew
	ys = 1.0 / imageh
	tx = x  * xs
	tx2 = (x + w) * xs
	ty = (y * ys)
	ty2 = (y + h) * ys
	return (tx, ty, tx2, ty2)


def create_cairo_for_parameters(parameters, surface):
	cr = cairo.Context(surface)
	cr.select_font_face(parameters["font_name"], parameters["slant"], parameters["weight"])
	cr.set_font_size(parameters["size"] * 1.333)
	fontopts = cr.get_font_options()
	fontopts.set_hint_style(parameters["hint_style"])
	fontopts.set_antialias(parameters["antialias"])
	fontopts.set_subpixel_order(parameters["subpixel_order"])
	cr.set_font_options(fontopts)
	return cr

def read_symbols_from_file(filename):
	with file(filename, "r") as f:
		text = f.read()
	return text.rstrip("\n\r").decode("UTF-8")

#-------------------------------------------------------------------------------
# tuple data types
#-------------------------------------------------------------------------------
Glyph = namedtuple('Glyph', 'symbol x_bearing y_bearing width height x_advance y_advance')
DrawnGlyph = namedtuple('DrawnGlyph', 'symbol x_bearing y_bearing width height x_advance y_advance x y')
FontInfo = namedtuple('FontInfo', 'ascent descent height max_x_advance max_y_advance')

def collect_glyphs(parameters):
	fake_surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, 32, 32)
	cr = create_cairo_for_parameters(parameters, fake_surface)

	glyphs = []
	for symbol in parameters["symbols"]:
		symbol_extents = cr.text_extents(symbol)
		glyphs.append(Glyph(*((symbol,) + symbol_extents)))

	return glyphs

def xform_glyphs(glyphs):
	# sort glyphs
	def glyph_height(g):
		return -g.height
	glyphs.sort(key=glyph_height)

def check_glyphs_fit(glyphs, w, h):
	line_height = 0
	x = 0
	y = 0
	for g in glyphs:
		if x + g.width > w:
			x = 0
			y += line_height + 1
			line_height = 0
		line_height = max(line_height, g.height)
		if y + line_height > h:
			return False
		x += g.width + 1
	return True

def make_new_wh(w, h, i):
	if i % 2:
		return w / 2, h
	else:
		return w, h / 2

def calculate_font_texture_dimensions(glyphs):
	total_area = 0
	for g in glyphs:
		total_area += g.width * g.height

	size = next_power_of_2(int(math.sqrt(total_area)))
	w, h = size, size
	exact_fit = False

	# if font doesn't fit, try multiplying texture area up to 4 times
	if not check_glyphs_fit(glyphs, w, h):
		exact_fit = True
		w *= 2
		if not check_glyphs_fit(glyphs, w, h):
			h *= 2

	# if font fitted before growing, we need to try to shrink it to the exact fit
	if not exact_fit:
		fits = True
		i = 0
		while True:
			neww, newh = make_new_wh(w, h, i)
			fits = check_glyphs_fit(glyphs, neww, newh)
			if fits:
				w, h = neww, newh
			else:
				break
			i += 1

	return w, h

def draw_glyph(cr, glyph, x, y):
	cr.move_to(x - glyph.x_bearing, y - glyph.y_bearing)
	cr.show_text(glyph.symbol)
	
def write_font_info_file(filename, drawnglyphs, fontinfo, iw, ih):
	with file(filename, "w") as f:
		f.write('<fontdef height="{0}">\n'.format(int(fontinfo.height)))
		for g in drawnglyphs:
			symbol = g.symbol.encode("UTF-8")
			offset_x = int(round(g.x_bearing))
			offset_y = int(round(fontinfo.ascent + g.y_bearing))
			width = int(round(g.width))
			height = int(round(g.height))
			x_advance = int(round(g.x_advance))
			(tx, ty, tx2, ty2) = convert_xywh_to_texcoords(g.x, g.y, width, height, iw, ih)
			f.write('\t<glyph symbol={0} offset_x="{1}" offset_y="{2}" width="{3}" height="{4}" tx="{5}" ty="{6}" tx2="{7}" ty2="{8}" x_advance="{9}"/>\n'
					.format(quoteattr(symbol), offset_x, offset_y, width, height, tx, ty, tx2, ty2, x_advance))
		f.write('</fontdef>\n')

def draw_glyphs(parameters):
	glyphs = collect_glyphs(parameters)
	xform_glyphs(glyphs)
	w, h = calculate_font_texture_dimensions(glyphs)
	surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, w, h)
	cr = create_cairo_for_parameters(parameters, surface)
	fontinfo = FontInfo(*cr.font_extents())

	drawnglyphs = []

	cr.set_operator(cairo.OPERATOR_SOURCE)
	cr.set_source_rgba(1, 1, 1, 1)
	line_height = 0
	x = 0
	y = 0
	for g in glyphs:
		if x + g.width > w:
			x = 0
			y += line_height + 1
			line_height = 0
		line_height = max(line_height, g.height)
		draw_glyph(cr, g, x, y)
		drawnglyphs.append(DrawnGlyph(*(g + (int(x), int(y)))))
		x += g.width + 1

	surface.write_to_png(p["filename"])
	write_font_info_file(p["filename"] + ".fontdef.xml", drawnglyphs, fontinfo, w, h)

#-------------------------------------------------------------------------------

parser = optparse.OptionParser()
parser.add_option("-o", "--output", dest="filename", default=default_parameters['filename'],
                  help="write resulting bitmap font to FILE and FILE.fontdef.xml", metavar="FILE")

parser.add_option("--font", dest="font_name", default=default_parameters['font_name'],
                  help="font face name", metavar="FACE")

parser.add_option("--slant", dest="slant", default=default_parameters['slant'],
                  help="font slant (normal, italic, oblique)", type="choice",
                  choices=("normal", "italic", "oblique"))

parser.add_option("--weight", dest="weight", default=default_parameters['weight'],
                  help="font weight (normal, bold)", type="choice", choices=("normal", "bold"))

parser.add_option("--size", dest="size", default=default_parameters['size'],
                  help="size of the font in pixels", type="int")

parser.add_option("--hint-style", dest="hint_style", default=default_parameters['hint_style'],
                  help="hint style (default, none, slight, medium, full)", type="choice",
                  choices=("default", "none", "slight", "medium", "full"))

parser.add_option("--symbols", dest="symbols_file", default=None,
                  help="file containing symbols (utf-8 encoded)", metavar="FILE")

(options, args) = parser.parse_args()

p = default_parameters
p.update(options.__dict__)
if p["symbols_file"]:
	p["symbols"] = read_symbols_from_file(p["symbols_file"])
p["slant"] = slant_map[p["slant"]]
p["weight"] = weight_map[p["weight"]]
p["hint_style"] = hint_style_map[p["hint_style"]]

draw_glyphs(p)
