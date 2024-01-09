// Copyright 2020 Consensys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iop

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/fft"

	"github.com/stretchr/testify/require"

	"bytes"
	"reflect"
)

func TestEvaluation(t *testing.T) {

	size := 8
	shift := 2
	d := fft.NewDomain(uint64(size))
	c := randomVector(size)
	p := NewPolynomial(c, Form{Basis: Canonical, Layout: Regular})
	ps := p.ShallowClone().Shift(shift)
	ref := p.Clone()
	ref.ToLagrange(d).ToRegular()

	// regular layout
	a := p.Evaluate(d.Generator)
	b := ps.Evaluate(d.Generator)
	if !a.Equal(&ref.Coefficients()[1]) {
		t.Fatal("error evaluation")
	}
	if !b.Equal(&ref.Coefficients()[1+shift]) {
		t.Fatal("error evaluation shifted")
	}

	// bit reversed layout
	p.ToBitReverse()
	ps.ToBitReverse()
	a = p.Evaluate(d.Generator)
	b = ps.Evaluate(d.Generator)
	if !a.Equal(&ref.Coefficients()[1]) {
		t.Fatal("error evaluation")
	}
	if !b.Equal(&ref.Coefficients()[1+shift]) {
		t.Fatal("error evaluation shifted")
	}

	// lagrange regular
	var x fr.Element
	x.SetRandom()
	expectedEval := p.Evaluate(x)
	expectedEvalShifted := ps.Evaluate(x)
	p.ToLagrange(d)
	ps.ToLagrange(d)
	plx := p.Evaluate(x)
	pslx := ps.Evaluate(x)
	if !plx.Equal(&expectedEval) {
		t.Fatal("error evaluation lagrange")
	}
	if !pslx.Equal(&expectedEvalShifted) {
		t.Fatal("error evaluation lagrange shifted")
	}

	// lagrange bit reverse
	p.ToBitReverse()
	ps.ToBitReverse()
	plx = p.Evaluate(x)
	pslx = ps.Evaluate(x)
	if !plx.Equal(&expectedEval) {
		t.Fatal("error evaluation lagrange")
	}
	if !pslx.Equal(&expectedEvalShifted) {
		t.Fatal("error evaluation lagrange shifted")
	}

}

func randomVector(size int) *[]fr.Element {

	r := make([]fr.Element, size)
	for i := 0; i < size; i++ {
		r[i].SetRandom()
	}
	return &r
}

func TestGetCoeff(t *testing.T) {

	size := 8
	v := make([]fr.Element, size)
	for i := 0; i < size; i++ {
		v[i].SetUint64(uint64(i))
	}
	wp := NewPolynomial(&v, Form{Layout: Regular, Basis: Canonical})
	wsp := wp.ShallowClone().Shift(1)

	var aa, bb fr.Element

	// regular layout
	for i := 0; i < size; i++ {

		a := wp.GetCoeff(i)
		b := wsp.GetCoeff(i)
		aa.SetUint64(uint64(i))
		bb.SetUint64(uint64((i + 1) % size))
		if !a.Equal(&aa) {
			t.Fatal("error GetCoeff")
		}
		if !b.Equal(&bb) {
			t.Fatal("error GetCoeff")
		}
	}

	// bit reverse + bitReverse and shifted
	wp.ToBitReverse()
	wsp.ToBitReverse()
	for i := 0; i < size; i++ {

		a := wp.GetCoeff(i)
		b := wsp.GetCoeff(i)
		aa.SetUint64(uint64(i))
		bb.SetUint64(uint64((i + 1) % size))
		if !a.Equal(&aa) {
			t.Fatal("error GetCoeff")
		}
		if !b.Equal(&bb) {
			t.Fatal("error GetCoeff")
		}
	}

}

