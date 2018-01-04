package ui

import (
	"io/ioutil"
	"log"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/drawutil"
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
	&FontTheme{RegularFont},
	&FontTheme{MediumFont},
	&FontTheme{MonoFont},
}

var curFontTheme *FontTheme

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

func RegularFont() error {
	return loadFont(goregular.TTF)
}
func MediumFont() error {
	return loadFont(gomedium.TTF)
}
func MonoFont() error {
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
	FontFace = drawutil.NewFace(f, &FontOpt)
	return nil
}

func SetNamedFont(name string) error {
	NamedFontName = name
	err := namedFont()
	if err != nil {
		return err
	}
	// include function for cycle-font-theme
	fontThemes = append(fontThemes, &FontTheme{namedFont})
	return nil
}
