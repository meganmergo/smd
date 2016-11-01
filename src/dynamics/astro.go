package dynamics

import (
	"integrator"
	"math"
	"os"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
)

const (
	stepSize = 1.0
)

var logger kitlog.Logger
var wg sync.WaitGroup

/* Handles the astrodynamical propagations. */

// Astrocodile is an orbit propagator.
// It's a play on words from STK's Atrogrator.
type Astrocodile struct {
	Vehicle   *Spacecraft // As pointer because SC may be altered during propagation.
	Orbit     *Orbit      // As pointer because the orbit changes during propagation.
	StartDT   time.Time
	EndDT     time.Time
	CurrentDT time.Time
	StopChan  chan (bool)
	histChan  chan<- (AstroState)
	initV     float64
}

// NewAstro returns a new Astrocodile instance from the position and velocity vectors.
func NewAstro(s *Spacecraft, o *Orbit, start, end time.Time, filepath string) (*Astrocodile, *sync.WaitGroup) {
	// If no filepath is provided, then no output will be written.
	var histChan chan (AstroState)
	if filepath != "" {
		histChan = make(chan (AstroState), 1000) // a 1k entry buffer
		wg.Add(1)
		go func() {
			defer wg.Done()
			StreamStates(filepath, histChan, false)
		}()
	} else {
		histChan = nil
	}
	// Must switch to UTC as all ephemeris data is in UTC.
	if start.Location() != time.UTC {
		start = start.UTC()
	}
	if end.Location() != time.UTC {
		end = end.UTC()
	}

	a := &Astrocodile{s, o, start, end, start, make(chan (bool), 1), histChan, norm(o.V)}
	// Write the first data point.
	if histChan != nil {
		histChan <- AstroState{a.CurrentDT, *s, *o}
	}

	logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	logger = kitlog.NewContext(logger).With("spacecraft", s.Name).With("time", a.CurrentDT)
	if end.Before(start) {
		logger.Log("level", "warning", "subsys", "astro", "message", "no end date")
	}

	return a, &wg
}

// LogStatus returns the status of the propagation and vehicle.
// TODO: Support center of orbit change.
func (a *Astrocodile) LogStatus() {
	logger.Log("level", "info", "subsys", "prop", "Δv", norm(a.Orbit.V)-a.initV, "fuel", a.Vehicle.FuelMass)
}

// Propagate starts the propagation.
func (a *Astrocodile) Propagate() {
	// Add a ticker status report based on the duration of the simulation.
	var tickDuration time.Duration
	if a.EndDT.After(a.StartDT) {
		tickDuration = time.Duration(a.EndDT.Sub(a.StartDT).Hours()*0.01) * time.Second
	} else {
		tickDuration = 1 * time.Minute
	}
	if tickDuration > 0 {
		logger.Log("level", "notice", "subsys", "astro", "reportPeriod", tickDuration, "orbit", a.Orbit)
		a.LogStatus()
		ticker := time.NewTicker(tickDuration)
		go func() {
			for _ = range ticker.C {
				a.LogStatus()
			}
		}()
	} else {
		// Happens only during tests.
		logger.Log("level", "notice", "subsys", "astro", "orbit", a.Orbit)
	}
	integrator.NewRK4(0, stepSize, a).Solve() // Blocking.
	a.LogStatus()
	logger.Log("level", "notice", "subsys", "astro", "orbit", a.Orbit)
	if a.Vehicle.FuelMass < 0 {
		logger.Log("level", "critical", "subsys", "prop", "fuel(kg)", a.Vehicle.FuelMass)
	}
}

// Stop implements the stop call of the integrator.
func (a *Astrocodile) Stop(i uint64) bool {
	select {
	// TODO: Change this to a call to Spacecraft given the orbit information.
	case <-a.StopChan:
		if a.histChan != nil {
			close(a.histChan)
		}
		return true // Stop because there is a request to stop.
	default:
		a.CurrentDT = a.CurrentDT.Add(time.Duration(stepSize) * time.Second)
		if a.EndDT.Before(a.StartDT) {
			// Check if any waypoint still needs to be reached.
			for _, wp := range a.Vehicle.WayPoints {
				if !wp.Cleared() {
					return false
				}
			}
			if a.histChan != nil {
				close(a.histChan)
			}
			return true
		}
		if a.CurrentDT.Sub(a.EndDT).Nanoseconds() > 0 {
			if a.histChan != nil {
				close(a.histChan)
			}
			return true // Stop, we've reached the end of the simulation.
		}
	}
	return false
}

// GetState returns the state for the integrator.
func (a *Astrocodile) GetState() (s []float64) {
	s = make([]float64, 7)
	for i := 0; i < 3; i++ {
		s[i] = a.Orbit.R[i]
		s[i+3] = a.Orbit.V[i]
	}
	s[6] = a.Vehicle.FuelMass
	return
}

// SetState sets the updated state.
func (a *Astrocodile) SetState(i uint64, s []float64) {
	if a.histChan != nil {
		a.histChan <- AstroState{a.CurrentDT, *a.Vehicle, *a.Orbit}
	}
	a.Orbit.R = []float64{s[0], s[1], s[2]}
	a.Orbit.V = []float64{s[3], s[4], s[5]}
	// Let's execute any function which is in the queue of this time step.
	for _, f := range a.Vehicle.FuncQ {
		if f == nil {
			continue
		}
		f()
	}
	a.Vehicle.FuncQ = make([]func(), 5) // Clear the queue.
	//if a.Orbit.Origin == Sun && norm(a.Orbit.R) < 1e7 {
	// Must manually fix, not sure this will work because of the integrator which calls
	// the Func function with its own values, each need to be fixed prior to the propagation.
	//	a.Orbit.ToXCentric(Sun, a.CurrentDT)
	//	}
	// I think I need to update Func in order to change the R and V on the fly.
	// However, I can't just use the ToXCentric because the orbit frame is already switched.
	// That means I need to update the orbit ref only after all the calls to the function.
	// Note completely sure how to handle that.
	if norm(a.Orbit.R) < 0 {
		panic("negative distance to body")
	}
	if a.Vehicle.FuelMass > 0 && s[6] <= 0 {
		logger.Log("level", "critical", "subsys", "prop", "fuel(kg)", s[6])
	}
	a.Vehicle.FuelMass = s[6]
}

// Func is the integration function.
func (a *Astrocodile) Func(t float64, f []float64) (fDot []float64) {
	fDot = make([]float64, 7) // init return vector
	radius := norm([]float64{f[0], f[1], f[2]})
	// Let's add the thrust to increase the magnitude of the velocity.
	Δv, usedFuel := a.Vehicle.Accelerate(a.CurrentDT, a.Orbit)
	twoBodyVelocity := -a.Orbit.Origin.μ / math.Pow(radius, 3)
	for i := 0; i < 3; i++ {
		// The first three components are the velocity.
		fDot[i] = f[i+3]
		// The following three are the instantenous acceleration.
		fDot[i+3] = f[i]*twoBodyVelocity + Δv[i]
	}
	fDot[6] = -usedFuel
	return
}

// AstroState stores propagated state.
type AstroState struct {
	dt    time.Time
	sc    Spacecraft
	orbit Orbit
}
