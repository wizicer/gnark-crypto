// Copyright 2020 ConsenSys Software Inc.
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

// Code generated by consensys/gnark-crypto DO NOT EDIT

package kzg

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"reflect"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bw6-756"
	"github.com/consensys/gnark-crypto/ecc/bw6-756/fr"
	"github.com/consensys/gnark-crypto/ecc/bw6-756/fr/fft"
	"github.com/consensys/gnark-crypto/ecc/bw6-756/fr/polynomial"
)

// testSRS re-used accross tests of the KZG scheme
var testSRS *SRS

func init() {
	const srsSize = 230
	testSRS, _ = NewSRS(ecc.NextPowerOfTwo(srsSize), new(big.Int).SetInt64(42))
}

func TestDividePolyByXminusA(t *testing.T) {

	const pSize = 230

	// build random polynomial
	pol := make(polynomial.Polynomial, pSize)
	pol[0].SetRandom()
	for i := 1; i < pSize; i++ {
		pol[i] = pol[i-1]
	}

	// evaluate the polynomial at a random point
	var point fr.Element
	point.SetRandom()
	evaluation := pol.Eval(&point)

	// probabilistic test (using Schwartz Zippel lemma, evaluation at one point is enough)
	var randPoint, xminusa fr.Element
	randPoint.SetRandom()
	polRandpoint := pol.Eval(&randPoint)
	polRandpoint.Sub(&polRandpoint, &evaluation) // f(rand)-f(point)

	// compute f-f(a)/x-a
	h := dividePolyByXminusA(pol, evaluation, point)
	pol = nil // h reuses this memory

	if len(h) != 229 {
		t.Fatal("inconsistant size of quotient")
	}

	hRandPoint := h.Eval(&randPoint)
	xminusa.Sub(&randPoint, &point) // rand-point

	// f(rand)-f(point)	==? h(rand)*(rand-point)
	hRandPoint.Mul(&hRandPoint, &xminusa)

	if !hRandPoint.Equal(&polRandpoint) {
		t.Fatal("Error f-f(a)/x-a")
	}
}

func TestSerializationSRS(t *testing.T) {

	// create a SRS
	srs, err := NewSRS(64, new(big.Int).SetInt64(42))
	if err != nil {
		t.Fatal(err)
	}

	// serialize it...
	var buf bytes.Buffer
	_, err = srs.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}

	// reconstruct the SRS
	var _srs SRS
	_, err = _srs.ReadFrom(&buf)
	if err != nil {
		t.Fatal(err)
	}

	// compare
	if !reflect.DeepEqual(srs, &_srs) {
		t.Fatal("scheme serialization failed")
	}

}

func TestCommit(t *testing.T) {

	// create a polynomial
	f := make(polynomial.Polynomial, 60)
	for i := 0; i < 60; i++ {
		f[i].SetRandom()
	}

	// commit using the method from KZG
	_kzgCommit, err := Commit(f, testSRS)
	if err != nil {
		t.Fatal(err)
	}
	var kzgCommit bw6756.G1Affine
	kzgCommit.Unmarshal(_kzgCommit.Marshal())

	// check commitment using manual commit
	var x fr.Element
	x.SetString("42")
	fx := f.Eval(&x)
	var fxbi big.Int
	fx.ToBigIntRegular(&fxbi)
	var manualCommit bw6756.G1Affine
	manualCommit.Set(&testSRS.G1[0])
	manualCommit.ScalarMultiplication(&manualCommit, &fxbi)

	// compare both results
	if !kzgCommit.Equal(&manualCommit) {
		t.Fatal("error KZG commitment")
	}

}

