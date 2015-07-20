/*package geom contains routines for computing geometric quantities.

Contains implementations of algorithms described in Platis & Theoharis, 2015
as well as Schneider & Eberly.
*/
package geom

import (
	"math"
)

// Vec is a three dimensional vector. (Duh!)
type Vec [3]float32

// PluckerVec represents a ray. If P and L are the position of the ray's 
// origin and the unit vector representing its direction, respectively, then
// U = L and V = L cross P.
type PluckerVec struct {
	U, V Vec
}

// AnchoredPluckerVec is a Plucker vector which also saves the position of
// the ray's origin.
type AnchoredPluckerVec struct {
	PluckerVec
	P Vec
}

// Init initializes a Plucker vector given a ray origin, P, and a unit
// direction vector, L.
func (p *PluckerVec) Init(P, L *Vec) {
	p.U = *L

	p.V[0] = -P[1]*L[2] + P[2]*L[1]
    p.V[1] = -P[2]*L[0] + P[0]*L[2]
    p.V[2] = -P[0]*L[1] + P[1]*L[0]
}

// InitFromSegment initialized a Plucker vector which corresponds to a ray
// pointing from the position vector P1 to the position vector P2.
func (p *PluckerVec) InitFromSegment(P1, P2 *Vec) {
	var sum float32
	for i := 0; i < 3; i++ {
		p.U[i] = P2[i] - P1[i]
		sum += p.U[i]*p.U[i]
	}
	sum = float32(math.Sqrt(float64(sum)))
	for i := 0; i < 3; i++ { p.U[i] /= sum }

	p.V[0] = -P1[1]*p.U[2] + P1[2]*p.U[1]
    p.V[1] = -P1[2]*p.U[0] + P1[0]*p.U[2]
    p.V[2] = -P1[0]*p.U[1] + P1[1]*p.U[0]
}

// Init initializes an anchored Plucker vector given a ray origin, P, and a
// unit direction vector, L.
func (ap *AnchoredPluckerVec) Init(P, L *Vec) {
	ap.PluckerVec.Init(P, L)
	ap.P = *P
}

// InitFromSegment initialized a Plucker vector which corresponds to a ray
// pointing from the position vector P1 to the position vector P2.
func (ap *AnchoredPluckerVec) InitFromSegment(P1, P2 *Vec) {
	ap.PluckerVec.InitFromSegment(P1, P2)
	ap.P = *P1
}

// Dot computes the permuted inner product of p1 and p2, i.e.
// p1.U*p2.V + p1.V*p2.U.
func (p1 *PluckerVec) Dot(p2 *PluckerVec, flip bool) float32 {
	var sum float32
	for i := 0; i < 3; i++ {
		sum += p1.U[i]*p2.V[i] + p1.V[i]*p2.U[i]
	}
	if flip {
		return sum
	} else {
		return -sum
	}
}

// Dot computes the permuted inner product of p1 and p2, i.e.
// p1.U*p2.V + p1.V*p2.U and also returns a sign flag of -1, 0, or +1 if
// that product is negative, zero, or positive, respectively.
func (p1 *PluckerVec) SignDot(p2 *PluckerVec, flip bool) (float32, int) {
	dot := p1.Dot(p2, flip)
	if dot == 0 {
		return dot, 0
	} else if dot > 0 {
		return dot, +1
	} else {
		return dot, -1
	}
}

// Tetra is a tetrahedron. (Duh!)
//
// Face ordering is:
// F0(V3, V2, V1)
// F1(V2, V3, V0)
// F2(V1, V0, V3)
// F3(V0, V1, V2)
type Tetra [4]Vec

var tetraIdxs = [4][3]int {
	[3]int{ 3, 2, 1 },
	[3]int{ 2, 3, 0 },
	[3]int{ 1, 0, 3 },
	[3]int{ 0, 1, 2 },
}

// VertexIdx returns the index into the given tetrahedron corresponding to
// the specified face and vertex.
func (_ *Tetra) VertexIdx(face, vertex int) int {
	return tetraIdxs[face][vertex]
}

