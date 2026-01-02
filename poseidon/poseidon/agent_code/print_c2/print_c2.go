package print_c2

import (
	// Standard

	// Poseidon

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/profiles"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

// Run - package function to run print_c2
func Run(task structs.Task) {
	msg := task.NewResponse()
	msg.UserOutput = profiles.GetAllC2Info()
	msg.Completed = true
	msg.Status = "completed"
	task.Job.SendResponses <- msg
	return
}
