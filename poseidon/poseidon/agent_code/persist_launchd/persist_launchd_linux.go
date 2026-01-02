package persist_launchd

import (

	// Poseidon

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

func runCommand(task structs.Task) {
	msg := task.NewResponse()
	msg.SetError("Not implemented")
	task.Job.SendResponses <- msg
	return
}
