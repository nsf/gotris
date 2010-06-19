#!/usr/bin/python
# -*- coding: utf-8 -*-

# compiled font is a binary blob:
# 1. magic (MFNT) - 4 bytes
# 2. number of symbols - 4 bytes
# 3. font y advance - 4 bytes
# 4. an array of glyphs (offset_x, offset_y, width, height, tx, ty, tx2, ty2, x_advance) - 36 * number of symbols
#    (iiIIffffI)
# 5. png texture

import sys
import struct
import os
from xml2obj import xml2obj

def print_usage_and_exit():
	print "usage: {0} <UNPACKED FONT>".format(sys.argv[0])
	sys.exit(1)

if len(sys.argv) != 2:
	print_usage_and_exit()

fontfile = sys.argv[1]
if not os.path.exists(fontfile):
	print_usage_and_exit()

glyphs = []

with file(fontfile + ".fontdef.xml", 'r') as f:
	xmlobj = xml2obj(f.read())

font_y_advance = int(xmlobj.height)

for g in xmlobj.glyph:
	glyphs.append((unicode(g.symbol), int(g.offset_x), int(g.offset_y), int(g.width), int(g.height), float(g.tx), float(g.ty), float(g.tx2), float(g.ty2), int(g.x_advance)))

with file(fontfile[:-4] + ".font", 'w') as f:
	f.write("MFNT")
	f.write(struct.pack("<I", len(glyphs)))
	f.write(struct.pack("<I", font_y_advance))
	for g in glyphs:
		f.write(struct.pack("<iiIIffffI", g[1], g[2], g[3], g[4], g[5], g[6], g[7], g[8], g[9]))

	unicode_fontcp = []

	for i, g in enumerate(glyphs):
		unicode_fontcp.append((g[0], i+1))

	def unicode_fontcp_key(item):
		return item[0]
	unicode_fontcp.sort(key=unicode_fontcp_key)

	for entry in unicode_fontcp:
		f.write(struct.pack("<II", ord(entry[0]), entry[1]))

	with file(fontfile, 'r') as imgf:
		imgdata = imgf.read()
	f.write(imgdata)
