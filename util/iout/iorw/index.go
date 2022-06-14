package iorw

import (
	"bytes"
	"context"
	"fmt"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// NOTE: considerered golang.org/x/text/search.Matcher but index backwards search is not implemented, as well as some options flexibility

//----------

func Index(r ReaderAt, i int, sep []byte, ignoreCase bool) (int, int, error) {
	ctx := context.Background()
	opt := &IndexOpt{IgnoreCase: ignoreCase}
	return IndexCtx(ctx, r, i, sep, opt)
}

// Returns (-1, 0, nil) if not found.
func IndexCtx(ctx context.Context, r ReaderAt, i int, sep []byte, opt *IndexOpt) (index int, n int, _ error) {
	return indexCtx2(ctx, r, i, sep, -1, opt)
}
func indexCtx2(ctx context.Context, r ReaderAt, i int, sep []byte, chunk int, opt *IndexOpt) (index int, n int, _ error) {

	pfcFn := prepareForCompareFn(opt)
	sep, sepN, err := pfcFn(sep)
	if err != nil {
		return 0, 0, err // TODO: continue?
	}

	chunk, err = setupChunkSize(chunk, sepN, opt)
	if err != nil {
		return 0, 0, err
	}

	max := r.Max()
	for k := i; k < max; k += chunk - (sepN - 1) {
		c := chunk
		if c > max-k {
			c = max - k
		}

		j, n, err := indexCtx3(r, k, c, sep, pfcFn, opt)
		if err != nil || j >= 0 {
			return j, n, err
		}

		// check context cancelation
		if err := ctx.Err(); err != nil {
			return -1, 0, err
		}
	}

	return -1, 0, nil
}
func indexCtx3(r ReaderAt, i, n int, sep []byte, pfcFn pfcType, opt *IndexOpt) (int, int, error) {
	return indexCtx4(bytes.Index, r, i, n, sep, pfcFn, opt)
}
func indexCtx4(indexFn func(s, sep []byte) int, r ReaderAt, i, n int, sep []byte, pfcFn pfcType, opt *IndexOpt) (int, int, error) {
	p, err := r.ReadFastAt(i, n)
	if err != nil {
		return 0, 0, err
	}
	p2, _, err := pfcFn(p) // prepare for compare
	if err != nil {
		return 0, 0, err // TODO: continue?
	}
	j := indexFn(p2, sep) // can be used by index/lastindex
	if j >= 0 {
		n := len(sep)
		if opt.IgnoringDiacritics() {
			j, n = correctRunesPos(p, p2, sep, j)
		}
		return i + j, n, nil
	}
	return -1, 0, nil
}

//----------

func LastIndex(r ReaderAt, i int, sep []byte, ignoreCase bool) (int, int, error) {
	ctx := context.Background()
	opt := &IndexOpt{IgnoreCase: ignoreCase}
	return LastIndexCtx(ctx, r, i, sep, opt)
}

// Returns (-1, 0, nil) if not found.
func LastIndexCtx(ctx context.Context, r ReaderAt, i int, sep []byte, opt *IndexOpt) (int, int, error) {
	return lastIndexCtx2(ctx, r, i, sep, -1, opt)
}
func lastIndexCtx2(ctx context.Context, r ReaderAt, i int, sep []byte, chunk int, opt *IndexOpt) (index int, n int, _ error) {

	pfcFn := prepareForCompareFn(opt)
	sep, sepN, err := pfcFn(sep)
	if err != nil {
		return 0, 0, err // TODO: continue?
	}

	chunk, err = setupChunkSize(chunk, len(sep), opt)
	if err != nil {
		return 0, 0, err
	}

	min := r.Min()
	for k := i; k > min; k -= chunk - (sepN - 1) {
		c := chunk
		if c > k-min {
			c = k - min
		}

		j, n, err := lastIndexCtx3(r, k-c, k, sep, pfcFn, opt)
		if err != nil || j >= 0 {
			return j, n, err
		}

		// check context cancelation
		if err := ctx.Err(); err != nil {
			return -1, 0, err
		}
	}

	return -1, 0, nil
}
func lastIndexCtx3(r ReaderAt, i, n int, sep []byte, pfcFn pfcType, opt *IndexOpt) (int, int, error) {
	return indexCtx4(bytes.LastIndex, r, i, n, sep, pfcFn, opt)
}

//----------
//----------
//----------

type IndexOpt struct {
	IgnoreCase           bool
	IgnoreCaseDiacritics bool // also lower the case of diacritics (slow)
	IgnoreDiacritics     bool
}

func (opt *IndexOpt) IgnoringDiacritics() bool {
	return opt.IgnoreCaseDiacritics || opt.IgnoreDiacritics
}

//----------
//----------
//----------

type pfcType func([]byte) (result []byte, nSrcBytesRead int, _ error)

func prepareForCompareFn(opt *IndexOpt) pfcType {
	w := []transform.Transformer{}
	if opt.IgnoreCase {
		tla := &toLowerAscii{lowerDiacritics: opt.IgnoreCaseDiacritics}
		w = append(w, tla)
	}
	if opt.IgnoreDiacritics {
		// https://go.dev/blog/normalization
		w = append(w,
			norm.NFD, // decompose
			runes.Remove(runes.In(unicode.Mn)),
			norm.NFC, // compose
		)
	}
	t := transform.Chain(w...) // ok if w is empty
	return func(b []byte) ([]byte, int, error) {
		return transform.Bytes(t, b)
	}
}

//----------

type toLowerAscii struct {
	lowerDiacritics bool
}

// implement transform.Transformer
func (tla *toLowerAscii) Reset() {}
func (tla *toLowerAscii) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {

	if tla.lowerDiacritics {
		// ~8x slower
		// 'áb' will match 'ÁB' but not 'ab'
		b := bytes.ToLower(src)
		n := copy(dst, b)
		return n, len(b), nil
	}

	min := len(src)
	if min > len(dst) {
		min = len(dst)
	}
	for i := 0; i < min; i++ {
		c := src[i]
		if 'A' <= c && c <= 'Z' {
			dst[i] = c + ('a' - 'A')
		} else {
			dst[i] = c
		}
	}
	return min, min, nil
}

//----------
//----------
//----------

const chunkSize = 32 * 1024

func setupChunkSize(chunkN, sepN int, opt *IndexOpt) (int, error) {
	cN := chunkN
	autoChunk := cN <= 0
	if autoChunk {
		cN = chunkSize
	}
	if opt.IgnoringDiacritics() {
		// because the src contains diacritics, need a big enough chunk size to search a src equal to the separator but full of diacritics. Here we give N extra bytes for each sep byte.
		sepN *= 4
	}
	if cN < sepN {
		if !autoChunk {
			return 0, fmt.Errorf("chunk smaller then sepN: %v, %v", chunkN, sepN)
		}
		cN = sepN
	}
	return cN, nil
}

//----------

func correctRunesPos(src, norm, sep []byte, j int) (int, int) {
	// correct j
	runes1 := []rune(string(norm[:j])) // n runes before j
	runes2 := []rune(string(src))      // runes from original p
	// n bytes before j from original p
	if len(runes1) <= len(runes2) {
		j = len(string(runes2[:len(runes1)]))
	}

	n := len(sep)
	// correct n
	runes3 := []rune(string(sep))  // n runes in sep
	runes4 := runes2[len(runes1):] // runes from original p after j
	// n bytes in original p
	if len(runes3) <= len(runes4) {
		n = len(string(runes4[:len(runes3)]))
	}
	return j, n
}
