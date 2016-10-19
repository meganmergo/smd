package main

import (
	"dynamics"
	"log"
	"math"
	"time"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	/* Building spacecraft */
	eps := dynamics.NewUnlimitedEPS()
	thrusters := []dynamics.Thruster{&dynamics.HPHET12k5{}, &dynamics.HPHET12k5{}}
	//thrusters := []dynamics.Thruster{&dynamics.PPS1350{}, &dynamics.PPS1350{}}
	waypoints := []dynamics.Waypoint{dynamics.NewOutwardSpiral(dynamics.Earth, nil)}
	dryMass := 1000.0
	fuelMass := 500.0
	sc := &dynamics.Spacecraft{Name: "IT1", DryMass: dryMass, FuelMass: fuelMass, EPS: eps, Thrusters: thrusters, Cargo: []*dynamics.Cargo{}, WayPoints: waypoints}

	/* Building propagation */
	start := time.Now()                                   // Propagate starting now for ease.
	end := start.Add(time.Duration(-1) * time.Nanosecond) // Propagate until waypoint reached.
	// Falcon 9 delivers at 24.68 350x250km.
	a, e := dynamics.Radii2ae(350+dynamics.Earth.Radius, 250+dynamics.Earth.Radius)
	i := dynamics.Deg2rad(24.68)
	ω := dynamics.Deg2rad(10) // Made up
	Ω := dynamics.Deg2rad(5)  // Made up
	ν := dynamics.Deg2rad(1)  // I don't care about that guy.
	orbit := dynamics.NewOrbitFromOE(a, e, i, ω, Ω, ν, &dynamics.Earth)
	astro := dynamics.NewAstro(sc, orbit, &start, &end, "../outputdata/propCont")
	// Start propagation.
	log.Printf("Depart from: %s\n", orbit.String())
	astro.Propagate()
	log.Printf("Arrived at: %s\n", orbit.String())
}