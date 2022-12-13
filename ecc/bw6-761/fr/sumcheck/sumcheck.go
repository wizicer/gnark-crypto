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
	"github.com/consensys/gnark-crypto/ecc/bw6-761/fr"
	"github.com/consensys/gnark-crypto/ecc/bw6-761/fr/polynomial"
	fiatshamir "github.com/consensys/gnark-crypto/fiat-shamir"
	"hash"
	"strconv"
)

// This does not make use of parallelism and represents polynomials as lists of coefficients
// It is currently geared towards arithmetic hashes. Once we have a more unified hash function interface, this can be generified.

// Claims to a multi-sumcheck statement. i.e. one of the form ∑_{0≤i<2ⁿ} fⱼ(i) = cⱼ for 1 ≤ j ≤ m.
// Later evolving into a claim of the form gⱼ = ∑_{0≤i<2ⁿ⁻ʲ} g(r₁, r₂, ..., rⱼ₋₁, Xⱼ, i...)
type Claims interface {
	Combine(a fr.Element) polynomial.Polynomial // Combine into the 0ᵗʰ sumcheck subclaim. Create g := ∑_{1≤j≤m} aʲ⁻¹fⱼ for which now we seek to prove ∑_{0≤i<2ⁿ} g(i) = c := ∑_{1≤j≤m} aʲ⁻¹cⱼ. Return g₁.
	Next(fr.Element) polynomial.Polynomial      // Return the evaluations gⱼ(k) for 1 ≤ k < degⱼ(g). Update the claim to gⱼ₊₁ for the input value as rⱼ
	VarsNum() int                               //number of variables
	ClaimsNum() int                             //number of claims
	ProveFinalEval(r []fr.Element) interface{}  //in case it is difficult for the verifier to compute g(r₁, ..., rₙ) on its own, the prover can provide the value and a proof
}

// LazyClaims is the Claims data structure on the verifier side. It is "lazy" in that it has to compute fewer things.
type LazyClaims interface {
	ClaimsNum() int                      // ClaimsNum = m
	VarsNum() int                        // VarsNum = n
	CombinedSum(a fr.Element) fr.Element // CombinedSum returns c = ∑_{1≤j≤m} aʲ⁻¹cⱼ
	Degree(i int) int                    //Degree of the total claim in the i'th variable
	VerifyFinalEval(r []fr.Element, combinationCoeff fr.Element, purportedValue fr.Element, proof interface{}) error
}

// Proof of a multi-sumcheck statement.
type Proof struct {
	PartialSumPolys []polynomial.Polynomial `json:"partialSumPolys"`
	FinalEvalProof  interface{}             `json:"finalEvalProof"` //in case it is difficult for the verifier to compute g(r₁, ..., rₙ) on its own, the prover can provide the value and a proof
}

func setupTranscript(claimsNum int, varsNum int, settings *fiatshamir.Settings) (challengeNames []string, err error) {
	numChallenges := varsNum
	if claimsNum >= 2 {
		numChallenges++
	}
	challengeNames = make([]string, numChallenges)
	if claimsNum >= 2 {
		challengeNames[0] = settings.Prefix + "combine"
	}
	prefix := settings.Prefix + "partialSumPolys."
	for i := 0; i < varsNum; i++ {
		challengeNames[i+numChallenges-varsNum] = prefix + strconv.Itoa(i)
	}
	if settings.Transcript == nil {
		transcript := fiatshamir.NewTranscript(settings.Hash, challengeNames...)
		settings.Transcript = &transcript
	}

	for i := range settings.BaseChallenges {
		err = settings.Transcript.Bind(challengeNames[0], settings.BaseChallenges[i])
	}
	return
}

func next(transcript *fiatshamir.Transcript, bindings []fr.Element, remainingChallengeNames *[]string) (fr.Element, error) {
	challengeName := (*remainingChallengeNames)[0]
	for i := range bindings {
		bytes := bindings[i].Bytes()
		if err := transcript.Bind(challengeName, bytes[:]); err != nil {
			return fr.Element{}, err
		}
	}
	var res fr.Element
	bytes, err := transcript.ComputeChallenge(challengeName)
	res.SetBytes(bytes)

	*remainingChallengeNames = (*remainingChallengeNames)[1:]

	return res, err
}

