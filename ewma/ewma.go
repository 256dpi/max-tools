package main

import (
	"sync"
	"time"

	"github.com/256dpi/max-go"

	"github.com/cloudflare/golibs/ewma"
)

type object struct {
	in    *max.Inlet
	out   *max.Outlet
	ewma  *ewma.Ewma
	mutex sync.Mutex
}

// TODO: Support Update with time.

func (o *object) Init(obj *max.Object, args []max.Atom) bool {
	// add inlet and outlets
	o.in = obj.Inlet(max.Any, "number to average", true)
	o.out = obj.Outlet(max.Float, "average result")

	// get half life
	halfLife := time.Second
	if len(args) > 0 {
		hl, _ := args[0].(int64)
		if hl > 0 {
			halfLife = time.Duration(hl) * time.Millisecond
		}
	}

	// create ewma
	o.ewma = ewma.NewEwma(halfLife)

	return true
}

func (o *object) Handle(_ int, _ string, data []max.Atom) {
	// get value
	value := 0.0
	if len(data) > 0 {
		switch v := data[0].(type) {
		case int64:
			value = float64(v)
		case float64:
			value = v
		}
	}

	// acquire mutex
	o.mutex.Lock()

	// update ewma
	o.ewma.UpdateNow(value)
	cur := o.ewma.Current

	// release mutex
	o.mutex.Unlock()

	// send output
	o.out.Float(cur)
}

func (o *object) Free() {}

func init() {
	max.Register("ewma", &object{})
}

func main() {
	// not called
}
