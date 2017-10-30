package ui

import (
	"io/ioutil"
	"log"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/drawutil2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// This is temporary until a theme structure is in place.

var FontOpt truetype.Options // Contains: Size, DPI, Hinting
var FontFace font.Face
var NamedFontName string

type FontTheme struct {
	F func() error
}

var fontThemes = []*FontTheme{
	&FontTheme{regularFont},
	&FontTheme{mediumFont},
	&FontTheme{monoFont},
}

var curFontTheme *FontTheme

// Loads first font from font themes
func DefaultFont() {
	CycleFontTheme()
}

func CycleFontTheme() {
	index := -1
	if curFontTheme != nil {
		for i, e := range fontThemes {
			if e == curFontTheme {
				index = i
				break
			}
		}
	}
	// n-1 attempts to load a good font
	for i := 0; i < len(fontThemes)-1; i++ {
		index = (index + 1) % len(fontThemes)
		e := fontThemes[index]
		err := e.F()
		if err != nil {
			log.Print(err)
			continue
		}
		curFontTheme = e
		break
	}
}

func regularFont() error {
	return loadFont(goregular.TTF)
}
func mediumFont() error {
	return loadFont(gomedium.TTF)
}
func monoFont() error {
	return loadFont(gomono.TTF)
}
func namedFont() error {
	b, err := ioutil.ReadFile(NamedFontName)
	if err != nil {
		return err
	}
	return loadFont(b)
}
func loadFont(ttf []byte) error {
	f, err := truetype.Parse(ttf)
	if err != nil {
		return err
	}
	if FontFace != nil {
		FontFace.Close()
	}
	FontFace = drawutil2.NewFace(f, &FontOpt)
	return nil
}

func SetNamedFont(name string) error {
	NamedFontName = name
	err := namedFont()
	if err != nil {
		return err
	}
	curFontTheme = &FontTheme{namedFont}
	// include function for cycle-font-theme
	fontThemes = append(fontThemes, curFontTheme)
	return nil
}