// Orient arranges tetrahedron points so that all faces point outward for
// dir = +1 and inward for dir = -1.
func (t *Tetra) Orient(dir int) {
	v, w, n := Vec{}, Vec{}, Vec{}
	for i := 0; i < 3; i++ {
		v[i] = t[1][i] - t[0][i]
		w[i] = t[2][i] - t[0][i]
	}
	n[0] = -v[1]*w[2] + v[2]*w[1]
    n[1] = -v[2]*w[0] + v[0]*w[2]
    n[2] = -v[0]*w[1] + v[1]*w[0]

	var dot float32
	for i := 0; i < 3; i++ {
		dot += n[i] * (t[3][i] - t[0][i])
	}

	if (dot < 0 && dir == -1) || (dot > 0 && dir == +1) {
		t[0], t[1] = t[1], t[0]
	}
}

// TetraFaceBary contains information specifying the barycentric coordinates
// of a point on a face of a tetrahedron.
type TetraFaceBary struct {
	w [3]float32
	face int
}

// Distance calculates the distance from an anchored Plucker vector to a point
// in a tetrahedron described by the given unscaled barycentric coordinates.
func (t *Tetra) Distance(ap *AnchoredPluckerVec, bary *TetraFaceBary) float32 {
	// Computes one coordinate of the intersection point from the barycentric
	// coordinates of the intersection, then solves P_intr = P + t * L for t.
	var sum float32
	for i := 0; i < 3; i++ { sum += bary.w[i] }
	u0, u1, u2 := bary.w[0] / sum, bary.w[1] / sum, bary.w[2] / sum
	var dim int
	for dim = 0; dim < 3; dim++ {
		if ap.U[dim] != 0 { break }
	}

	p0 := t[t.VertexIdx(bary.face, 0)][dim]
	p1 := t[t.VertexIdx(bary.face, 1)][dim]
	p2 := t[t.VertexIdx(bary.face, 2)][dim]

	return ((u0*p0 + u1*p1 + u2*p2) - ap.P[dim]) / ap.U[dim]
}

// PluckerTetra is a tetrahedron represented by the Plucker vectors that make
// up its edges. It is used for Platis & Theoharis's interseciton detection
// algorithm.
//
// The raw ordering of edges is
// {0-1, 0-2, 0-3, 1-2, 1-3, 2-3}
type PluckerTetra [6]PluckerVec

var pluckerTetraEdges = [4][3]int{
	[3]int{ 5, 3, 4 }, // {3-2, 2-1, 1-3}
	[3]int{ 5, 2, 1 }, // {2-3, 3-0, 0-2}
	[3]int{ 0, 2, 4 }, // {1-0, 0-3, 3-1}
	[3]int{ 0, 3, 1 }, // {0-1, 1-2, 2-0}
}

var pluckerTetraFlips = [4][3]bool{
	[3]bool{ true,  true,  false }, // {3-2, 2-1, 1-3}
	[3]bool{ false, true,  false }, // {2-3, 3-0, 0-2}
	[3]bool{ true,  false, true  }, // {1-0, 0-3, 3-1}
	[3]bool{ false, false, true  }, // {0-1, 1-2, 2-0}
}

// Init initializes a Plucker Tetrahedron from a normal Tetrahedron.
func (pt *PluckerTetra) Init(t *Tetra) {
	pt[0].InitFromSegment(&t[0], &t[1])
	pt[1].InitFromSegment(&t[0], &t[2])
	pt[2].InitFromSegment(&t[0], &t[3])
	pt[3].InitFromSegment(&t[1], &t[2])
	pt[4].InitFromSegment(&t[1], &t[3])
	pt[5].InitFromSegment(&t[2], &t[3])
}

// EdgeIdx returns the index into pt which corresponds to the requested
// face and edge. A flag is also returned indicating whether the vector stored
// in pt needs to be flipped when doing operations on that face.
func (_ *PluckerTetra) EdgeIdx(face, edge int) (idx int, flip bool) {
	idx = pluckerTetraEdges[face][edge]
	flip = pluckerTetraFlips[face][edge]
	return idx, flip
}