func TestRoundTrip(t *testing.T) {
	assert := require.New(t)
	var buf bytes.Buffer

	size := 8
	d := fft.NewDomain(uint64(8))

	p := NewPolynomial(randomVector(size), Form{Basis: Canonical, Layout: Regular})
	p.ToLagrangeCoset(d)

	// serialize
	written, err := p.WriteTo(&buf)
	assert.NoError(err)

	// deserialize
	var reconstructed Polynomial
	read, err := reconstructed.ReadFrom(&buf)
	assert.NoError(err)

	assert.Equal(read, written, "number of bytes written != number of bytes read")

	// compare
	assert.Equal(p.Basis, reconstructed.Basis)
	assert.Equal(p.Layout, reconstructed.Layout)
	assert.Equal(p.shift, reconstructed.shift)
	assert.Equal(p.size, reconstructed.size)
	c1, c2 := p.Coefficients(), reconstructed.Coefficients()
	assert.True(reflect.DeepEqual(c1, c2))
}

// list of functions to turn a polynomial in Lagrange-regular form
// to all different forms in ordered using this encoding:
// int(p.Basis)*4 + int(p.Layout)*2 + int(p.Status)
// p is in Lagrange/Regular here. This function is for testing purpose
// only.
type TransfoTest func(p polynomial, d *fft.Domain) polynomial

// CANONICAL REGULAR
func fromLagrange0(p *Polynomial, d *fft.Domain) *Polynomial {
	r := p.Clone()
	r.Basis = Canonical
	r.Layout = Regular
	d.FFTInverse(r.Coefficients(), fft.DIF)
	fft.BitReverse(r.Coefficients())
	return r
}

// CANONICAL BITREVERSE
func fromLagrange1(p *Polynomial, d *fft.Domain) *Polynomial {
	r := p.Clone()
	r.Basis = Canonical
	r.Layout = BitReverse
	d.FFTInverse(r.Coefficients(), fft.DIF)
	return r
}

// LAGRANGE REGULAR
func fromLagrange2(p *Polynomial, d *fft.Domain) *Polynomial {
	r := p.Clone()
	r.Basis = Lagrange
	r.Layout = Regular
	return r
}

// LAGRANGE BITREVERSE
func fromLagrange3(p *Polynomial, d *fft.Domain) *Polynomial {
	r := p.Clone()
	r.Basis = Lagrange
	r.Layout = BitReverse
	fft.BitReverse(r.Coefficients())
	return r
}

// LAGRANGE_COSET REGULAR
func fromLagrange4(p *Polynomial, d *fft.Domain) *Polynomial {
	r := p.Clone()
	r.Basis = LagrangeCoset
	r.Layout = Regular
	d.FFTInverse(r.Coefficients(), fft.DIF)
	d.FFT(r.Coefficients(), fft.DIT, fft.OnCoset())
	return r
}

// LAGRANGE_COSET BITREVERSE
func fromLagrange5(p *Polynomial, d *fft.Domain) *Polynomial {
	r := p.Clone()
	r.Basis = LagrangeCoset
	r.Layout = BitReverse
	d.FFTInverse(r.Coefficients(), fft.DIF)
	d.FFT(r.Coefficients(), fft.DIT, fft.OnCoset())
	fft.BitReverse(r.Coefficients())
	return r
}

func fromLagrange(p *Polynomial, d *fft.Domain) *Polynomial {
	id := p.Form
	switch id {
	case canonicalRegular:
		return fromLagrange0(p, d)
	case canonicalBitReverse:
		return fromLagrange1(p, d)
	case lagrangeRegular:
		return fromLagrange2(p, d)
	case lagrangeBitReverse:
		return fromLagrange3(p, d)
	case lagrangeCosetRegular:
		return fromLagrange4(p, d)
	case lagrangeCosetBitReverse:
		return fromLagrange5(p, d)
	default:
		panic("unknown id")
	}
}

func cmpCoefficents(p, q *fr.Vector) bool {
	if p.Len() != q.Len() {
		return false
	}
	for i := 0; i < p.Len(); i++ {
		if !((*p)[i].Equal(&(*q)[i])) {
			return false
		}
	}
	return true
}