func TestVerifySinglePoint(t *testing.T) {

	domain := fft.NewDomain(64, 0, false)

	// create a polynomial
	f := randomPolynomial(60)

	// commit the polynomial
	digest, err := Commit(f, testSRS)
	if err != nil {
		t.Fatal(err)
	}

	// compute opening proof at a random point
	var point fr.Element
	point.SetString("4321")
	proof, err := Open(f, &point, domain, testSRS)
	if err != nil {
		t.Fatal(err)
	}

	// verify the claimed valued
	expected := f.Eval(&point)
	if !proof.ClaimedValue.Equal(&expected) {
		t.Fatal("inconsistant claimed value")
	}

	// verify correct proof
	err = Verify(&digest, &proof, testSRS)
	if err != nil {
		t.Fatal(err)
	}

	// verify wrong proof
	proof.ClaimedValue.Double(&proof.ClaimedValue)
	err = Verify(&digest, &proof, testSRS)
	if err == nil {
		t.Fatal("verifying wrong proof should have failed")
	}
}

func TestBatchVerifySinglePoint(t *testing.T) {

	domain := fft.NewDomain(64, 0, false)

	// create polynomials
	f := make([]polynomial.Polynomial, 10)
	for i := 0; i < 10; i++ {
		f[i] = randomPolynomial(40 + 2*i)
	}

	// commit the polynomials
	digests := make([]Digest, 10)
	for i := 0; i < 10; i++ {
		digests[i], _ = Commit(f[i], testSRS)

	}

	// pick a hash function
	hf := sha256.New()

	// compute opening proof at a random point
	var point fr.Element
	point.SetString("4321")
	proof, err := BatchOpenSinglePoint(f, digests, &point, hf, domain, testSRS)
	if err != nil {
		t.Fatal(err)
	}

	// verify the claimed values
	for i := 0; i < 10; i++ {
		expectedClaim := f[i].Eval(&point)
		if !expectedClaim.Equal(&proof.ClaimedValues[i]) {
			t.Fatal("inconsistant claimed values")
		}
	}

	// verify correct proof
	err = BatchVerifySinglePoint(digests, &proof, hf, testSRS)
	if err != nil {
		t.Fatal(err)
	}

	// verify wrong proof
	proof.ClaimedValues[0].Double(&proof.ClaimedValues[0])
	err = BatchVerifySinglePoint(digests, &proof, hf, testSRS)
	if err == nil {
		t.Fatal("verifying wrong proof should have failed")
	}

}

func TestBatchVerifyMultiPoints(t *testing.T) {

	domain := fft.NewDomain(64, 0, false)

	// create polynomials
	f := make([]polynomial.Polynomial, 10)
	for i := 0; i < 10; i++ {
		f[i] = randomPolynomial(40 + 2*i)
	}

	// commit the polynomials
	digests := make([]Digest, 10)
	for i := 0; i < 10; i++ {
		digests[i], _ = Commit(f[i], testSRS)
	}

	// pick a hash function
	hf := sha256.New()

	// compute 2 batch opening proofs at 2 random points
	points := make([]fr.Element, 2)
	batchProofs := make([]BatchOpeningProof, 2)
	points[0].SetRandom()
	batchProofs[0], _ = BatchOpenSinglePoint(f[:5], digests[:5], &points[0], hf, domain, testSRS)
	points[1].SetRandom()
	batchProofs[1], _ = BatchOpenSinglePoint(f[5:], digests[5:], &points[1], hf, domain, testSRS)

	// fold the 2 batch opening proofs
	proofs := make([]OpeningProof, 2)
	foldedDigests := make([]Digest, 2)
	proofs[0], foldedDigests[0], _ = FoldProof(digests[:5], &batchProofs[0], hf)
	proofs[1], foldedDigests[1], _ = FoldProof(digests[5:], &batchProofs[1], hf)

	// check the the individual batch proofs are correct
	err := Verify(&foldedDigests[0], &proofs[0], testSRS)
	if err != nil {
		t.Fatal(err)
	}
	err = Verify(&foldedDigests[1], &proofs[1], testSRS)
	if err != nil {
		t.Fatal(err)
	}

	// batch verify correct folded proofs
	err = BatchVerifyMultiPoints(foldedDigests, proofs, testSRS)
	if err != nil {
		t.Fatal(err)
	}

	// batch verify tampered folded proofs
	proofs[0].ClaimedValue.Double(&proofs[0].ClaimedValue)
	err = BatchVerifyMultiPoints(foldedDigests, proofs, testSRS)
	if err == nil {
		t.Fatal(err)
	}

}

