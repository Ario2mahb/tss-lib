// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package facproof

import (
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/common"
)

const (
	ProofFacBytesParts = 11
)

type (
	ProofFac struct {
		P, Q, A, B, T, Sigma, Z1, Z2, W1, W2, V *big.Int
	}
)

var (
	// rangeParameter l limits the bits of p or q to be in [1024-l, 1024+l]
	rangeParameter = new(big.Int).Lsh(big.NewInt(1), 15)
	one            = big.NewInt(1)
)

// NewProof implements proofFac
func NewProof(ec elliptic.Curve, N0, NCap, s, t, N0p, N0q *big.Int) (*ProofFac, error) {
	if ec == nil || N0 == nil || NCap == nil || s == nil || t == nil || N0p == nil || N0q == nil {
		return nil, errors.New("ProveFac constructor received nil value(s)")
	}

	q := ec.Params().N
	sqrtN0 := new(big.Int).Sqrt(N0)

	leSqrtN0 := new(big.Int).Mul(rangeParameter, q)
	leSqrtN0 = new(big.Int).Mul(leSqrtN0, sqrtN0)
	lNCap := new(big.Int).Mul(rangeParameter, NCap)
	lN0NCap := new(big.Int).Mul(lNCap, N0)
	leN0NCap := new(big.Int).Mul(lN0NCap, q)
	leNCap := new(big.Int).Mul(lNCap, q)

	// Fig 28.1 sample
	alpha := common.GetRandomPositiveInt(leSqrtN0)
	beta := common.GetRandomPositiveInt(leSqrtN0)
	mu := common.GetRandomPositiveInt(lNCap)
	nu := common.GetRandomPositiveInt(lNCap)
	sigma := common.GetRandomPositiveInt(lN0NCap)
	r := common.GetRandomPositiveRelativelyPrimeInt(leN0NCap)
	x := common.GetRandomPositiveInt(leNCap)
	y := common.GetRandomPositiveInt(leNCap)

	// Fig 28.1 compute
	modNCap := common.ModInt(NCap)
	P := modNCap.Exp(s, N0p)
	P = modNCap.Mul(P, modNCap.Exp(t, mu))

	Q := modNCap.Exp(s, N0q)
	Q = modNCap.Mul(Q, modNCap.Exp(t, nu))

	A := modNCap.Exp(s, alpha)
	A = modNCap.Mul(A, modNCap.Exp(t, x))

	B := modNCap.Exp(s, beta)
	B = modNCap.Mul(B, modNCap.Exp(t, y))

	T := modNCap.Exp(Q, alpha)
	T = modNCap.Mul(T, modNCap.Exp(t, r))

	// Fig 28.2 e
	var e *big.Int
	{
		eHash := common.SHA512_256i(N0, NCap, s, t, P, Q, A, B, T, sigma)
		e = common.RejectionSample(q, eHash)
	}

	// Fig 28.3
	z1 := new(big.Int).Mul(e, N0p)
	z1 = new(big.Int).Add(z1, alpha)

	z2 := new(big.Int).Mul(e, N0q)
	z2 = new(big.Int).Add(z2, beta)

	w1 := new(big.Int).Mul(e, mu)
	w1 = new(big.Int).Add(w1, x)

	w2 := new(big.Int).Mul(e, nu)
	w2 = new(big.Int).Add(w2, y)

	v := new(big.Int).Mul(nu, N0p)
	v = new(big.Int).Sub(sigma, v)
	v = new(big.Int).Mul(e, v)
	v = new(big.Int).Add(v, r)

	return &ProofFac{P: P, Q: Q, A: A, B: B, T: T, Sigma: sigma, Z1: z1, Z2: z2, W1: w1, W2: w2, V: v}, nil
}

func NewProofFromBytes(bzs [][]byte) (*ProofFac, error) {
	if !common.NonEmptyMultiBytes(bzs, ProofFacBytesParts) {
		return nil, fmt.Errorf("expected %d byte parts to construct ProofFac", ProofFacBytesParts)
	}
	return &ProofFac{
		P:     new(big.Int).SetBytes(bzs[0]),
		Q:     new(big.Int).SetBytes(bzs[1]),
		A:     new(big.Int).SetBytes(bzs[2]),
		B:     new(big.Int).SetBytes(bzs[3]),
		T:     new(big.Int).SetBytes(bzs[4]),
		Sigma: new(big.Int).SetBytes(bzs[5]),
		Z1:    new(big.Int).SetBytes(bzs[6]),
		Z2:    new(big.Int).SetBytes(bzs[7]),
		W1:    new(big.Int).SetBytes(bzs[8]),
		W2:    new(big.Int).SetBytes(bzs[9]),
		V:     new(big.Int).SetBytes(bzs[10]),
	}, nil
}

