package main

import (
	"bufio"
	"io"
	"os/exec"
	"sync"

	"github.com/256dpi/max-go"
	"github.com/256dpi/max-tools/utils"
	"github.com/google/shlex"
)

// TODO: Support line as single symbol mode?
// TODO: Support merging stdout with stderr?

type object struct {
	in     *max.Inlet
	out    *max.Outlet
	status *max.Outlet
	done   *max.Outlet
	cmd    string
	wd     string
	ref    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.Reader
	killed bool
	mutex  sync.Mutex
}

func (o *object) Init(obj *max.Object, args []max.Atom) bool {
	// add inlet and outlets
	o.in = obj.Inlet(max.Any, "input command", true)
	o.out = obj.Outlet(max.Any, "output as list")
	o.status = obj.Outlet(max.Int, "status of command")
	o.done = obj.Outlet(max.Bang, "bang when done")

	// set command
	if len(args) > 0 {
		o.cmd, _ = args[0].(string)
	}

	// set working directory
	if len(args) > 1 {
		o.wd, _ = args[1].(string)
	}

	return true
}

func (o *object) Handle(_ int, msg string, data []max.Atom) {
	// acquire mutex
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// handle message
	switch msg {
	case "cmd":
		// TODO: Handle lists?

		// set command
		if len(data) > 0 {
			o.cmd, _ = data[0].(string)
		}
	case "wd":
		// set working directory
		if len(data) > 0 {
			o.wd, _ = data[0].(string)
		}
	case "start", "bang":
		// start command
		o.start()
	case "stop":
		// stop command
		o.stop()
	case "int":
		// start/stop command
		if len(data) > 0 {
			if utils.Int(data[0]) > 0 {
				o.start()
			} else {
				o.stop()
			}
		}
	case "write":
		// TODO: Write to command
	case "writeln":
		// TODO: Send enter to command.
	case "close":
		// TODO: Close input.
	}
}

func (o *object) Free() {
	// stop process
	o.stop()
}

func (o *object) start() {
	// check if started
	if o.ref != nil {
		max.Error("already started")
		return
	}

	// check command
	if o.cmd == "" {
		max.Error("missing command")
		return
	}

	// split command
	cmdList, err := shlex.Split(o.cmd)
	if err != nil {
		max.Error("failed to split command: %s", err.Error())
		return
	}

	// get binary and args
	bin := cmdList[0]
	var args []string
	if len(cmdList) > 1 {
		args = cmdList[1:]
	}

	// prepare command
	cmd := exec.Command(bin, args...)
	cmd.Dir = o.wd

	// log command
	max.Log("running command: %s %v", cmd.Path, cmd.Args[1:])

	// get input pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		max.Error("failed to prepare input: %s", err.Error())
		return
	}

	// get output pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		max.Error("failed to prepare output: %s", err.Error())
		return
	}

	// start command
	err = cmd.Start()
	if err != nil {
		max.Error("failed to run command: %s", err.Error())
		return
	}

	// set state
	o.ref = cmd
	o.stdin = stdin
	o.stdout = stdout
	o.killed = false

	// run handler
	go o.handler()
}

func (o *object) handler() {
	// set status
	o.status.Int(1)

	// scan output
	scanner := bufio.NewScanner(o.stdout)
	for scanner.Scan() {
		o.out.Any(scanner.Text(), nil)
	}

	// await exit
	err := o.ref.Wait()

	// acquire mutex
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// handle error
	if err != nil && !o.killed {
		max.Error("command failed: %s", err.Error())
	}

	// set status and done
	o.status.Int(0)
	o.done.Bang()

	// clear state
	o.ref = nil
	o.stdin = nil
	o.stdout = nil
}

func (o *object) stop() {
	// check if stopped
	if o.ref == nil {
		max.Error("already stopped")
		return
	}

	// set flag
	o.killed = true

	// kill command
	err := o.ref.Process.Kill()
	if err != nil {
		max.Error("failed to stop command: %s", err.Error())
		return
	}
}

func main() {
	max.Register("syscmd", &object{})
}
