//go:build windows

package sudo

import "github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"

func sudoWithPromptOption(task structs.Task, args Arguments) {
	msg := task.NewResponse()
	msg.SetError("Not Implemented on Windows")
	task.Job.SendResponses <- msg
	return
}
