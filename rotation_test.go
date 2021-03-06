package smd

import (
	"math"
	"testing"
	"time"

	"github.com/gonum/matrix/mat64"
)

func TestR1R2R3(t *testing.T) {
	x := math.Pi / 3.0
	s, c := math.Sincos(x)
	r1 := R1(x)
	r2 := R2(x)
	r3 := R3(x)
	// Test items equal to 1.
	if r1.At(0, 0) != r2.At(1, 1) || r1.At(0, 0) != r3.At(2, 2) || r3.At(2, 2) != 1 {
		t.Fatal("expected R1.At(0, 0) = R2.At(1, 1) = R3.At(2, 2) = 1\n")
	}
	// Test items equal to 0.
	if r1.At(0, 1) != r1.At(0, 2) || r1.At(1, 0) != r1.At(2, 0) || r1.At(0, 1) != 0 {
		t.Fatal("misplaced zeros in R1\n")
	}
	if r2.At(0, 1) != r2.At(1, 2) || r2.At(1, 0) != r2.At(1, 2) || r2.At(1, 2) != 0 {
		t.Fatal("misplaced zeros in R2\n")
	}
	if r3.At(2, 0) != r3.At(2, 1) || r3.At(0, 2) != r3.At(1, 2) || r3.At(1, 2) != 0 {
		t.Fatal("misplaced zeros in R3\n")
	}
	// Test R1.
	if r1.At(1, 1) != r1.At(2, 2) || r1.At(2, 2) != c {
		t.Fatal("expected R1 cosines misplaced\n")
	}
	if r1.At(2, 1) != -r1.At(1, 2) || r1.At(1, 2) != s {
		t.Fatal("expected R1 sines misplaced\n")
	}
	// Test R2.
	if r2.At(0, 0) != r2.At(2, 2) || r2.At(2, 2) != c {
		t.Fatal("expected R2 cosines misplaced\n")
	}
	if r2.At(2, 0) != -r2.At(0, 2) || r2.At(2, 0) != s {
		t.Fatal("expected R2 sines misplaced\n")
	}
	// Test R3.
	if r3.At(1, 1) != r3.At(0, 0) || r3.At(0, 0) != c {
		t.Fatal("expected R3 cosines misplaced\n")
	}
	if r3.At(0, 1) != -r3.At(1, 0) || r3.At(0, 1) != s {
		t.Fatal("expected R3 sines misplaced\n")
	}
}

func TestRot313(t *testing.T) {
	var R1R3, R3R1R3m mat64.Dense
	θ1 := math.Pi / 17
	θ2 := math.Pi / 16
	θ3 := math.Pi / 15
	R1R3.Mul(R1(θ2), R3(θ1))
	R3R1R3m.Mul(R3(θ3), &R1R3)
	R3R1R3m.Sub(&R3R1R3m, R3R1R3(θ1, θ2, θ3))
	if !mat64.EqualApprox(&R3R1R3m, mat64.NewDense(3, 3, nil), 1e-16) {
		t.Logf("\n%+v", mat64.Formatted(&R3R1R3m))
		t.Logf("\n%+v", mat64.Formatted(R3R1R3(θ1, θ2, θ3)))
		t.Fatal("failed")
	}
}

func TestPQW2ECI(t *testing.T) {
	i := Deg2rad(87.87)
	ω := Deg2rad(53.38)
	Ω := Deg2rad(227.89)
	Rp := Rot313Vec(-ω, -i, -Ω, []float64{-466.7639, 11447.0219, 0})
	Re := []float64{6525.368103709379, 6861.531814548294, 6449.118636407358}
	if !vectorsEqual(Re, Rp) {
		t.Fatalf("R conversion failed:\n%+v\n%+v", Re, Rp)
	}
	Vp := Rot313Vec(-ω, -i, -Ω, []float64{-5.996222, 4.753601, 0})
	Ve := []float64{4.902278620687254, 5.533139558121602, -1.9757104281719946}
	if !vectorsEqual(Ve, Vp) {
		t.Fatalf("V conversion failed:\n%+v\n%+v", Ve, Vp)
	}
	// Test Matrix rotation.
	vec := []float64{1e-12, 0, 0}
	noiseQ := mat64.NewSymDense(3, []float64{vec[0], 0, 0, 0, vec[1], 0, 0, 0, vec[2]})
	dcm := R3R1R3(-ω, -i, -Ω)
	var QECI, QECI0 mat64.Dense
	QECI0.Mul(noiseQ, dcm.T())
	QECI.Mul(dcm, &QECI0)
	t.Logf("\n%+v\n", mat64.Formatted(&QECI))
}

func TestGEO2ECEF(t *testing.T) {
	latitude := 34.352496 * deg2rad
	longitude := 46.4464 * deg2rad
	r := GEO2ECEF(5085.22, latitude, longitude)
	rExp := []float64{6520.963141870237, 6858.799558071129, 6468.573721101338}
	if !vectorsEqual(r, rExp) {
		t.Fatalf("Got: %+v", r)
	}
}

func TestECInECEF(t *testing.T) {
	// TODO: Improve this test.
	r := []float64{6520.963141870237, 6858.799558071129, 6468.573721101338}
	for _, dur := range []time.Duration{0, time.Hour} {
		angle := EarthRotationRate * dur.Seconds()
		rPrime := ECI2ECEF(ECEF2ECI(r, angle), angle)
		if !vectorsEqual(r, rPrime) {
			t.Fatal("not reversible")
		}
	}
}
