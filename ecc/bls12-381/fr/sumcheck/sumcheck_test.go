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

package sumcheck

import (
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr/polynomial"
	fiatshamir "github.com/consensys/gnark-crypto/fiat-shamir"
	"github.com/stretchr/testify/assert"
	"hash"
	"math/bits"
	"strings"
	"testing"
)

type singleMultilinClaim struct {
	g polynomial.MultiLin
}

func (c singleMultilinClaim) ProveFinalEval(r []fr.Element) interface{} {
	return nil // verifier can compute the final eval itself
}

func (c singleMultilinClaim) VarsNum() int {
	return bits.TrailingZeros(uint(len(c.g)))
}

func (c singleMultilinClaim) ClaimsNum() int {
	return 1
}

func sumForX1One(g polynomial.MultiLin) polynomial.Polynomial {
	sum := g[len(g)/2]
	for i := len(g)/2 + 1; i < len(g); i++ {
		sum.Add(&sum, &g[i])
	}
	return []fr.Element{sum}
}

func (c singleMultilinClaim) Combine(fr.Element) polynomial.Polynomial {
	return sumForX1One(c.g)
}

func (c *singleMultilinClaim) Next(r fr.Element) polynomial.Polynomial {
	c.g.Fold(r)
	return sumForX1One(c.g)
}

type singleMultilinLazyClaim struct {
	g          polynomial.MultiLin
	claimedSum fr.Element
}

func (c singleMultilinLazyClaim) VerifyFinalEval(r []fr.Element, combinationCoeff fr.Element, purportedValue fr.Element, proof interface{}) error {
	val := c.g.Evaluate(r, nil)
	if val.Equal(&purportedValue) {
		return nil
	}
	return fmt.Errorf("mismatch")
}

func (c singleMultilinLazyClaim) CombinedSum(combinationCoeffs fr.Element) fr.Element {
	return c.claimedSum
}

func (c singleMultilinLazyClaim) Degree(i int) int {
	return 1
}

func (c singleMultilinLazyClaim) ClaimsNum() int {
	return 1
}

func (c singleMultilinLazyClaim) VarsNum() int {
	return bits.TrailingZeros(uint(len(c.g)))
}

func testSumcheckSingleClaimMultilin(polyInt []uint64, hashGenerator func() hash.Hash) error {
	poly := make(polynomial.MultiLin, len(polyInt))
	for i, n := range polyInt {
		poly[i].SetUint64(n)
	}

	claim := singleMultilinClaim{g: poly.Clone()}

	proof, err := Prove(&claim, fiatshamir.WithHash(hashGenerator()))
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, p := range proof.PartialSumPolys {

		sb.WriteString("\t{")
		for i := 0; i < len(p); i++ {
			sb.WriteString(p[i].String())
			if i+1 < len(p) {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("}\n")
	}

	lazyClaim := singleMultilinLazyClaim{g: poly, claimedSum: poly.Sum()}
	if err = Verify(lazyClaim, proof, fiatshamir.WithHash(hashGenerator())); err != nil {
		return err
	}

	proof.PartialSumPolys[0][0].Add(&proof.PartialSumPolys[0][0], &fr.Element{1})
	lazyClaim = singleMultilinLazyClaim{g: poly, claimedSum: poly.Sum()}
	if Verify(lazyClaim, proof, fiatshamir.WithHash(hashGenerator())) == nil {
		return fmt.Errorf("incorrect proof accepted")
	}
	return nil
}

func TestSumcheckDeterministicHashSingleClaimMultilin(t *testing.T) {
	//printMsws(36)

	polys := [][]uint64{
		{1, 2, 3, 4},             // 1 + 2X₁ + X₂
		{1, 2, 3, 4, 5, 6, 7, 8}, // 1 + 4X₁ + 2X₂ + X₃
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, // 1 + 8X₁ + 4X₂ + 2X₃ + X₄
	}

	const MaxStep = 4
	const MaxStart = 4
	hashGens := make([]func() hash.Hash, 0, MaxStart*MaxStep)

	for step := 0; step < MaxStep; step++ {
		for startState := 0; startState < MaxStart; startState++ {
			hashGens = append(hashGens, NewMessageCounterGenerator(startState, step))
		}
	}

	for _, poly := range polys {
		for _, hashGen := range hashGens {
			assert.NoError(t, testSumcheckSingleClaimMultilin(poly, hashGen),
				"failed with poly %v and hashGen %v", poly, hashGen())
		}
	}
}
