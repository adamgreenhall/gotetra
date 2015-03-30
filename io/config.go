package io

import (
	"fmt"
	"strings"

	"code.google.com/p/gcfg"
)


const (
	ExampleConvertSnapshotFile = `[ConvertSnapshot]

#######################
# Required Parameters #
#######################

Input = path/to/input/dir
Output = path/to/output/dir

InputFormat = LGadget-2

Cells = 8 # It's unlikely that you will want to change this.

#######################
# Optional Parameters #
#######################

# ProfileFile = pprof.out
# LogFile = log.out

# IteratedInput = path/to/iterated/input_with_single_%d_format
# IteratedOutput = path/to/iterated/input_with_single_%d_format

# IterationStart = 0 
# IterationEnd = 100
# Inclusive. If IterationEnd isn't set, folders will be iterated through until
# an invalid one is found.`
	ExampleDensityFile = `[Density]

######################
# RequiredParameters #
######################

Input  = path/to/input/dir
Output = path/to/output/dir

# Default way of specifying pixel size and particles per tetrahedron:
TotalPixels = 500
Particles   = 25

# Alternative way of specifying pixel size and particles per tetrahedron:
# ImagePixels     = 100
# ProjectionDepth = 3

#####################
# OptionalParamters #
#####################

# ProfileFile = prof.out
# LogFile = log.out

# SubsampleLength = 2

# Will result in files named pre_*foo*_app.gtet:
# PrependName = pre_
# AppendName  = _app`
	ExampleBoundsFile = `[Box "my_z_slice"]
# A thin slice containing a big halo for the L0125 box.

#######################
# Required Parameters #
#######################

# Location of lowermost corner:
X = 107.9
Y = 79
Z = 78.5

XWidth = 42.14
YWidth = 42.14
ZWidth = 4.21

#######################
# Optional Parameters #
#######################

# Given axis must be one of [ X | Y | Z ].
# ProjectionAxis = Z

[Ball "my_halo"]
# A bounding box around a sphere whose radius is three times larger than
# the halo's R_vir.

X = 4.602
Y = 100.7
Z = 80.7

Radius = 2.17
RadiusMultiplier = 3 # optional`
)

type SharedConfig struct {
	// Required
	Input, Output string
	// Optional
	LogFile, ProfileFile string
}

func (con *SharedConfig) ValidInput() bool {
	return con.Input != ""
}
func (con *SharedConfig) ValidOutput() bool {
	return con.Output != ""
}
func (con *SharedConfig) ValidLogFile() bool {
	return con.LogFile != ""
}
func (con *SharedConfig) ValidProfileFile() bool {
	return con.ProfileFile != ""
}

type ConvertSnapshotConfig struct {
	SharedConfig
	// Required
	Cells int
	InputFormat string

	// Optional
	IteratedInput, IteratedOutput string
	IterationStart, IterationEnd int
}

func DefaultConvertSnapshotWrapper() *ConvertSnapshotWrapper {
	con := ConvertSnapshotConfig{}
	con.IterationStart = 0
	con.IterationEnd = -1
	return &ConvertSnapshotWrapper{con}
}

func (con *ConvertSnapshotConfig) ValidCells() bool {
	return con.Cells > 0
}
func (con *ConvertSnapshotConfig) ValidInputFormat() bool {
	return con.InputFormat != ""
}
func (con *ConvertSnapshotConfig) ValidIteratedInput() bool {
	return con.IteratedInput != ""
}
func (con *ConvertSnapshotConfig) ValidIteratedOutput() bool {
	return con.IteratedOutput != ""
}
func (con *ConvertSnapshotConfig) ValidIterationStart() bool {
	return con.IterationStart >= 0
}
func (con *ConvertSnapshotConfig) ValidIterationEnd() bool {
	return con.IterationEnd >= 0
}

type DensityConfig struct {
	SharedConfig
	// Required
	TotalPixels, Particles int

	// Optional
	ImagePixels, ProjectionDepth int
	SubsampleLength int
	AppendName, PrependName string
}

func DefaultDensityWrapper() *DensityWrapper {
	dc := DensityConfig{ }
	dc.SubsampleLength = 1
	return &DensityWrapper{dc}
}

func (con *DensityConfig) ValidTotalPixels() bool {
	return con.TotalPixels > 0
}
func (con *DensityConfig) ValidParticles() bool {
	return con.Particles > 0
}
func (con *DensityConfig) ValidSubsampleLength() bool {
	return con.SubsampleLength > 0
}
func (con *DensityConfig) ValidImagePixels() bool {
	return con.ImagePixels > 0
}
func (con *DensityConfig) ValidProjectionDepth() bool {
	return con.ProjectionDepth > 0
}

type ConvertSnapshotWrapper struct {
	ConvertSnapshot ConvertSnapshotConfig
}

type DensityWrapper struct {
	Density DensityConfig
}