func TestPutInLagrangeForm(t *testing.T) {

	size := 64
	domain := fft.NewDomain(uint64(size))

	// reference vector in Lagrange-regular form
	c := randomVector(size)
	p := NewPolynomial(c, Form{Basis: Canonical, Layout: Regular})

	// CANONICAL REGULAR
	{
		_p := fromLagrange(p, domain)
		q := _p.Clone()
		q.ToLagrange(domain)
		if q.Basis != Lagrange {
			t.Fatal("expected basis is Lagrange")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is BitReverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// CANONICAL BITREVERSE
	{
		_p := fromLagrange1(p, domain)
		q := _p.Clone()
		q.ToLagrange(domain)
		if q.Basis != Lagrange {
			t.Fatal("expected basis is Lagrange")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is Regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE REGULAR
	{
		_p := fromLagrange2(p, domain)
		q := _p.Clone()
		q.ToLagrange(domain)

		if q.Basis != Lagrange {
			t.Fatal("expected basis is Lagrange")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is Regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE BITREVERSE
	{
		_p := fromLagrange3(p, domain)
		q := _p.Clone()
		q.ToLagrange(domain)
		if q.Basis != Lagrange {
			t.Fatal("expected basis is Lagrange")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is BitReverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE_COSET REGULAR
	{
		_p := fromLagrange4(p, domain)
		q := _p.Clone()
		q.ToLagrange(domain)
		if q.Basis != Lagrange {
			t.Fatal("expected basis is Lagrange")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is Regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE_COSET BITREVERSE
	{
		_p := fromLagrange5(p, domain)
		q := _p.Clone()
		q.ToLagrange(domain)
		if q.Basis != Lagrange {
			t.Fatal("expected basis is Lagrange")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is BitReverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

}

// CANONICAL REGULAR
func fromCanonical0(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Canonical
	_p.Layout = Regular
	return _p
}

// CANONICAL BITREVERSE
func fromCanonical1(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Canonical
	_p.Layout = BitReverse
	return _p
}

// LAGRANGE REGULAR
func fromCanonical2(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Lagrange
	_p.Layout = Regular
	d.FFT(_p.Coefficients(), fft.DIF)
	fft.BitReverse(_p.Coefficients())
	return _p
}

// LAGRANGE BITREVERSE
func fromCanonical3(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Lagrange
	_p.Layout = BitReverse
	d.FFT(_p.Coefficients(), fft.DIF)
	return _p
}

// LAGRANGE_COSET REGULAR
func fromCanonical4(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = LagrangeCoset
	_p.Layout = Regular
	d.FFT(_p.Coefficients(), fft.DIF, fft.OnCoset())
	fft.BitReverse(_p.Coefficients())
	return _p
}

// LAGRANGE_COSET BITREVERSE
func fromCanonical5(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = LagrangeCoset
	_p.Layout = BitReverse
	d.FFT(_p.Coefficients(), fft.DIF, fft.OnCoset())
	return _p
}

func TestPutInCanonicalForm(t *testing.T) {

	size := 64
	domain := fft.NewDomain(uint64(size))

	// reference vector in canonical-regular form
	c := randomVector(size)
	p := NewPolynomial(c, Form{Basis: Canonical, Layout: Regular})

	// CANONICAL REGULAR
	{
		_p := fromCanonical0(p, domain)
		q := _p.Clone()
		q.ToCanonical(domain)
		if q.Basis != Canonical {
			t.Fatal("expected basis is canonical")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// CANONICAL BITREVERSE
	{
		_p := fromCanonical1(p, domain)
		q := _p.Clone()
		q.ToCanonical(domain)
		if q.Basis != Canonical {
			t.Fatal("expected basis is canonical")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is bitReverse")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE REGULAR
	{
		_p := fromCanonical2(p, domain)
		q := _p.Clone()
		q.ToCanonical(domain)
		if q.Basis != Canonical {
			t.Fatal("expected basis is canonical")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is bitReverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(p.coefficients, q.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE BITREVERSE
	{
		_p := fromCanonical3(p, domain)
		q := _p.Clone()
		q.ToCanonical(domain)
		if q.Basis != Canonical {
			t.Fatal("expected basis is canonical")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE_COSET REGULAR
	{
		_p := fromCanonical4(p, domain)
		q := _p.Clone()
		q.ToCanonical(domain)
		if q.Basis != Canonical {
			t.Fatal("expected basis is canonical")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is BitReverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE_COSET BITREVERSE
	{
		_p := fromCanonical5(p, domain)
		q := _p.Clone()
		q.ToCanonical(domain)
		if q.Basis != Canonical {
			t.Fatal("expected basis is canonical")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

}

// CANONICAL REGULAR
func fromLagrangeCoset0(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Canonical
	_p.Layout = Regular
	d.FFTInverse(_p.Coefficients(), fft.DIF, fft.OnCoset())
	fft.BitReverse(_p.Coefficients())
	return _p
}

// CANONICAL BITREVERSE
func fromLagrangeCoset1(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Canonical
	_p.Layout = BitReverse
	d.FFTInverse(_p.Coefficients(), fft.DIF, fft.OnCoset())
	return _p
}

// LAGRANGE REGULAR
func fromLagrangeCoset2(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Lagrange
	_p.Layout = Regular
	d.FFTInverse(_p.Coefficients(), fft.DIF, fft.OnCoset())
	d.FFT(_p.Coefficients(), fft.DIT)
	return _p
}

// LAGRANGE BITREVERSE
func fromLagrangeCoset3(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = Lagrange
	_p.Layout = BitReverse
	d.FFTInverse(_p.Coefficients(), fft.DIF, fft.OnCoset())
	d.FFT(_p.Coefficients(), fft.DIT)
	fft.BitReverse(_p.Coefficients())
	return _p
}

// LAGRANGE_COSET REGULAR
func fromLagrangeCoset4(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = LagrangeCoset
	_p.Layout = Regular
	return _p
}

// LAGRANGE_COSET BITREVERSE
func fromLagrangeCoset5(p *Polynomial, d *fft.Domain) *Polynomial {
	_p := p.Clone()
	_p.Basis = LagrangeCoset
	_p.Layout = BitReverse
	fft.BitReverse(p.Coefficients())
	return _p
}

func TestPutInLagrangeCosetForm(t *testing.T) {

	size := 64
	domain := fft.NewDomain(uint64(size))

	// reference vector in canonical-regular form
	c := randomVector(size)
	p := NewPolynomial(c, Form{Basis: LagrangeCoset, Layout: Regular})

	// CANONICAL REGULAR
	{
		_p := fromLagrangeCoset0(p, domain)
		q := _p.Clone()
		q.ToLagrangeCoset(domain)
		if q.Basis != LagrangeCoset {
			t.Fatal("expected basis is lagrange coset")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is bit reverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// CANONICAL BITREVERSE
	{
		_p := fromLagrangeCoset1(p, domain)
		q := _p.Clone()
		q.ToLagrangeCoset(domain)
		if q.Basis != LagrangeCoset {
			t.Fatal("expected basis is lagrange coset")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE REGULAR
	{
		_p := fromLagrangeCoset2(p, domain)
		q := _p.Clone()
		q.ToLagrangeCoset(domain)
		if q.Basis != LagrangeCoset {
			t.Fatal("expected basis is lagrange coset")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE BITREVERSE
	{
		_p := fromLagrangeCoset3(p, domain)
		q := _p.Clone()
		q.ToLagrangeCoset(domain)
		if q.Basis != LagrangeCoset {
			t.Fatal("expected basis is lagrange coset")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is bit reverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE_COSET REGULAR
	{
		_p := fromLagrangeCoset4(p, domain)
		q := _p.Clone()
		q.ToLagrangeCoset(domain)
		if q.Basis != LagrangeCoset {
			t.Fatal("expected basis is lagrange coset")
		}
		if q.Layout != Regular {
			t.Fatal("expected layout is regular")
		}
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

	// LAGRANGE_COSET BITREVERSE
	{
		_p := fromLagrangeCoset5(p, domain)
		q := _p.Clone()
		q.ToLagrangeCoset(domain)
		if q.Basis != LagrangeCoset {
			t.Fatal("expected basis is lagrange coset")
		}
		if q.Layout != BitReverse {
			t.Fatal("expected layout is bit reverse")
		}
		fft.BitReverse(q.Coefficients())
		if !cmpCoefficents(q.coefficients, p.coefficients) {
			t.Fatal("wrong coefficients")
		}
	}

}
