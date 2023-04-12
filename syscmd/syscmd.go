package main

import (
	"bufio"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/256dpi/max-go"
	"github.com/google/shlex"
	"github.com/kr/pretty"
)

type object struct {
	in     *max.Inlet
	out    *max.Outlet
	status *max.Outlet
	done   *max.Outlet
	cmd    []string
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
		var err error
		o.cmd, err = shlex.Split(args[0].(string))
		if err != nil {
			max.Error("failed to split command: %s", err.Error())
		}
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
		// set command
		if len(data) > 0 {
			var err error
			o.cmd, err = shlex.Split(data[0].(string))
			if err != nil {
				max.Error("failed to split command: %s", err.Error())
			}
		}
	case "args":
		// set command
		o.cmd = nil
		for _, value := range data {
			switch v := value.(type) {
			case string:
				o.cmd = append(o.cmd, v)
			case int64:
				o.cmd = append(o.cmd, strconv.FormatInt(v, 10))
			case float64:
				o.cmd = append(o.cmd, strconv.FormatFloat(v, 'f', -1, 64))
			}
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
			if max.ToInt(data[0]) > 0 {
				o.start()
			} else {
				o.stop()
			}
		}
	case "write":
		// write string
		if len(data) > 0 {
			if str, ok := data[0].(string); ok {
				o.write(str)
			}
		}
	case "writeln":
		// write string
		if len(data) > 0 {
			if str, ok := data[0].(string); ok {
				o.write(str + "\n")
			}
		}
	case "close":
		o.close()
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
	if len(o.cmd) == 0 {
		max.Error("missing command")
		return
	}

	// get binary and args
	bin := o.cmd[0]
	var args []string
	if len(o.cmd) > 1 {
		args = o.cmd[1:]
	}

	// prepare command
	cmd := exec.Command(bin, args...)
	cmd.Dir = o.wd
	cmd.SysProcAttr = sysProcAttrs()

	// log command
	max.Log("running command: %s %v", cmd.Path, pretty.Sprint(cmd.Args[1:]))

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

	// use output also for stderr
	cmd.Stderr = cmd.Stdout

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

func (o *object) write(str string) {
	// check if started
	if o.ref == nil {
		max.Error("not started")
		return
	}

	// check stdin
	if o.stdin == nil {
		max.Error("already closed")
		return
	}

	// write
	_, err := o.stdin.Write([]byte(str))
	if err != nil {
		max.Error("write failed: %s", err.Error())
		return
	}
}

func (o *object) close() {
	// check if started
	if o.ref == nil {
		max.Error("not started")
		return
	}

	// check stdin
	if o.stdin == nil {
		max.Error("already closed")
		return
	}

	// close
	err := o.stdin.Close()
	if err != nil {
		max.Error("close failed: %s", err.Error())
		return
	}

	// clear
	o.stdin = nil
}

func (o *object) handler() {
	// set status
	o.status.Int(1)

	// scan output
	scanner := bufio.NewScanner(o.stdout)
	for scanner.Scan() {
		// get line
		line := scanner.Text()

		// replace tabs
		line = strings.ReplaceAll(line, "\t", "    ")

		// emit line
		o.out.Any(line, nil)
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
