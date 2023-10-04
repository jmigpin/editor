package widget

// offset, start, deletedn, insertedn
func StableOffsetScroll(o int, s, dn, in int) int {
	// TODO: need to know if the insertion/delete is on the same line to try to keep the line percentage (smooth scrolling)

	ed := s + dn // end of deletes
	ei := s + in // end of inserts

	if o <= s { // o<s<={ed,ei}
		return o
	}
	if o <= ed { // s<o<=ed
		if o <= ei { // s<o<={ed,ei}
			return o // inserts cover the deletes
		} else { // s<ei<o<=ed
			// add missing to cover the deletes
			return o - (o - ei)
		}
	}
	if o <= ei { // s<ed<o<=ei
		// inserts cover the deletes
		return o
	}
	// s<{ed,ei}<o
	o += in - dn // add missing bytes to reach old offset
	return o
}