// Prove create a non-interactive sumcheck proof
func Prove(claims Claims, transcriptSettings fiatshamir.Settings) (Proof, error) {

	var proof Proof
	remainingChallengeNames, err := setupTranscript(claims.ClaimsNum(), claims.VarsNum(), &transcriptSettings)
	transcript := transcriptSettings.Transcript
	if err != nil {
		return proof, err
	}

	var combinationCoeff fr.Element
	if claims.ClaimsNum() >= 2 {
		if combinationCoeff, err = next(transcript, []fr.Element{}, &remainingChallengeNames); err != nil {
			return proof, err
		}
	}

	varsNum := claims.VarsNum()
	proof.PartialSumPolys = make([]polynomial.Polynomial, varsNum)
	proof.PartialSumPolys[0] = claims.Combine(combinationCoeff)
	challenges := make([]fr.Element, varsNum)

	for j := 0; j+1 < varsNum; j++ {
		if challenges[j], err = next(transcript, proof.PartialSumPolys[j], &remainingChallengeNames); err != nil {
			return proof, err
		}
		proof.PartialSumPolys[j+1] = claims.Next(challenges[j])
	}

	if challenges[varsNum-1], err = next(transcript, proof.PartialSumPolys[varsNum-1], &remainingChallengeNames); err != nil {
		return proof, err
	}

	proof.FinalEvalProof = claims.ProveFinalEval(challenges)

	return proof, nil
}

func Verify(claims LazyClaims, proof Proof, transcriptSettings fiatshamir.Settings) error {
	remainingChallengeNames, err := setupTranscript(claims.ClaimsNum(), claims.VarsNum(), &transcriptSettings)
	transcript := transcriptSettings.Transcript
	if err != nil {
		return err
	}

	var combinationCoeff fr.Element

	if claims.ClaimsNum() >= 2 {
		if combinationCoeff, err = next(transcript, []fr.Element{}, &remainingChallengeNames); err != nil {
			return err
		}
	}

	r := make([]fr.Element, claims.VarsNum())

	// Just so that there is enough room for gJ to be reused
	maxDegree := claims.Degree(0)
	for j := 1; j < claims.VarsNum(); j++ {
		if d := claims.Degree(j); d > maxDegree {
			maxDegree = d
		}
	}
	gJ := make(polynomial.Polynomial, maxDegree+1) //At the end of iteration j, gJ = ∑_{i < 2ⁿ⁻ʲ⁻¹} g(X₁, ..., Xⱼ₊₁, i...)		NOTE: n is shorthand for claims.VarsNum()
	gJR := claims.CombinedSum(combinationCoeff)    // At the beginning of iteration j, gJR = ∑_{i < 2ⁿ⁻ʲ} g(r₁, ..., rⱼ, i...)

	for j := 0; j < claims.VarsNum(); j++ {
		if len(proof.PartialSumPolys[j]) != claims.Degree(j) {
			return fmt.Errorf("malformed proof")
		}
		copy(gJ[1:], proof.PartialSumPolys[j])
		gJ[0].Sub(&gJR, &proof.PartialSumPolys[j][0]) // Requirement that gⱼ(0) + gⱼ(1) = gⱼ₋₁(r)
		// gJ is ready

		//Prepare for the next iteration
		if r[j], err = next(transcript, proof.PartialSumPolys[j], &remainingChallengeNames); err != nil {
			return err
		}
		// This is an extremely inefficient way of interpolating. TODO: Interpolate without symbolically computing a polynomial
		gJCoeffs := polynomial.InterpolateOnRange(gJ[:(claims.Degree(j) + 1)])
		gJR = gJCoeffs.Eval(&r[j])
	}

	return claims.VerifyFinalEval(r, combinationCoeff, gJR, proof.FinalEvalProof)
}

// -------- fiatshamir  --------- TODO: Replace with existing fiat-shamir impl

// This is a very bad fiat-shamir challenge generator
type MessageCounter struct {
	state uint64
	step  uint64
}

func (m *MessageCounter) Write(p []byte) (n int, err error) {
	inputBlockSize := (len(p)-1)/fr.Bytes + 1
	m.step += uint64(inputBlockSize) * m.step
	return len(p), nil
}

func (m *MessageCounter) Sum(b []byte) []byte {
	inputBlockSize := (len(b)-1)/fr.Bytes + 1
	resI := m.state + uint64(inputBlockSize)*m.step
	var res fr.Element
	res.SetInt64(int64(resI))
	resBytes := res.Bytes()
	return resBytes[:]
}

func (m *MessageCounter) Reset() {
	m.state = 0
}

func (m *MessageCounter) Size() int {
	return fr.Bytes
}

func (m *MessageCounter) BlockSize() int {
	return fr.Bytes
}

func NewMessageCounter(startState, step int) hash.Hash {
	transcript := &MessageCounter{state: uint64(startState), step: uint64(step)}
	return transcript
}

func NewMessageCounterGenerator(startState, step int) func() hash.Hash {
	return func() hash.Hash {
		return NewMessageCounter(startState, step)
	}
}
