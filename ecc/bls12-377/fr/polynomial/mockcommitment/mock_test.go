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

package mockcommitment

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr"
	bls12377_pol "github.com/consensys/gnark-crypto/ecc/bls12-377/fr/polynomial"
	"github.com/consensys/gnark-crypto/polynomial"
)

func TestCommit(t *testing.T) {

	size := 60
	f := make(bls12377_pol.Polynomial, size)
	for i := 0; i < size; i++ {
		f[i].SetRandom()
	}

	var s Scheme

	c, err := s.Commit(&f)
	if err != nil {
		t.Fatal(err)
	}

	var _c fr.Element
	_c.SetInterface(c.Marshal())

	if !_c.Equal(&f[0]) {
		t.Fatal("err mock commitment (commit)")
	}
}

func TestVerifySinglePoint(t *testing.T) {

	size := 60
	f := make(bls12377_pol.Polynomial, size)
	for i := 0; i < size; i++ {
		f[i].SetRandom()
	}

	var s Scheme

	c, err := s.Commit(&f)
	if err != nil {
		t.Fatal(err)
	}

	var point fr.Element
	point.SetRandom()

	o, err := s.Open(point, &f)
	if err != nil {
		t.Fatal(err)
	}

	// verify the claimed valued
	res := o.(*MockProof)
	expected := f.Eval(point).(fr.Element)
	if !res.ClaimedValue.Equal(&expected) {
		t.Fatal("err mock commitment (open)")
	}

	err = s.Verify(c, o)
	if err != nil {
		t.Fatal(err)
	}

}

func TestBatchVerifySinglePoint(t *testing.T) {

	// create polynomials
	size := 60
	f := make([]polynomial.Polynomial, 10)
	for i := 0; i < 10; i++ {
		_f := make(bls12377_pol.Polynomial, size)
		for i := 0; i < size; i++ {
			_f[i].SetRandom()
		}
		f[i] = &_f
	}

	var s Scheme

	// commit the polynomials
	digests := make([]polynomial.Digest, 10)
	for i := 0; i < 10; i++ {
		digests[i], _ = s.Commit(f[i])
	}

	var point fr.Element
	point.SetRandom()

	proof, err := s.BatchOpenSinglePoint(&point, digests, f)
	if err != nil {
		t.Fatal(err)
	}

	// verify the claimed values
	_proof := proof.(*MockBatchProofsSinglePoint)
	for i := 0; i < 10; i++ {
		expectedClaim := f[i].Eval(point).(fr.Element)
		if !expectedClaim.Equal(&_proof.ClaimedValues[i]) {
			t.Fatal("inconsistant claimed values")
		}
	}

	// verify the proof
	err = s.BatchVerifySinglePoint(digests, proof)
	if err != nil {
		t.Fatal(err)
	}

}
