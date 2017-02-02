package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/ChristopherRabotin/smd"
)

func norm(v []float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func main() {
	CheckEnvVars()
	runtime.GOMAXPROCS(3)

	start := time.Date(2016, 3, 14, 9, 31, 0, 0, time.UTC) // ExoMars launch date.
	//end := start.Add(time.Duration(-1) * time.Nanosecond)  // Propagate until waypoint reached.
	end := time.Date(2018, 07, 13, 0, 0, 0, 0, time.UTC)

	/* Let's propagate out of Mars at a guessed date of 7 months after launch date from Earth.
	Note that we only output the CSV because we don't need to visualize this.
	*/
	endM := end.Add(time.Duration(4 * 30.5 * 24))
	scMars := SpacecraftFromMars("IM")
	scMars.LogInfo()
	astroM := smd.NewMission(scMars, InitialMarsOrbit(), end, endM, smd.GaussianVOP, smd.Perturbations{}, smd.ExportConfig{Filename: "IM", AsCSV: false, Cosmo: false, Timestamp: false})
	astroM.Propagate()
	// Convert the position to heliocentric.
	astroM.Orbit.ToXCentric(smd.Sun, astroM.CurrentDT)
	target := astroM.Orbit
	//	target := smd.NewOrbitFromOE(226090298.679, 0.088, 26.195, 3.516, 326.494, 278.358, smd.Sun)
	//	fmt.Printf("target orbit: %s\n", target)
	sc := SpacecraftFromEarth("IE", *target)
	sc.LogInfo()
	astro := smd.NewMission(sc, InitialEarthOrbit(), start, end, smd.GaussianVOP, smd.Perturbations{}, smd.ExportConfig{Filename: "IE", AsCSV: true, Cosmo: true, Timestamp: false})
	astro.Propagate()

}

// CheckEnvVars checks that all the environment variables required are set, without checking their value. It will panic if one is missing.
func CheckEnvVars() {
	envvars := []string{"VSOP87", "DATAOUT"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty,", envvar))
		}
	}
}
