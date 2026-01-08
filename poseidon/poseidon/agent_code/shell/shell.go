package shell

import (
	"encoding/json"
	"fmt"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

// Run - Function that executes the shell command
func Run(task structs.Task) {
	executeShellCommand(task)
}

type Arguments struct {
	Shell string `json:"shell"`
}

func RunConfig(task structs.Task) {
	msg := task.NewResponse()
	args := Arguments{}
	err := json.Unmarshal([]byte(task.Params), &args)
	if err != nil {
		msg.SetError(err.Error())
		task.Job.SendResponses <- msg
		return
	}
	shellBin = args.Shell
	msg.Completed = true
	msg.UserOutput = fmt.Sprintf("Shell updated to: %s", shellBin)

	task.Job.SendResponses <- msg
	return
}
