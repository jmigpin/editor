package event

import "testing"

func TestComposeDiacriticLiteralAndDeadKeys(t *testing.T) {
	defer diacriticsData.clear()
	diacriticsData.clear()

	for _, literal := range []rune{'`', '~'} {
		ks, ru := KSymNone, literal
		if ComposeDiacritic(&ks, &ru) || ru != literal {
			t.Fatalf("literal %q started composition", literal)
		}
	}

	ks, ru := KSymAcute, '´'
	if !ComposeDiacritic(&ks, &ru) {
		t.Fatal("dead acute did not start composition")
	}
	ks, ru = KSymO, 'o'
	if ComposeDiacritic(&ks, &ru) || ru != 'ó' {
		t.Fatalf("dead acute + o produced %q", ru)
	}

	for _, next := range []KeySym{KSymAcute, KSymSpace} {
		ks, ru = KSymAcute, '´'
		if !ComposeDiacritic(&ks, &ru) {
			t.Fatal("dead acute did not latch")
		}
		ks, ru = next, ' '
		if ComposeDiacritic(&ks, &ru) || ru != '´' {
			t.Fatalf("dead acute + %v produced %q", next, ru)
		}
	}
}

func TestComposeDiacriticMultiKey(t *testing.T) {
	defer diacriticsData.clear()
	diacriticsData.clear()

	ks, ru := KSymMultiKey, rune(0)
	if !ComposeDiacritic(&ks, &ru) {
		t.Fatal("MultiKey did not start composition")
	}
	ks, ru = KSymNone, '`'
	if !ComposeDiacritic(&ks, &ru) {
		t.Fatal("literal grave after MultiKey did not latch")
	}
	ks, ru = KSymA, 'a'
	if ComposeDiacritic(&ks, &ru) || ru != 'à' {
		t.Fatalf("MultiKey + grave + a produced %q", ru)
	}
}
