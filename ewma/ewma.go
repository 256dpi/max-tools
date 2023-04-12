package main

import (
	"sync"
	"time"

	"github.com/256dpi/max-go"
	"github.com/cloudflare/golibs/ewma"

	"github.com/256dpi/max-tools/utils"
)

type object struct {
	in    *max.Inlet
	out   *max.Outlet
	ewma  *ewma.Ewma
	mutex sync.Mutex
}

// TODO: Support Update with time.

func (o *object) Init(obj *max.Object, args []max.Atom) bool {
	// add inlet and outlet
	o.in = obj.Inlet(max.Any, "number to average", true)
	o.out = obj.Outlet(max.Float, "average result")

	// get half life
	halfLife := time.Second
	if len(args) > 0 {
		hl := utils.Int(args[0])
		if hl > 0 {
			halfLife = time.Duration(hl) * time.Millisecond
		}
	}

	// create ewma
	o.ewma = ewma.NewEwma(halfLife)

	return true
}

func (o *object) Handle(_ int, _ string, data []max.Atom) {
	// acquire mutex
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// get value
	var value float64
	if len(data) > 0 {
		value = utils.Float(data[0])
	}

	// update ewma
	o.ewma.UpdateNow(value)

	// send output
	o.out.Float(o.ewma.Current)
}

func (o *object) Free() {}

func main() {
	max.Register("ewma", &object{})
}
