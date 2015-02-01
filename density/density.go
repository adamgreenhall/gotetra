/*package density interpolates sequences of particle positions onto a density
grid.
*/
package density

import (
	"log"
	"math"

	"github.com/phil-mansfield/gotetra/rand"
	"github.com/phil-mansfield/gotetra/geom"
	"github.com/phil-mansfield/gotetra/catalog"
)

// Interpolator creates a grid-based density distribution from seqeunces of
// positions.
type Interpolator interface {
	// Interpolate adds the density distribution implied by points to the
	// density grid used by the Interpolator. Particles should all be within
	// the bounds of the bounding grid and points not within the interpolation
	// grid will be ignored.
	Interpolate(gs []Grid, mass float64, ids []int64, xs []geom.Vec)
}

type Grid struct {
	Rhos []float64
	BoxWidth, CellWidth, CellVolume float64
	G, BG geom.Grid
}

type Cell struct {
	Width, X, Y, Z int
}

type cic struct { }
type ngp struct { }

type pointSelector func(*mcarlo, *geom.CellBounds) int
type PointSelectorFlag int

const (
	Flat PointSelectorFlag = iota
	PropToCells
)

type mcarlo struct {
	subIntr Interpolator
	man *catalog.ParticleManager
	countWidth int64
	steps int

	gen *rand.Generator

	pointSelect pointSelector

	// Buffers
	idxBuf geom.TetraIdxs
	tet geom.Tetra
	randBuf []float64
	vecBuf []geom.Vec
}

type sobol struct {
	subIntr Interpolator
	man *catalog.ParticleManager
	countWidth int64

	idxBuf geom.TetraIdxs
	tet geom.Tetra
	xs, ys, zs []float64
	vecBuf []geom.Vec
}

type cellCenter struct {
	man *catalog.ParticleManager
	countWidth int64
	idxBuf geom.TetraIdxs
	tet geom.Tetra
	vBuf geom.Vec
}

func NewGrid(boxWidth float64, gridWidth int, rhos []float64, c *Cell) *Grid {
	g := &Grid{}
	g.Init(boxWidth, gridWidth, rhos, c)
	return g
}

func (g *Grid) Init(boxWidth float64, gridWidth int, rhos []float64, c *Cell) {
	if len(rhos) != c.Width * c.Width * c.Width {
		panic("Length of rhos doesn't match cell width.")
	}

	g.G.Init(&[3]int{c.X*c.Width, c.Y*c.Width, c.Z*c.Width}, c.Width)
	g.BG.Init(&[3]int{0, 0, 0}, c.Width * gridWidth)
	g.BoxWidth = boxWidth
	g.CellWidth = boxWidth / float64(c.Width * gridWidth)
	g.CellVolume = g.CellWidth * g.CellWidth * g.CellWidth
	g.Rhos = rhos
}

func CloudInCell() Interpolator {
	return &cic{}
}

func NearestGridPoint() Interpolator {
	return &ngp{}
}

func CellCenter(man *catalog.ParticleManager, countWidth int64) Interpolator {
	return &cellCenter{man, countWidth, geom.TetraIdxs{},
		geom.Tetra{}, geom.Vec{}}
}

func PointSelectorFromString(str string) PointSelectorFlag {
	switch str {
	case "Flat":
		return Flat
	case "PropToCells":
		return PropToCells
	}
	log.Fatalf("Unrecognized PointSelector string, '%s'", str)
	panic("Impossible")
}

func flat(intr *mcarlo, cb *geom.CellBounds) int {
	return intr.steps
}

func propToCell(intr *mcarlo, cb *geom.CellBounds) int {
	//vol := intr.tet.Volume()
	//if vol < 3e-4 {
	//	return intr.steps / 5
	//}
	min, _ := intr.tet.MinMaxLeg()
	
	//if rat >= 60 && rat < 600 {
	//	return intr.steps
	//}

	if min >= 0.251 {
		return intr.steps
	}

	return 0
}

func MonteCarlo(man *catalog.ParticleManager, countWidth int64,
	gen *rand.Generator, steps int, flag PointSelectorFlag) Interpolator {

	var pointSelect pointSelector
	switch flag {
	case Flat:
		pointSelect = flat
	case PropToCells:
		pointSelect = propToCell
	}

	return &mcarlo{
		NearestGridPoint(), man, countWidth, steps,
		gen, pointSelect, geom.TetraIdxs{}, geom.Tetra{}, 
		make([]float64, steps * 3), make([]geom.Vec, steps),
	}
}

