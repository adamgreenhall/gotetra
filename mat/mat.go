package mat

import (
	"math"
)

type Matrix struct {
	Vals []float64
	Width, Height int
}

type LUFactors struct {
	lu Matrix
	pivot []int
	d float64
}

func NewMatrix(vals []float64, width, height int) *Matrix {
	if width <= 0 {
		panic("width must be positive.")
	} else if height <= 0 {
		panic("height must be positive.")
	} else if width * height != len(vals) {
		panic("height * width must equal len(vals).")
	}

	return &Matrix{Vals: vals, Width: width, Height: height}
}

func NewLUFactors(n int) *LUFactors {
	luf := new(LUFactors)

	luf.lu.Vals, luf.lu.Width, luf.lu.Height = make([]float64, n*n), n, n
	luf.pivot = make([]int, n)
	luf.d = 1

	return luf
}

func (m *Matrix) LU() *LUFactors {
	if m.Width != m.Height { panic("m is non-square.") }

	lu := NewLUFactors(m.Width)
	m.LUFactorsAt(lu)
	return lu
}

func (m *Matrix) LUFactorsAt(luf *LUFactors) {
	if luf.lu.Width != m.Width || luf.lu.Height != m.Height {
		panic("luf has different dimenstions than m.")
	}

	n := m.Width
	for i := 0; i < n; i++ { luf.pivot[i] = i }
	lu := luf.lu.Vals
	mat := m.Vals

	// Maintained for determinant calculations.
	luf.d = 1

	// Crout's algorithm.
	copy(lu, m.Vals)

	// Swap rows.
	for k := 0; k < n; k++ {
		maxRow := findMaxRow(n, mat, k)
		luf.pivot[k], luf.pivot[maxRow] = luf.pivot[maxRow], luf.pivot[k]

		if k != maxRow {
			swapRows(k, maxRow, n, lu)
			luf.d = -luf.d
		}
	}

	// This nonsense.
	for k := 0; k < n; k++ {
		kOffset := k*n
		for i := k + 1; i < n; i++ {
			iOffset := i*n
			lu[iOffset + k] /= lu[kOffset + k]
			tmp := lu[iOffset + k]
			for j := k + 1; j < n; j++ {
				lu[iOffset + j] -= tmp * lu[kOffset + j]
			}
		}
	}
}

// Finds the index of the row containing the maximum value in the column.
// Ignores the values above the point m_col,col since those have already been
// swapped.
func findMaxRow(n int, m []float64, col int) int {
	max, maxRow := -1.0, 0
	
	for i := col; i < n; i++ {
		val := math.Abs(m[i*n + col])
		if val > max {
			max = val
			maxRow = i
		}
	}
	return maxRow
}

func swapRows(i1, i2, n int, lu []float64) {
	i1Offset, i2Offset := n*i1, n*i2
	for j := 0; j < n; j++ {
		idx1, idx2 := i1Offset + j, i2Offset + j
		lu[idx1], lu[idx2] = lu[idx2], lu[idx1]
	}
}

// SolveVector solves M * xs = bs for xs.
//
// bs and xs may poin to the same physical memory.
func (luf *LUFactors) SolveVector(bs, xs []float64) {
	n := luf.lu.Width
	if n != len(bs) {
		panic("len(b) != luf.Width")
	} else if n != len(xs) {
		panic("len(x) != luf.Width")
	}

	// A x = b -> (L U) x = b -> L (U x) = b -> L y = b
	ys := xs
	if &bs[0] == &ys[0] {
		bs = make([]float64, n)
		copy(bs, ys)
	}

	// Solve L * y = b for y.
	forwardSubst(n, luf.pivot, luf.lu.Vals, bs, ys)
	// Solve U * x = y for x.
	backSubst(n, luf.lu.Vals, ys, xs)
}

// Solves L * y = b for y.
// y_i = (b_i - sum_j=0^i-1 (alpha_ij y_j)) / alpha_ij
func forwardSubst(n int, pivot []int, lu, bs, ys []float64) {
	for i := 0; i < n; i++ {
		ys[pivot[i]] = bs[i]
	}
	for i := 0; i < n; i++ {
		sum := 0.0
		for j := 0; j < i; j++ {
			sum += lu[i*n + j] * ys[j]
		}
		ys[i] = (ys[i] - sum)
	}
}

// Solves U * x = y for x.
// x_i = (y_i - sum_j=i+^N-1 (beta_ij x_j)) / beta_ii
func backSubst(n int, lu, ys, xs []float64) {
	for i := n - 1; i >= 0; i-- {
		sum := 0.0
		for j := i + 1; j < n; j++ {
			sum += lu[i*n + j] * xs[j]
		}
		xs[i] = (ys[i] - sum) / lu[i*n + i]
	}
}

// SolveMatrix solves the equation m * x = b.
// 
// x and b may point to the same physical memory.
func (luf *LUFactors) SolveMatrix(b, x *Matrix) {
	xs := x.Vals
	n := luf.lu.Width

	if b.Width != b.Height {
		panic("b matrix is non-square.")
	} else if x.Width != x.Height {
		panic("x matrix is non-square.") 
	} else if n != b.Width {
		panic("b matrix different size than m matrix.")
	} else if n != x.Width {
		panic("x matrix different size than m matrix.")
	}

	col := make([]float64, n)

	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			col[i] = xs[i*n + j]
		}
		luf.SolveVector(col, col)
		for i := 0; i < n; i++ {
			xs[i*n + j] = col[i]
		}
	}
}

func (luf *LUFactors) Invert(out *Matrix) {
	n := luf.lu.Width
	if out.Width != out.Height {
		panic("out matrix is non-square.")
	} else if n != out.Width {
		panic("out matrix different size than m matrix.")
	}

	for i := range out.Vals {
		out.Vals[i] = 0
	}
	for i := 0; i < n; i++ {
		out.Vals[i*n + i] = 1
	}

	luf.SolveMatrix(out, out)
}

func (luf *LUFactors) Determinant() float64 {
	d := luf.d
	lu := luf.lu.Vals
	n := luf.lu.Width

	for i := 0; i < luf.lu.Width; i++ {
		d *= lu[i*n + i]
	}
	return d
}
