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

	// order matters to match map below
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
	// latch key
	for i, ru2 := range diacritics {
		if ru2 == *ru {
			if !diacriticsData.on || *ks != diacriticsData.ks {
				diacriticsData.on = true
				diacriticsData.ks = *ks
				diacriticsData.ru = *ru
				diacriticsData.dindex = i
				return true
			}
		}
	}

	// ensure state is cleared at the end (also accounts for multikey case)
	defer func() {
		switch *ks {
		case KSymShiftL,
			KSymShiftR,
			KSymShiftLock,
			KSymCapsLock,
			KSymControlL,
			KSymControlR,
			KSymAltR,
			KSymAltL,
			KSymAltGr:
			// don't clear yet
		default:
			diacriticsData.clear()
		}
	}()

	// latch key not present from previous stroke
	if !diacriticsData.on {
		return false
	}

	// allow same keysym (or space) to use the diacritic rune
	if *ks == diacriticsData.ks || *ks == KSymSpace {
		*ks = diacriticsData.ks
		*ru = diacriticsData.ru
		return false
	}

	// diacritics order matters
	m := map[rune][]rune{
		// vowels
		'A': {'À', 'Á', 'Â', 'Ã', 'Ä', 'Å', '_', '_', '_', 'Ă', 'Ǎ'},
		'a': {'à', 'á', 'â', 'ã', 'ä', 'å', '_', '_', '_', 'ă', 'ǎ'},
		'E': {'È', 'É', 'Ê', 'Ẽ', 'Ë', '_', '_', '_', '_', 'Ĕ', 'Ě'},
		'e': {'è', 'é', 'ê', 'ẽ', 'ë', '_', '_', '_', '_', 'ĕ', 'ě'},
		'I': {'Ì', 'Í', 'Î', 'Ĩ', 'Ï', '_', '_', '_', '_', 'Ĭ', 'Ǐ'},
		'i': {'ì', 'í', 'î', 'ĩ', 'ï', '_', '_', '_', '_', 'ĭ', 'ǐ'},
		'O': {'Ò', 'Ó', 'Ô', 'Õ', 'Ö', '_', '_', '_', '_', 'Ŏ', 'Ǒ'},
		'o': {'ò', 'ó', 'ô', 'õ', 'ö', '_', '_', '_', '_', 'ŏ', 'ǒ'},
		'U': {'Ù', 'Ú', 'Û', 'Ũ', 'Ü', 'Ů', '_', '_', '_', 'Ŭ', 'Ǔ'},
		'u': {'ù', 'ú', 'û', 'ũ', 'ü', 'ů', '_', '_', '_', 'ŭ', 'ǔ'},

		// other letters
		//'_': {'_', '_', '_', '_', '_', '_', '_', '_', '_', '_'},
		'C': {'_', 'Ć', 'Ĉ', '_', '_', '_', '_', 'Ç', '_', 'Č'},
		'c': {'_', 'ć', 'ĉ', '_', '_', '_', '_', 'ç', '_', 'č'},
		'G': {'_', '_', 'Ĝ', '_', '_', '_', '_', 'Ģ', '_', 'Ǧ'},
		'g': {'_', '_', 'ĝ', '_', '_', '_', '_', 'ģ', '_', 'ǧ'},
		'H': {'_', '_', 'Ĥ', '_', '_', '_', '_', 'Ḩ', '_', 'Ȟ'},
		'h': {'_', '_', 'ĥ', '_', '_', '_', '_', 'ḩ', '_', 'ȟ'},
		'J': {'_', '_', 'Ĵ', '_', '_', '_', '_', '_', '_', '_'},
		'j': {'_', '_', 'ĵ', '_', '_', '_', '_', '_', '_', '_'},
		'K': {'_', '_', '_', '_', '_', '_', '_', 'Ķ', '_', '_'},
		'k': {'_', '_', '_', '_', '_', '_', '_', 'ķ', '_', '_'},
		'L': {'_', 'Ĺ', '_', '_', '_', '_', '_', 'Ļ', '_', '_'},
		'l': {'_', 'ĺ', '_', '_', '_', '_', '_', 'ļ', '_', '_'},
		'N': {'_', 'Ń', '_', 'Ñ', '_', '_', '_', 'Ņ', '_', 'Ň'},
		'n': {'_', 'ń', '_', 'ñ', '_', '_', '_', 'ņ', '_', 'ň'},
		'R': {'_', 'Ŕ', '_', '_', '_', '_', '_', 'Ŗ', '_', '_'},
		'r': {'_', 'ŕ', '_', '_', '_', '_', '_', 'ŗ', '_', '_'},
		'S': {'_', 'Ś', 'Ŝ', '_', '_', '_', '_', 'Ş', '_', '_'},
		's': {'_', 'ś', 'ŝ', '_', '_', '_', '_', 'ş', '_', '_'},
		'T': {'_', '_', '_', '_', '_', '_', '_', 'Ţ', '_', '_'},
		't': {'_', '_', '_', '_', '_', '_', '_', 'ţ', '_', '_'},
		'W': {'_', '_', 'Ŵ', '_', '_', '_', '_', '_', '_', '_'},
		'w': {'_', '_', 'ŵ', '_', '_', '_', '_', '_', '_', '_'},
		'Y': {'_', 'Ý', 'Ŷ', '_', 'Ÿ', '_', '_', '_', '_', '_'},
		'y': {'_', 'ý', 'ŷ', '_', 'ÿ', '_', '_', '_', '_', '_'},
		'Z': {'_', 'Ź', '_', '_', '_', '_', '_', '_', '_', 'Ž'},
		'z': {'_', 'ź', '_', '_', '_', '_', '_', '_', '_', 'ž'},
	}
	if sru, ok := m[*ru]; ok {
		if diacriticsData.dindex < len(sru) {
			ru2 := sru[diacriticsData.dindex]
			if ru2 != '_' {
				*ru = ru2
			}
		}
	}
	return false
}

type diacriticsState struct {
	on       bool
	ks       KeySym
	ru       rune
	dindex   int
	multiKey bool
}

func (ds *diacriticsState) clear() {
	*ds = diacriticsState{}
}
