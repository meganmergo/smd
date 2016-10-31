package dynamics

import (
	"fmt"
	"math"
	"time"
)

// Orbit defines an orbit via its orbital elements.
type Orbit struct {
	R      []float64       // Radius vector
	V      []float64       // Velocity vector
	Origin CelestialObject // Orbit orgin
}

// GetOE returns the orbital elements of this orbit.
func (o *Orbit) GetOE() (a, e, i, ω, Ω, ν float64) {
	h := cross(o.R, o.V)

	N := []float64{-h[1], h[0], 0}

	eVec := make([]float64, 3)
	for j := 0; j < 3; j++ {
		eVec[j] = ((math.Pow(norm(o.V), 2)-o.Origin.μ/norm(o.R))*o.R[j] - dot(o.R, o.V)*o.V[j]) / o.Origin.μ
	}
	e = norm(eVec) // Eccentricity
	// We suppose the orbit is NOT parabolic.
	a = -o.Origin.μ / (2 * (0.5*dot(o.V, o.V) - o.Origin.μ/norm(o.R)))
	i = math.Acos(h[2] / norm(h))
	Ω = math.Acos(N[0] / norm(N))

	if N[1] < 0 { // Quadrant check.
		Ω = 2*math.Pi - Ω
	}
	ω = math.Acos(dot(N, eVec) / (norm(N) * e))
	if eVec[2] < 0 { // Quadrant check
		ω = 2*math.Pi - ω
	}
	ν = math.Acos(dot(eVec, o.R) / (e * norm(o.R)))
	if dot(o.R, o.V) < 0 {
		ν = 2*math.Pi - ν
	}

	return
}

// String implements the stringer interface.
func (o *Orbit) String() string {
	a, e, i, ω, Ω, ν := o.GetOE()
	return fmt.Sprintf("a=%0.5f e=%0.5f i=%0.5f ω=%0.5f Ω=%0.5f ν=%0.5f", a, e, i, ω, Ω, ν)
}

// ToXCentric converts this orbit the provided celestial object centric equivalent.
// Panics if the vehicle is not within the SOI of the object.
// Panics if already in this frame.
func (o *Orbit) ToXCentric(b CelestialObject, dt time.Time) {
	if o.Origin == b {
		panic(fmt.Errorf("already in orbit around %s", b.Name))
	}
	fmt.Printf("Switching to orbit around %s\n", b.Name)
	if b.SOI == -1 {
		// Switch to heliocentric
		// Get planet ecliptic coordinates.
		relPos, relVel := o.Origin.HelioOrbit(dt)
		// Switch to ecliptic coordinates.
		o.R = MxV33(R1(Deg2rad(o.Origin.tilt)), o.R)
		o.V = MxV33(R1(Deg2rad(o.Origin.tilt)), o.V)
		// Switch frame origin.
		for i := 0; i < 3; i++ {
			o.R[i] += relPos[i]
			o.V[i] += relVel[i]
		}
	} else {
		// Switch to planet centric
		// Get planet ecliptic coordinates.
		relPos, relVel := b.HelioOrbit(dt)
		// Update frame origin.
		for i := 0; i < 3; i++ {
			o.R[i] -= relPos[i]
			o.V[i] -= relVel[i]
		}
		// Switch from ecliptic coordinates to equatorial.
		o.R = MxV33(R1(-Deg2rad(b.tilt)), o.R)
		o.V = MxV33(R1(-Deg2rad(b.tilt)), o.V)
	}
	o.Origin = b // Don't forget to switch origin
}

// NewOrbitFromOE creates an orbit from the orbital elements.
func NewOrbitFromOE(a, e, i, ω, Ω, ν float64, c CelestialObject) *Orbit {
	// Check for edge cases which are not supported.
	if ν < 1e-10 {
		panic("ν ~= 0 is not supported")
	}
	if e < 0 || e > 1 {
		panic("only circular and elliptical orbits supported")
	}
	μ := c.μ
	p := a * (1.0 - math.Pow(e, 2)) // semi-parameter
	R, V := make([]float64, 3), make([]float64, 3)
	// Compute R and V in the perifocal frame (PQW).
	R[0] = p * math.Cos(ν) / (1 + e*math.Cos(ν))
	R[1] = p * math.Sin(ν) / (1 + e*math.Cos(ν))
	R[2] = 0
	V[0] = -math.Sqrt(μ/p) * math.Sin(ν)
	V[1] = math.Sqrt(μ/p) * (e + math.Cos(ν))
	V[2] = 0
	// Compute ECI rotation.
	R = PQW2ECI(i, ω, Ω, R)
	V = PQW2ECI(i, ω, Ω, V)
	return &Orbit{R, V, c}
}

// NewOrbit returns orbital elements from the R and V vectors. Needed for prop
func NewOrbit(R, V []float64, c CelestialObject) *Orbit {
	return &Orbit{R, V, c}
}

// Helper functions go here.

// Radii2ae returns the semi major axis and the eccentricty from the radii.
func Radii2ae(rA, rP float64) (a, e float64) {
	if rA < rP {
		panic("periapsis cannot be greater than apoapsis")
	}
	a = (rP + rA) / 2
	e = (rA - rP) / (rA + rP)
	return
}