type BallConfig struct {
	// Required
	X, Y, Z, Radius float64

	// Optional
	RadiusMultiplier float64
	Name string
}

func (ball *BallConfig) CheckInit(name string, totalWidth float64) error {
	if ball.Radius == 0 {
		return fmt.Errorf(
			"Need to specify a positive radius for Ball '%s'.", name,
		)
	}

	if ball.X >= totalWidth || ball.X < 0 {
		return fmt.Errorf(
			"X center of Ball '%s' must be in range [0, %g), but is %g",
			name, totalWidth, ball.X,
		)
	} else if ball.Y >= totalWidth || ball.Y < 0 {
		return fmt.Errorf(
			"Y center of Ball '%s' must be in range [0, %g), but is %g",
			name, totalWidth, ball.Y,
		)
	} else if ball.Z >= totalWidth || ball.Z < 0 {
		return fmt.Errorf(
			"Z center of Ball '%s' must be in range [0, %g), but is %g",
			name, totalWidth, ball.Z,
		)
	}

	ball.Name = name
	if ball.RadiusMultiplier == 0 {
		ball.RadiusMultiplier = 1
	} else if ball.RadiusMultiplier < 0 {
		return fmt.Errorf(
			"Ball '%s' given a negative radius multiplier, %g.",
			name, ball.RadiusMultiplier,
		)
	}

	return nil
}

func (ball *BallConfig) Box(totalWidth float64) *BoxConfig {
	box := &BoxConfig{}
	rad := ball.Radius * ball.RadiusMultiplier

	box.XWidth, box.YWidth, box.ZWidth = 2 * rad, 2 * rad, 2 * rad

	if ball.X > rad {
		box.X = ball.X - rad
	} else {
		box.X = ball.X - rad + totalWidth
	}

	if ball.Y > rad {
		box.Y = ball.Y - rad
	} else {
		box.Y = ball.Y - rad + totalWidth
	}

	if ball.Z > rad {
		box.Z = ball.Z - rad
	} else {
		box.Z = ball.Z - rad + totalWidth
	}

	box.Name = ball.Name
	
	return box
}

type BoxConfig struct {
	// Required
	X, Y, Z float64
	XWidth, YWidth, ZWidth float64

	// Optional
	ProjectionAxis string

	// Optional, "undocumented"
	Name string
}

func (box *BoxConfig) CheckInit(name string, totalWidth float64) error {
	if box.XWidth <= 0 {
		return fmt.Errorf(
			"Need to specify a positive XWidth for Box '%s'", name,
		)
	} else if box.YWidth <= 0 {
		return fmt.Errorf(
			"Need to specify a positive YWidth for Box '%s'", name,
		)
	} else if box.ZWidth <= 0 {
		return fmt.Errorf(
			"Need to specify a positive ZWidth for Box '%s'", name,
		)
	}

	if box.X >= totalWidth || box.X < 0 {
		return fmt.Errorf(
			"X origin of Box '%s' must be in range [0, %g), but is %g",
			name, totalWidth, box.X,
		)
	} else if box.Y >= totalWidth || box.Y < 0 {
		return fmt.Errorf(
			"Y origin of Box '%s' must be in range [0, %g), but is %g",
			name, totalWidth, box.Y,
		)
	} else if box.Z >= totalWidth || box.Z < 0 {
		return fmt.Errorf(
			"Z origin of Box '%s' must be in range [0, %g), but is %g",
			name, totalWidth, box.Z,
		)
	}

	tmp := box.ProjectionAxis
	box.ProjectionAxis = strings.ToUpper(box.ProjectionAxis)
	if box.ProjectionAxis != "" ||
		box.ProjectionAxis != "X" ||
		box.ProjectionAxis != "Y" ||
		box.ProjectionAxis != "Z" {

		return fmt.Errorf(
			"ProjectionAxis of Box '%s' must be one of [X | Y | Z]. '%s' is " + 
				"not recognized.", box.Name, tmp,
		)
	}

	box.Name = name

	return nil
}

func (box *BoxConfig) IsProjection() bool { return box.ProjectionAxis != "" }

type BoundsConfig struct {
	Ball map[string]*BallConfig
	Box  map[string]*BoxConfig
}

func ReadBoundsConfig(fname string, totalWidth float64) ([]BoxConfig, error) {
	bc := BoundsConfig{}

	if err := gcfg.ReadFileInto(&bc, fname); err != nil {
		return nil, err
	}

	boxes := []BoxConfig{}
	for name, ball := range bc.Ball {
		if err := ball.CheckInit(name, totalWidth); err != nil {
			return nil, err
		}
		boxes = append(boxes, *ball.Box(totalWidth))
	}
	for name, box := range bc.Box {
		if err := box.CheckInit(name, totalWidth); err != nil {
			return nil, err
		}
		boxes = append(boxes, *box)
	}

	return boxes, nil
}
