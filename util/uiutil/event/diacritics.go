package event

var UseMultiKey = false

var diacriticsData diacriticsState

func ComposeDiacritic(ks *KeySym, ru *rune) (isLatch bool) {
	if UseMultiKey {
		if *ks == KSymMultiKey {
			diacriticsData.multiKey = true
			return true
		} else if !diacriticsData.multiKey {
			return false
		}
	}

	// order matters
	diacritics := []rune{
		'`', // grave
		'´', // acute
		'^', // circumflex
		'~', // tilde
		'¨', // diaeresis 0xa8
		'˚', // ring above 0x2da

		'¯', // macron 0xaf
		'¸', // cedilla 0xb8
		'˘', // breve 0x2d8
		'ˇ', // caron 0x2c7
	}

	dindex := -1
	for i, aru := range diacritics {
		if aru == *ru {
			dindex = i
			break
		}
	}

	// latch key
	if dindex >= 0 {
		diacriticsData.ks = *ks
		diacriticsData.ru = *ru
		diacriticsData.dindex = dindex
		return true
	}

	// latch key is present from previous stroke
	if diacriticsData.ks != 0 {

		// allow space to use the diacritic rune
		if *ks == KSymSpace {
			*ks = diacriticsData.ks
			*ru = diacriticsData.ru
			diacriticsData.clear()
			return false
		}

		// diacritis order matters
		m := map[rune][]rune{
			// vowels
			'A': []rune{'À', 'Á', 'Â', 'Ã', 'Ä', 'Å'},
			'a': []rune{'à', 'á', 'â', 'ã', 'ä', 'å'},
			'E': []rune{'È', 'É', 'Ê', 'Ẽ', 'Ë', '_'},
			'e': []rune{'è', 'é', 'ê', 'ẽ', 'ë', '_'},
			'I': []rune{'Ì', 'Í', 'Î', 'Ĩ', 'Ï', '_'},
			'i': []rune{'ì', 'í', 'î', 'ĩ', 'ï', '_'},
			'O': []rune{'Ò', 'Ó', 'Ô', 'Õ', 'Ö', '_'},
			'o': []rune{'ò', 'ó', 'ô', 'õ', 'ö', '_'},
			'U': []rune{'Ù', 'Ú', 'Û', 'Ũ', 'Ü', 'Ů'},
			'u': []rune{'ù', 'ú', 'û', 'ũ', 'ü', 'ů'},

			// other letters
			'C': []rune{'_', 'Ć', 'Ĉ', '_', '_', '_'},
			'c': []rune{'_', 'ć', 'ĉ', '_', '_', '_'},
			'G': []rune{'_', '_', 'Ĝ', '_', '_', '_'},
			'g': []rune{'_', '_', 'ĝ', '_', '_', '_'},
			'H': []rune{'_', '_', 'Ĥ', '_', '_', '_'},
			'h': []rune{'_', '_', 'ĥ', '_', '_', '_'},
			'J': []rune{'_', '_', 'Ĵ', '_', '_', '_'},
			'j': []rune{'_', '_', 'ĵ', '_', '_', '_'},
			'L': []rune{'_', 'Ĺ', '_', '_', '_', '_'},
			'l': []rune{'_', 'ĺ', '_', '_', '_', '_'},
			'N': []rune{'_', 'Ń', '_', 'Ñ', '_', '_'},
			'n': []rune{'_', 'ń', '_', 'ñ', '_', '_'},
			'R': []rune{'_', 'Ŕ', '_', '_', '_', '_'},
			'r': []rune{'_', 'ŕ', '_', '_', '_', '_'},
			'S': []rune{'_', 'Ś', 'Ŝ', '_', '_', '_'},
			's': []rune{'_', 'ś', 'ŝ', '_', '_', '_'},
			'W': []rune{'_', '_', 'Ŵ', '_', '_', '_'},
			'w': []rune{'_', '_', 'ŵ', '_', '_', '_'},
			'Y': []rune{'_', 'Ý', 'Ŷ', '_', 'Ÿ', '_'},
			'y': []rune{'_', 'ý', 'ŷ', '_', 'ÿ', '_'},
			'Z': []rune{'_', 'Ź', '_', '_', '_', '_'},
			'z': []rune{'_', 'ź', '_', '_', '_', '_'},
		}
		if sru, ok := m[*ru]; ok {
			if diacriticsData.dindex < len(sru) {
				ru2 := sru[diacriticsData.dindex]
				if ru2 != '_' {
					*ru = ru2
				}
				diacriticsData.clear()
			}
		}
	}

	return false
}

type diacriticsState struct {
	ks       KeySym
	ru       rune
	dindex   int
	multiKey bool
}

func (ds *diacriticsState) clear() {
	*ds = diacriticsState{}
}
