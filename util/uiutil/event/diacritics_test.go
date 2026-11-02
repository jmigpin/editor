package event

import "testing"

func TestComposeDiacriticLiteralAndDeadKeys(t *testing.T) {
	oldUseMultiKey := UseMultiKey
	defer func() {
		UseMultiKey = oldUseMultiKey
		diacriticsData.clear()
	}()

	for _, useMultiKey := range []bool{false, true} {
		UseMultiKey = useMultiKey
		diacriticsData.clear()

		for _, literal := range []rune{'`', '~'} {
			ks, ru := KSymNone, literal
			if ComposeDiacritic(&ks, &ru) || ru != literal {
				t.Fatalf("useMultiKey=%v: literal %q started composition", useMultiKey, literal)
			}
		}

		ks, ru := KSymAcute, '´'
		if !ComposeDiacritic(&ks, &ru) {
			t.Fatalf("useMultiKey=%v: dead acute did not start composition", useMultiKey)
		}
		ks, ru = KSymO, 'o'
		if ComposeDiacritic(&ks, &ru) || ru != 'ó' {
			t.Fatalf("useMultiKey=%v: dead acute + o produced %q", useMultiKey, ru)
		}

		for _, next := range []KeySym{KSymAcute, KSymSpace} {
			ks, ru = KSymAcute, '´'
			if !ComposeDiacritic(&ks, &ru) {
				t.Fatalf("useMultiKey=%v: dead acute did not latch", useMultiKey)
			}
			ks, ru = next, ' '
			if ComposeDiacritic(&ks, &ru) || ru != '´' {
				t.Fatalf("useMultiKey=%v: dead acute + %v produced %q", useMultiKey, next, ru)
			}
		}
	}
}

func TestComposeDiacriticMultiKey(t *testing.T) {
	oldUseMultiKey := UseMultiKey
	UseMultiKey = true
	defer func() {
		UseMultiKey = oldUseMultiKey
		diacriticsData.clear()
	}()
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
