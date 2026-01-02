package pwd

import (
	// Standard

	"os"

	// Poseidon

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

// Run - interface method that retrieves a process list
func Run(task structs.Task) {
	msg := task.NewResponse()
	dir, err := os.Getwd()
	if err != nil {
		msg.SetError(err.Error())
		task.Job.SendResponses <- msg
		return
	}
	msg.Completed = true
	msg.UserOutput = dir
	task.Job.SendResponses <- msg
	return
}
