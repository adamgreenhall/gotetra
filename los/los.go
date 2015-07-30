package los

import (
	"github.com/phil-mansfield/gotetra/render/io"
	rGeom "github.com/phil-mansfield/gotetra/render/geom"
	"github.com/phil-mansfield/gotetra/los/geom"
)

// CountAll computes profiles for all the given halos which count the number
// of tetrahedra overlapping points at a given radius.
func CountAll(
	hs []HaloProfiles, ts []geom.Tetra, ss []geom.Sphere,
) {
	for hi := range hs {
		h := &hs[hi]
		for ti := range ts {
			t := &ts[ti]
			s := &ss[ti]

			// This can be sped up significantly, if need be.
			if h.SphereIntersect(s) { h.Count(t) }
		}
	}
}

func DensityAll(
	hs []HaloProfiles, ts []geom.Tetra, ss []geom.Sphere, rhos []float64,
) {
	for hi := range hs {
		h := &hs[hi]
		for ti := range ts {
			t := &ts[ti]
			s := &ss[ti]

			if h.SphereIntersect(s) { h.Density(t, rhos[ti]) }
		}
	}
}

func coords(idx, cells int64) (x, y, z int64) {
    x = idx % cells
    y = (idx % (cells * cells)) / cells
    z = idx / (cells * cells)
    return x, y, z
}

func index(x, y, z, cells int64) int64 {
    return x + y * cells + z * cells * cells
}

func unpackTetra(idxs *rGeom.TetraIdxs, xs []rGeom.Vec, t *geom.Tetra) {
    for i := 0; i < 4; i++ {
		t[i] = geom.Vec(xs[idxs[i]])
    }
}

// UnpackTetrahedra converts the raw position data in a sheet segment into
// tetrahedra.
func UnpackTetrahedra(
	xs []rGeom.Vec, hd *io.SheetHeader, tsBuf []geom.Tetra,
) {
	n := hd.SegmentWidth*hd.SegmentWidth*hd.SegmentWidth
	idxBuf := new(rGeom.TetraIdxs)
	for writeIdx := int64(0); writeIdx < n; writeIdx++ {
		x, y, z := coords(writeIdx, hd.SegmentWidth)
		readIdx := index(x, y, z, hd.SegmentWidth)

		for dir := int64(0); dir < 6; dir++ {
			tIdx := 6 * writeIdx + dir
			idxBuf.Init(readIdx, hd.GridWidth + 1, 1, int(dir))
			unpackTetra(idxBuf, xs, &tsBuf[tIdx])
			tsBuf[tIdx].Orient(+1)
		}
	}

}

func TetraDensity(hd *io.SheetHeader, ts []geom.Tetra, rhos []float64) {
	tw := hd.TotalWidth
	unitTetRho := tw*tw*tw / (6 * float64(hd.Count))
	for i := range ts {
		rhos[i] = unitTetRho / ts[i].Volume()
	}
}

func WrapHalo(hps []HaloProfiles, hd *io.SheetHeader) {
	tw := float32(hd.TotalWidth)
	for i := range hps {
		h := &hps[i]
		for j := 0; j < 3; j++ {
			if h.C[j] + h.R < hd.Origin[j] { h.C[j] += tw }
		}
	}
}

func WrapXs(xs []rGeom.Vec, hd *io.SheetHeader) {
	tw := float32(hd.TotalWidth)
	for i := range xs {
		for j := 0; j < 3; j++ {
			if xs[i][j] < hd.Origin[j] { xs[i][j] += tw }
		}
	}
}