func SobolSequence(man *catalog.ParticleManager, countWidth int64, steps int) Interpolator {
	seq := rand.NewSobolSequence()
	buf := []float64{0, 0, 0}
	xs := make([]float64, steps)
	ys := make([]float64, steps)
	zs := make([]float64, steps)

	for i := 0; i < steps; i++ {
		seq.NextAt(buf)
		xs[i] = buf[0]
		ys[i] = buf[1]
		zs[i] = buf[2]
	}

	return &sobol{NearestGridPoint(), man, countWidth, geom.TetraIdxs{},
		geom.Tetra{}, xs, ys, zs, make([]geom.Vec, steps)}
}

// Interpolate interpolates a sequence of particles onto a density grid via a
// nearest grid point scheme.
func (intr *ngp) Interpolate(gs []Grid, mass float64, ids []int64, xs []geom.Vec) {
	frac := mass / gs[0].CellVolume
	for _, pt := range xs {
		xp, yp, zp := float64(pt[0]), float64(pt[1]), float64(pt[2])
		xc, yc, zc := cellPoints(xp, yp, zp, gs[0].CellWidth)
		i, j, k := int(xc), int(yc), int(zc)

		for gIdx := range gs {
			g := &gs[gIdx]
			if idx, ok := g.G.IdxCheck(i, j, k); ok {
				g.Rhos[idx] += frac
				continue
			}
		}
	}
}

// Interpolate interpolates a sequence of particles onto a density grid via a
// cloud in cell scheme.
func (intr *cic) Interpolate(gs []Grid, mass float64, ids []int64, xs []geom.Vec) {
	frac := mass / gs[0].CellVolume
	cw, cw2 := gs[0].CellWidth, gs[0].CellWidth / 2
	for _, pt := range xs {
		
		xp, yp, zp := float64(pt[0])-cw2, float64(pt[1])-cw2, float64(pt[2])-cw2
		xc, yc, zc := cellPoints(xp, yp, zp, gs[0].CellWidth)
		dx, dy, dz := (xp / cw)-xc, (yp / cw)-yc, (zp / cw)-zc
		tx, ty, tz := 1-dx, 1-dy, 1-dz

		i0, i1 := gs[0].nbrs(int(xc))
		j0, j1 := gs[0].nbrs(int(yc))
		k0, k1 := gs[0].nbrs(int(zc))

		over000 := tx*ty*tz*frac
		over100 := dx*ty*tz*frac
		over010 := tx*dy*tz*frac
		over110 := dx*dy*tz*frac
		over001 := tx*ty*dz*frac
		over101 := dx*ty*dz*frac
		over011 := tx*dy*dz*frac
		over111 := dx*dy*dz*frac

		for gIdx := range gs {
			g := &gs[gIdx]

			g.incr(i0, j0, k0, over000)
			g.incr(i1, j0, k0, over100)
			g.incr(i0, j1, k0, over010)
			g.incr(i1, j1, k0, over110)
			g.incr(i0, j0, k1, over001)
			g.incr(i1, j0, k1, over101)
			g.incr(i0, j1, k1, over011)
			g.incr(i1, j1, k1, over111)
		}
	}
}

func (g *Grid) nbrs(i int) (i0, i1 int) {
	if i == -1 {
		return g.BG.Width - 1, 0
	}
	if i+1 == g.BG.Width {
		return i, 0
	}
	return i, i + 1
}

func (g *Grid) incr(i, j, k int, frac float64) {
	if idx, ok := g.G.IdxCheck(i, j, k); ok {
		g.Rhos[idx] += frac
	}
}

func cellPoints(x, y, z, cw float64) (xc, yc, zc float64) {
	return math.Floor(x / cw), math.Floor(y / cw), math.Floor(z / cw)
}

func (intr *cellCenter) Interpolate(gs []Grid, mass float64, ids []int64, xs []geom.Vec) {
	cb := &geom.CellBounds{}

	misses, hits := 0, 0

	for _, id := range ids {
		for dir := 0; dir < 6; dir++ {
			intr.idxBuf.Init(id, intr.countWidth, 1, dir)

			p0 := intr.man.Get(intr.idxBuf[0])
			p1 := intr.man.Get(intr.idxBuf[1])
			p2 := intr.man.Get(intr.idxBuf[2])
			p3 := intr.man.Get(intr.idxBuf[3])

			if p0 == nil || p1 == nil || p2 == nil || p3 == nil {
				log.Printf("Tetrahedron [%v %v %v %v] not in manager.\n",
					p0, p1, p2, p3)
				continue
			}

			intr.tet.Init(&p0.Xs, &p1.Xs, &p2.Xs, &p3.Xs, gs[0].BoxWidth)
			intr.tet.CellBoundsAt(gs[0].CellWidth, cb)

			for i := range gs {
				if gs[i].G.Intersect(cb, &gs[i].BG) {
					dm, dh := intr.intrTetra(mass / 6.0, &gs[i], cb)
					misses += dm
					hits += dh
				}
			}
		}
	}
}

