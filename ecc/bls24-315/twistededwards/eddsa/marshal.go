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

package eddsa

import (
	"crypto/subtle"
	"io"
)

// Bytes returns the binary representation of the public key
// follows https://tools.ietf.org/html/rfc8032#section-3.1
// and returns a compressed representation of the point (x,y)
//
// x, y are the coordinates of the point
// on the twisted Edwards as big endian integers.
// compressed representation store x with a parity bit to recompute y
func (pk *PublicKey) Bytes() []byte {
	var res [sizePublicKey]byte
	pkBin := pk.A.Bytes()
	subtle.ConstantTimeCopy(1, res[:sizeFr], pkBin[:])
	return res[:]
}

// SetBytes sets p from binary representation in buf.
// buf represents a public key as x||y where x, y are
// interpreted as big endian binary numbers corresponding
// to the coordinates of a point on the twisted Edwards.
// It returns the number of bytes read from the buffer.
func (pk *PublicKey) SetBytes(buf []byte) (int, error) {
	n := 0
	if len(buf) < sizePublicKey {
		return n, io.ErrShortBuffer
	}
	if _, err := pk.A.SetBytes(buf[:sizeFr]); err != nil {
		return 0, err
	}
	n += sizeFr
	if !pk.A.IsOnCurve() {
		return n, errNotOnCurve
	}
	return n, nil
}

// Bytes returns the binary representation of pk,
// as byte array publicKey||scalar||randSrc
// where publicKey is as publicKey.Bytes(), and
// scalar is in big endian, of size sizeFr.
func (privKey *PrivateKey) Bytes() []byte {
	var res [sizePrivateKey]byte
	pubkBin := privKey.PublicKey.A.Bytes()
	subtle.ConstantTimeCopy(1, res[:sizeFr], pubkBin[:])
	subtle.ConstantTimeCopy(1, res[sizeFr:2*sizeFr], privKey.scalar[:])
	subtle.ConstantTimeCopy(1, res[2*sizeFr:], privKey.randSrc[:])
	return res[:]
}

// SetBytes sets pk from buf, where buf is interpreted
// as  publicKey||scalar||randSrc
// where publicKey is as publicKey.Bytes(), and
// scalar is in big endian, of size sizeFr.
// It returns the number byte read.
func (privKey *PrivateKey) SetBytes(buf []byte) (int, error) {
	n := 0
	if len(buf) < sizePrivateKey {
		return n, io.ErrShortBuffer
	}
	if _, err := privKey.PublicKey.A.SetBytes(buf[:sizeFr]); err != nil {
		return 0, err
	}
	n += sizeFr
	if !privKey.PublicKey.A.IsOnCurve() {
		return n, errNotOnCurve
	}
	subtle.ConstantTimeCopy(1, privKey.scalar[:], buf[sizeFr:2*sizeFr])
	n += sizeFr
	subtle.ConstantTimeCopy(1, privKey.randSrc[:], buf[2*sizeFr:])
	n += sizeFr
	return n, nil
}

// Bytes returns the binary representation of sig
// as a byte array of size 3*sizeFr x||y||s where
//   - x, y are the coordinates of a point on the twisted
//     Edwards represented in big endian
//   - s=r+h(r,a,m) mod l, the Hasse bound guarantess that
//     s is smaller than sizeFr (in particular it is supposed
//     s is NOT blinded)
func (sig *Signature) Bytes() []byte {
	var res [sizeSignature]byte
	sigRBin := sig.R.Bytes()
	subtle.ConstantTimeCopy(1, res[:sizeFr], sigRBin[:])
	subtle.ConstantTimeCopy(1, res[sizeFr:], sig.S[:])
	return res[:]
}

// SetBytes sets sig from a buffer in binary.
// buf is read interpreted as x||y||s where
//   - x,y are the coordinates of a point on the twisted
//     Edwards represented in big endian
//   - s=r+h(r,a,m) mod l, the Hasse bound guarantess that
//     s is smaller than sizeFr (in particular it is supposed
//     s is NOT blinded)
//
// It returns the number of bytes read from buf.
func (sig *Signature) SetBytes(buf []byte) (int, error) {
	n := 0
	if len(buf) < sizeSignature {
		return n, io.ErrShortBuffer
	}
	if _, err := sig.R.SetBytes(buf[:sizeFr]); err != nil {
		return 0, err
	}
	n += sizeFr
	if !sig.R.IsOnCurve() {
		return n, errNotOnCurve
	}
	subtle.ConstantTimeCopy(1, sig.S[:], buf[sizeFr:2*sizeFr])
	n += sizeFr
	return n, nil
}
