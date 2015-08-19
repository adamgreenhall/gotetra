package analyze

import (
	intr "github.com/phil-mansfield/gotetra/math/interpolate"
)

var (
	kernels = make(map[int]*intr.Kernel)
	derivKernels = make(map[int]*intr.Kernel)
)

type smoothParams struct {
	vals, derivs []float64
}

type internalSmoothOption func(*smoothParams)
// SmoothOption is an abstract data type which allows for the customization of
// calls to Smooth without cluttering the call signature in the common case.
// This works similarly to kwargs in other languages.
type SmoothOption internalSmoothOption

func (p *smoothParams) loadOptions(opts []SmoothOption) {
	for _, opt := range opts { opt(p) }
}

// Vals supplies Smooth with a slice which smoothed values can be written to.
func Vals(vals []float64) SmoothOption {
	return func(p *smoothParams) { p.vals = vals }
}

// Derivs supplies Smooth with a slice which smoothed derivatives can be
// written to.
func Derivs(derivs []float64) SmoothOption {
	return func(p *smoothParams) { p.derivs = derivs }
}

// Smooth returns a smoothed 1D series as well as the derivative of that series
// using a Savitzky-Golay filter of the given size. It also takes optional
// arguments which allow the smoothing to be done in-place.
func Smooth(
	xs, ys []float64, window int, opts ...SmoothOption,
) (vals, derivs []float64, ok bool) {
	if len(xs) != len(ys) {
		panic("Length of xs and ys must be the same.")
	} else if len(xs) <= window {
		return nil, nil, false
	}
	
	p := new(smoothParams)
	p.loadOptions(opts)
	vals = p.vals
	derivs = p.derivs
	if vals == nil { vals = make([]float64, len(xs)) }
	if derivs == nil { derivs = make([]float64, len(xs)) }

	dx := (xs[0] - xs[len(xs) - 1])/ float64(len(xs) - 1)
	k, kd := getSmoothingKernel(window, dx)

	k.ConvolveAt(ys, intr.Extension, vals)
	kd.ConvolveAt(ys, intr.Extension, derivs)
	return vals, derivs, true
}

func getSmoothingKernel(window int, dx float64) (k, kd *intr.Kernel) {
	k, ok := kernels[window]
	kd, _ = derivKernels[window]
	if ok { return k, kd }
	k = intr.NewSavGolKernel(4, window)
	kd = intr.NewSavGolDerivKernel(dx, 1, 4, window)
	kernels[window] = k
	derivKernels[window] = kd

	return k, kd
}
