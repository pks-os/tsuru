package app

import (
	"fmt"
	"github.com/timeredbull/tsuru/api/unit"
	"github.com/timeredbull/tsuru/log"
)

const (
	ChanSize    = 10
	runAttempts = 5
)

type Message struct {
	app     *App
	success chan bool
}

var env chan Message = make(chan Message, ChanSize)

type Cmd struct {
	cmd    string
	result chan CmdResult
	u      unit.Unit
}

type CmdResult struct {
	err    error
	output []byte
}

var cmds chan Cmd = make(chan Cmd)

func init() {
	go collectEnvVars()
	go runCommands()
}

func runCommands() {
	for cmd := range cmds {
		out, err := cmd.u.Command(cmd.cmd)
		if cmd.result != nil {
			r := CmdResult{output: out, err: err}
			cmd.result <- r
		}
	}
}

func runCmd(cmd string, msg Message) {
	c := Cmd{
		u:      msg.app.unit(),
		cmd:    cmd,
		result: make(chan CmdResult),
	}
	cmds <- c
	var r CmdResult
	r = <-c.result
	for i := 0; r.err != nil && i < runAttempts; i++ {
		cmds <- c
		r = <-c.result
	}
	log.Printf("running %s on %s, output:\n %s", cmd, msg.app.Name, string(r.output))
	if msg.success != nil {
		msg.success <- r.err == nil
	}
}

func collectEnvVars() {
	for e := range env {
		cmd := "cat > /home/application/apprc <<END\n"
		for k, v := range e.app.Env {
			cmd += fmt.Sprintf(`export %s="%s"`+"\n", k, v)
		}
		cmd += "END\n"
		runCmd(cmd, e)
	}
}
