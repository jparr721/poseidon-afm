//go:build windows

package persist_launchd

import (
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

func runCommand(task structs.Task) {
	msg := task.NewResponse()
	msg.SetError("Not Implemented on Windows")
	task.Job.SendResponses <- msg
	return
}