func (pf *ProofFac) Verify(ec elliptic.Curve, N0, NCap, s, t *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() || ec == nil || N0 == nil || NCap == nil || s == nil || t == nil {
		return false
	}
	if N0.Sign() != 1 {
		return false
	}
	if NCap.Sign() != 1 {
		return false
	}

	q := ec.Params().N
	sqrtN0 := new(big.Int).Sqrt(N0)

	leSqrtN0 := new(big.Int).Mul(rangeParameter, q)
	leSqrtN0 = new(big.Int).Mul(leSqrtN0, sqrtN0)
	lNCap := new(big.Int).Mul(rangeParameter, NCap)
	lN0NCap := new(big.Int).Mul(lNCap, N0)
	leN0NCap2 := new(big.Int).Lsh(new(big.Int).Mul(lN0NCap, q), 1)
	leNCap2 := new(big.Int).Lsh(new(big.Int).Mul(lNCap, q), 1)

	if !common.IsInInterval(pf.P, NCap) {
		return false
	}
	if !common.IsInInterval(pf.Q, NCap) {
		return false
	}
	if !common.IsInInterval(pf.A, NCap) {
		return false
	}
	if !common.IsInInterval(pf.B, NCap) {
		return false
	}
	if !common.IsInInterval(pf.T, NCap) {
		return false
	}
	if !common.IsInInterval(pf.Sigma, lN0NCap) {
		return false
	}
	if new(big.Int).GCD(nil, nil, pf.P, NCap).Cmp(one) != 0 {
		return false
	}
	if new(big.Int).GCD(nil, nil, pf.Q, NCap).Cmp(one) != 0 {
		return false
	}
	if new(big.Int).GCD(nil, nil, pf.A, NCap).Cmp(one) != 0 {
		return false
	}
	if new(big.Int).GCD(nil, nil, pf.B, NCap).Cmp(one) != 0 {
		return false
	}
	if new(big.Int).GCD(nil, nil, pf.T, NCap).Cmp(one) != 0 {
		return false
	}
	if !common.IsInInterval(pf.W1, leNCap2) {
		return false
	}
	if !common.IsInInterval(pf.W2, leNCap2) {
		return false
	}
	if !common.IsInInterval(pf.V, leN0NCap2) {
		return false
	}

	// Fig 28. Range Check
	if !common.IsInInterval(pf.Z1, leSqrtN0) {
		return false
	}

	if !common.IsInInterval(pf.Z2, leSqrtN0) {
		return false
	}

	var e *big.Int
	{
		eHash := common.SHA512_256i(N0, NCap, s, t, pf.P, pf.Q, pf.A, pf.B, pf.T, pf.Sigma)
		e = common.RejectionSample(q, eHash)
	}

	// Fig 28. Equality Check
	modNCap := common.ModInt(NCap)
	{
		LHS := modNCap.Mul(modNCap.Exp(s, pf.Z1), modNCap.Exp(t, pf.W1))
		RHS := modNCap.Mul(pf.A, modNCap.Exp(pf.P, e))

		if LHS.Cmp(RHS) != 0 {
			return false
		}
	}

	{
		LHS := modNCap.Mul(modNCap.Exp(s, pf.Z2), modNCap.Exp(t, pf.W2))
		RHS := modNCap.Mul(pf.B, modNCap.Exp(pf.Q, e))

		if LHS.Cmp(RHS) != 0 {
			return false
		}
	}

	{
		R := modNCap.Mul(modNCap.Exp(s, N0), modNCap.Exp(t, pf.Sigma))
		LHS := modNCap.Mul(modNCap.Exp(pf.Q, pf.Z1), modNCap.Exp(t, pf.V))
		RHS := modNCap.Mul(pf.T, modNCap.Exp(R, e))

		if LHS.Cmp(RHS) != 0 {
			return false
		}
	}

	return true
}

func (pf *ProofFac) ValidateBasic() bool {
	return pf.P != nil &&
		pf.Q != nil &&
		pf.A != nil &&
		pf.B != nil &&
		pf.T != nil &&
		pf.Sigma != nil &&
		pf.Z1 != nil &&
		pf.Z2 != nil &&
		pf.W1 != nil &&
		pf.W2 != nil &&
		pf.V != nil
}

func (pf *ProofFac) Bytes() [ProofFacBytesParts][]byte {
	return [...][]byte{
		pf.P.Bytes(),
		pf.Q.Bytes(),
		pf.A.Bytes(),
		pf.B.Bytes(),
		pf.T.Bytes(),
		pf.Sigma.Bytes(),
		pf.Z1.Bytes(),
		pf.Z2.Bytes(),
		pf.W1.Bytes(),
		pf.W2.Bytes(),
		pf.V.Bytes(),
	}
}