func (intr *cellCenter) intrTetra(mass float64, g *Grid, cb *geom.CellBounds) (int, int) {
	minX := maxInt(cb.Min[0], g.G.Origin[0])
	maxX := minInt(cb.Max[0], g.G.Origin[0] + g.G.Width - 1)
	minY := maxInt(cb.Min[1], g.G.Origin[1])
	maxY := minInt(cb.Max[1], g.G.Origin[1] + g.G.Width - 1)
	minZ := maxInt(cb.Min[2], g.G.Origin[2])
	maxZ := minInt(cb.Max[2], g.G.Origin[2] + g.G.Width - 1)

	frac := mass * g.CellVolume / intr.tet.Volume()

	misses, hits := 0, 0

	for z := minZ; z <= maxZ; z++ {
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				xIdx, yIdx, zIdx := g.BG.Wrap(x, y, z)
				intr.vBuf[0] = float32((float64(xIdx) + 0.5) * g.CellWidth)
				intr.vBuf[1] = float32((float64(yIdx) + 0.5) * g.CellWidth)
				intr.vBuf[2] = float32((float64(zIdx) + 0.5) * g.CellWidth)

				if intr.tet.Contains(&intr.vBuf) {
					idx := g.G.Idx(xIdx, yIdx, zIdx)
					g.Rhos[idx] += frac
					hits++
				} else {
					misses++
				}
			}
		}
	}

	return misses, hits
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (intr *mcarlo) Interpolate(gs []Grid, mass float64, ids []int64, xs []geom.Vec) {
	intersectGs := make([]Grid, len(gs))
	cb := &geom.CellBounds{}

	for _, id := range ids {
		for dir := 0; dir < 6; dir++ {
			intr.idxBuf.Init(id, intr.countWidth, 1, dir)
			
			p0 := intr.man.Get(intr.idxBuf[0])
			p1 := intr.man.Get(intr.idxBuf[1])
			p2 := intr.man.Get(intr.idxBuf[2])
			p3 := intr.man.Get(intr.idxBuf[3])

			if p0 == nil || p1 == nil || p2 == nil || p3 == nil {
				log.Printf("Tetrahedron [%v %v %v %v] not in manager.\n",
					p0, p1, p2, p3)

				continue
			}

			intr.tet.Init(&p0.Xs, &p1.Xs, &p2.Xs, &p3.Xs, gs[0].BoxWidth)
			intr.tet.CellBoundsAt(gs[0].CellWidth, cb)

			pts := intr.pointSelect(intr, cb)
			if pts == 0 { continue }

			ptMass := mass / float64(pts) / 6.0

			intr.tet.RandomSample(intr.gen, intr.randBuf[0: 3*pts],
				intr.vecBuf[0: pts])

			intersectNum := 0
			for i := range gs {
				if gs[i].G.Intersect(cb, &gs[i].BG) {
					intersectGs[intersectNum] = gs[i]
					intersectNum++
				}
			}

			intr.subIntr.Interpolate(intersectGs[0: intersectNum],
				ptMass, nil, intr.vecBuf[0: pts])
		}
	}
}

func (intr *sobol) Interpolate(gs []Grid, mass float64, ids []int64, xs []geom.Vec) {
	ptMass := mass / float64(len(intr.xs)) / 6.0
	intersectGs := make([]Grid, len(gs))
	cb := &geom.CellBounds{}

	for _, id := range ids {
		for dir := 0; dir < 6; dir++ {
			intr.idxBuf.Init(id, intr.countWidth, 1, dir)
			
			p0 := intr.man.Get(intr.idxBuf[0])
			p1 := intr.man.Get(intr.idxBuf[1])
			p2 := intr.man.Get(intr.idxBuf[2])
			p3 := intr.man.Get(intr.idxBuf[3])

			if p0 == nil || p1 == nil || p2 == nil || p3 == nil {
				log.Printf("Tetrahedron [%v %v %v %v] not in manager.\n",
					p0, p1, p2, p3)
				continue
			}

			intr.tet.Init(&p0.Xs, &p1.Xs, &p2.Xs, &p3.Xs, gs[0].BoxWidth)
			intr.tet.Distribute(intr.xs, intr.ys, intr.zs, intr.vecBuf)
			intr.tet.CellBoundsAt(gs[0].CellWidth, cb)

			intersectNum := 0
			for i := range gs {
				if gs[i].G.Intersect(cb, &gs[i].BG) {
					intersectGs[intersectNum] = gs[i]
					intersectNum++
				}
			}

			intr.subIntr.Interpolate(intersectGs[0: intersectNum],
				ptMass, nil, intr.vecBuf)
		}
	}
}
