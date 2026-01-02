package cd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/functions"

	// Poseidon

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

// Run - package function to run cd
func Run(task structs.Task) {
	fixedPath := task.Params
	if strings.HasPrefix(fixedPath, "~/") {
		dirname, _ := os.UserHomeDir()
		fixedPath = filepath.Join(dirname, fixedPath[2:])
	}
	fixedPath, _ = filepath.Abs(fixedPath)
	err := os.Chdir(fixedPath)
	msg := task.NewResponse()
	msg.Completed = true
	if err != nil {
		msg.SetError(err.Error())
	} else {
		msg.UserOutput = fmt.Sprintf("changed directory to: %s", task.Params)
		newCwd := functions.GetCwd()
		callbackUpdate := structs.CallbackUpdate{Cwd: &newCwd}
		msg.CallbackUpdate = &callbackUpdate
	}
	task.Job.SendResponses <- msg
	return
}