const benchSize = 1 << 16

func BenchmarkKZGCommit(b *testing.B) {
	benchSRS, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	if err != nil {
		b.Fatal(err)
	}
	// random polynomial
	p := randomPolynomial(benchSize / 2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Commit(p, benchSRS)
	}
}

func BenchmarkDivideByXMinusA(b *testing.B) {
	const pSize = 1 << 22

	// build random polynomial
	pol := make(polynomial.Polynomial, pSize)
	pol[0].SetRandom()
	for i := 1; i < pSize; i++ {
		pol[i] = pol[i-1]
	}
	var a, fa fr.Element
	a.SetRandom()
	fa.SetRandom()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dividePolyByXminusA(pol, fa, a)
		pol = pol[:pSize]
		pol[pSize-1] = pol[0]
	}
}

func BenchmarkKZGOpen(b *testing.B) {
	benchSRS, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	if err != nil {
		b.Fatal(err)
	}
	domain := fft.NewDomain(uint64(benchSize), 0, false)

	// random polynomial
	p := randomPolynomial(benchSize / 2)
	var r fr.Element
	r.SetRandom()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Open(p, &r, domain, benchSRS)
	}
}

func BenchmarkKZGVerify(b *testing.B) {
	benchSRS, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	if err != nil {
		b.Fatal(err)
	}
	// kzg scheme
	domain := fft.NewDomain(uint64(benchSize), 0, false)

	// random polynomial
	p := randomPolynomial(benchSize / 2)
	var r fr.Element
	r.SetRandom()

	// commit
	comm, err := Commit(p, benchSRS)
	if err != nil {
		b.Fatal(err)
	}

	// open
	openingProof, err := Open(p, &r, domain, benchSRS)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(&comm, &openingProof, benchSRS)
	}
}

func BenchmarkKZGBatchOpen10(b *testing.B) {
	benchSRS, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	if err != nil {
		b.Fatal(err)
	}
	domain := fft.NewDomain(uint64(benchSize), 0, false)

	// 10 random polynomials
	var ps [10]polynomial.Polynomial
	for i := 0; i < 10; i++ {
		ps[i] = randomPolynomial(benchSize / 2)
	}

	// commitments
	var commitments [10]Digest
	for i := 0; i < 10; i++ {
		commitments[i], _ = Commit(ps[i], benchSRS)
	}

	// pick a hash function
	hf := sha256.New()

	var r fr.Element
	r.SetRandom()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchOpenSinglePoint(ps[:], commitments[:], &r, hf, domain, benchSRS)
	}
}

func BenchmarkKZGBatchVerify10(b *testing.B) {
	benchSRS, err := NewSRS(ecc.NextPowerOfTwo(benchSize), new(big.Int).SetInt64(42))
	if err != nil {
		b.Fatal(err)
	}
	domain := fft.NewDomain(uint64(benchSize), 0, false)

	// 10 random polynomials
	var ps [10]polynomial.Polynomial
	for i := 0; i < 10; i++ {
		ps[i] = randomPolynomial(benchSize / 2)
	}

	// commitments
	var commitments [10]Digest
	for i := 0; i < 10; i++ {
		commitments[i], _ = Commit(ps[i], benchSRS)
	}

	// pick a hash function
	hf := sha256.New()

	var r fr.Element
	r.SetRandom()

	proof, err := BatchOpenSinglePoint(ps[:], commitments[:], &r, hf, domain, benchSRS)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchVerifySinglePoint(commitments[:], &proof, hf, benchSRS)
	}
}

func randomPolynomial(size int) polynomial.Polynomial {
	f := make(polynomial.Polynomial, size)
	for i := 0; i < size; i++ {
		f[i].SetRandom()
	}
	return f
}
