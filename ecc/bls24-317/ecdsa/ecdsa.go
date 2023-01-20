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

package ecdsa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"io"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bls24-317"
	"github.com/consensys/gnark-crypto/ecc/bls24-317/fr"
)

// PublicKey represents an ECDSA public key
type PublicKey struct {
	Q bls24317.G1Affine
}

// PrivateKey represents an ECDSA private key
type PrivateKey struct {
	PublicKey
	Secret *big.Int
}

// Signature represents an ECDSA signature
type Signature struct {
	R, S *big.Int
}

var one = new(big.Int).SetInt64(1)

// randFieldElement returns a random element of the order of the given
// curve using the procedure given in FIPS 186-4, Appendix B.5.1.
func randFieldElement(rand io.Reader) (k *big.Int, err error) {
	b := make([]byte, fr.Bits/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}

	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(fr.Modulus(), one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}

// GenerateKey generates a public and private key pair.
func GenerateKey(rand io.Reader) (*PrivateKey, error) {

	k, err := randFieldElement(rand)
	if err != nil {
		return nil, err

	}
	_, _, g, _ := bls24317.Generators()

	privateKey := new(PrivateKey)
	privateKey.Secret = k
	privateKey.PublicKey.Q.ScalarMultiplication(&g, k)
	return privateKey, nil
}

// hashToInt converts a hash value to an integer. Per FIPS 186-4, Section 6.4,
// we use the left-most bits of the hash to match the bit-length of the order of
// the curve. This also performs Step 5 of SEC 1, Version 2.0, Section 4.1.3.
func hashToInt(hash []byte) *big.Int {
	if len(hash) > fr.Bytes {
		hash = hash[:fr.Bytes]
	}

	ret := new(big.Int).SetBytes(hash)
	excess := len(hash)*8 - fr.Bits
	if excess > 0 {
		ret.Rsh(ret, uint(excess))
	}
	return ret
}

type zr struct{}

// Read replaces the contents of dst with zeros. It is safe for concurrent use.
func (zr) Read(dst []byte) (n int, err error) {
	for i := range dst {
		dst[i] = 0
	}
	return len(dst), nil
}

var zeroReader = zr{}

const (
	aesIV = "gnark-crypto IV." // must be 16 chars (equal block size)
)

func nonce(rand io.Reader, privateKey *PrivateKey, hash []byte) (csprng *cipher.StreamReader, err error) {
	// This implementation derives the nonce from an AES-CTR CSPRNG keyed by:
	//
	//    SHA2-512(privateKey.Secret ∥ entropy ∥ hash)[:32]
	//
	// The CSPRNG key is indifferentiable from a random oracle as shown in
	// [Coron], the AES-CTR stream is indifferentiable from a random oracle
	// under standard cryptographic assumptions (see [Larsson] for examples).
	//
	// [Coron]: https://cs.nyu.edu/~dodis/ps/merkle.pdf
	// [Larsson]: https://web.archive.org/web/20040719170906/https://www.nada.kth.se/kurser/kth/2D1441/semteo03/lecturenotes/assump.pdf

	// Get 256 bits of entropy from rand.
	entropy := make([]byte, 32)
	_, err = io.ReadFull(rand, entropy)
	if err != nil {
		return

	}

	// Initialize an SHA-512 hash context; digest...
	md := sha512.New()
	md.Write(privateKey.Secret.Bytes()) // the private key,
	md.Write(entropy)                   // the entropy,
	md.Write(hash)                      // and the input hash;
	key := md.Sum(nil)[:32]             // and compute ChopMD-256(SHA-512),
	// which is an indifferentiable MAC.

	// Create an AES-CTR instance to use as a CSPRNG.
	block, _ := aes.NewCipher(key)

	// Create a CSPRNG that xors a stream of zeros with
	// the output of the AES-CTR instance.
	csprng = &cipher.StreamReader{
		R: zeroReader,
		S: cipher.NewCTR(block, []byte(aesIV)),
	}

	return csprng, err
}

// Sign performs the ECDSA signature
//
// k ← 𝔽r (random)
// P = k ⋅ g1Gen
// r = x_P (mod order)
// s = k⁻¹ . (m + sk ⋅ r)
// signature = {s, r}
//
// SEC 1, Version 2.0, Section 4.1.3
func Sign(hash []byte, privateKey PrivateKey, rand io.Reader) (signature Signature, err error) {
	order := fr.Modulus()
	r, s, kInv := new(big.Int), new(big.Int), new(big.Int)
	for {
		for {
			csprng, err := nonce(rand, &privateKey, hash)
			if err != nil {
				return Signature{}, err
			}
			k, err := randFieldElement(csprng)
			if err != nil {
				return Signature{}, err
			}

			var P bls24317.G1Affine
			P.ScalarMultiplicationBase(k)
			kInv.ModInverse(k, order)

			P.X.BigInt(r)
			r.Mod(r, order)
			if r.Sign() != 0 {
				break
			}
		}
		s.Mul(r, privateKey.Secret)
		m := hashToInt(hash)
		s.Add(m, s).
			Mul(kInv, s).
			Mod(s, order) // order != 0
		if s.Sign() != 0 {
			break
		}
	}

	signature.R, signature.S = r, s

	return signature, err
}

// Verify validates the ECDSA signature
//
// R ?= (s⁻¹ ⋅ m ⋅ Base + s⁻¹ ⋅ R ⋅ publiKey)_x
//
// SEC 1, Version 2.0, Section 4.1.4
func Verify(hash []byte, signature Signature, publicKey bls24317.G1Affine) bool {

	order := fr.Modulus()

	if signature.R.Sign() <= 0 || signature.S.Sign() <= 0 {
		return false
	}
	if signature.R.Cmp(order) >= 0 || signature.S.Cmp(order) >= 0 {
		return false
	}

	sInv := new(big.Int).ModInverse(signature.S, order)
	e := hashToInt(hash)
	u1 := new(big.Int).Mul(e, sInv)
	u1.Mod(u1, order)
	u2 := new(big.Int).Mul(signature.R, sInv)
	u2.Mod(u2, order)
	var U bls24317.G1Jac
	U.JointScalarMultiplicationBase(&publicKey, u1, u2)

	var z big.Int
	U.Z.Square(&U.Z).
		Inverse(&U.Z).
		Mul(&U.Z, &U.X).
		BigInt(&z)

	z.Mod(&z, order)

	return z.Cmp(signature.R) == 0

}