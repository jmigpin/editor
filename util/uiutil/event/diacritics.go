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
	if dindex >= 0 && *ks != diacriticsData.ks {
		diacriticsData.ks = *ks
		diacriticsData.ru = *ru
		diacriticsData.dindex = dindex
		return true
	}

	// latch key is present from previous stroke
	if diacriticsData.ks != 0 {

		// allow same keysym (or space) to use the diacritic rune
		if *ks == diacriticsData.ks || *ks == KSymSpace {
			*ks = diacriticsData.ks
			*ru = diacriticsData.ru
			diacriticsData.clear()
			return false
		}

		// diacritis order matters
		m := map[rune][]rune{
			// vowels
			'A': {'À', 'Á', 'Â', 'Ã', 'Ä', 'Å'},
			'a': {'à', 'á', 'â', 'ã', 'ä', 'å'},
			'E': {'È', 'É', 'Ê', 'Ẽ', 'Ë', '_'},
			'e': {'è', 'é', 'ê', 'ẽ', 'ë', '_'},
			'I': {'Ì', 'Í', 'Î', 'Ĩ', 'Ï', '_'},
			'i': {'ì', 'í', 'î', 'ĩ', 'ï', '_'},
			'O': {'Ò', 'Ó', 'Ô', 'Õ', 'Ö', '_'},
			'o': {'ò', 'ó', 'ô', 'õ', 'ö', '_'},
			'U': {'Ù', 'Ú', 'Û', 'Ũ', 'Ü', 'Ů'},
			'u': {'ù', 'ú', 'û', 'ũ', 'ü', 'ů'},

			// other letters
			'C': {'_', 'Ć', 'Ĉ', '_', '_', '_'},
			'c': {'_', 'ć', 'ĉ', '_', '_', '_'},
			'G': {'_', '_', 'Ĝ', '_', '_', '_'},
			'g': {'_', '_', 'ĝ', '_', '_', '_'},
			'H': {'_', '_', 'Ĥ', '_', '_', '_'},
			'h': {'_', '_', 'ĥ', '_', '_', '_'},
			'J': {'_', '_', 'Ĵ', '_', '_', '_'},
			'j': {'_', '_', 'ĵ', '_', '_', '_'},
			'L': {'_', 'Ĺ', '_', '_', '_', '_'},
			'l': {'_', 'ĺ', '_', '_', '_', '_'},
			'N': {'_', 'Ń', '_', 'Ñ', '_', '_'},
			'n': {'_', 'ń', '_', 'ñ', '_', '_'},
			'R': {'_', 'Ŕ', '_', '_', '_', '_'},
			'r': {'_', 'ŕ', '_', '_', '_', '_'},
			'S': {'_', 'Ś', 'Ŝ', '_', '_', '_'},
			's': {'_', 'ś', 'ŝ', '_', '_', '_'},
			'W': {'_', '_', 'Ŵ', '_', '_', '_'},
			'w': {'_', '_', 'ŵ', '_', '_', '_'},
			'Y': {'_', 'Ý', 'Ŷ', '_', 'Ÿ', '_'},
			'y': {'_', 'ý', 'ŷ', '_', 'ÿ', '_'},
			'Z': {'_', 'Ź', '_', '_', '_', '_'},
			'z': {'_', 'ź', '_', '_', '_', '_'},
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
