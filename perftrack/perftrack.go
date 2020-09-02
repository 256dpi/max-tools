package main

import (
	"sync"

	"github.com/256dpi/max-go"
	"github.com/go-gl/mathgl/mgl64"

	"github.com/256dpi/max-tools/utils"
)

type object struct {
	leap  float64
	decay float64
	in    *max.Inlet
	out   *max.Outlet
	time  float64
	pos   mgl64.Vec3
	rot   mgl64.Quat
	mutex sync.Mutex
}

func (o *object) Init(obj *max.Object, args []max.Atom) bool {
	// args: leap, decay

	// get leap
	if len(args) > 0 {
		o.leap = utils.Float(args[0])
	}

	// get decay
	if len(args) > 1 {
		o.decay = utils.Float(args[1])
	}

	// add inlet and outlet
	o.in = obj.Inlet(max.Any, "t, px, py, pz, rw, rx, ry, rz", true)
	o.out = obj.Outlet(max.Any, "t, px, py, pz, rw, rx, ry, rz")

	return true
}

func (o *object) Handle(_ int, _ string, data []max.Atom) {
	// acquire mutex
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// check data
	if len(data) != 8 {
		max.Error("invalid arguments")
		return
	}

	// get time
	time := utils.Float(data[0])

	// get position
	px := utils.Float(data[1])
	py := utils.Float(data[2])
	pz := utils.Float(data[3])
	pos := mgl64.Vec3{px, py, pz}

	// get rotation
	rw := utils.Float(data[4])
	rx := utils.Float(data[5])
	ry := utils.Float(data[6])
	rz := utils.Float(data[7])
	rot := mgl64.Quat{W: rw, V: mgl64.Vec3{rx, ry, rz}}

	max.Pretty("input", time, pos, rot)

	// set values and return if zero
	if o.time == 0 {
		o.time = time
		o.pos = pos
		o.rot = rot
		return
	}

	/* Smoothing */

	// apply Lerp-EWMA to position
	if o.decay > 0 {
		pos = mgl64.Vec3{
			utils.Lerp(o.pos.X(), pos.X(), o.decay),
			utils.Lerp(o.pos.Y(), pos.Y(), o.decay),
			utils.Lerp(o.pos.Z(), pos.Z(), o.decay),
		}
	}

	// apply Slerp-EWMA to rotation
	if o.decay > 0 {
		rot = mgl64.QuatSlerp(o.rot, rot, o.decay)
	}

	/* Prediction */

	// get differences
	dTime := time - o.time
	dPos := pos.Sub(o.pos)
	dRot := rot.Sub(o.rot)

	max.Pretty("diff", dTime, dPos, dRot)

	// get factor
	fac := 1.0 / dTime * (dTime + o.leap)

	// predict position and rotation
	pTime := o.time + dTime*fac
	pPos := o.pos.Add(dPos.Mul(fac))
	pRot := o.rot.Add(dRot.Mul(mgl64.Quat{W: fac}))

	max.Pretty("pred", fac, pTime, pPos, pRot)

	// send output
	o.out.List([]max.Atom{
		pTime,
		pPos.X(), pPos.Y(), pPos.Z(),
		pRot.W, pRot.X(), pRot.Y(), pRot.Z(),
	})

	// update state
	o.time = time
	o.pos = pos
	o.rot = rot
}

func (o *object) Free() {}

func init() {
	max.Register("perftrack", &object{})
}

func main() {
	// not called
}
