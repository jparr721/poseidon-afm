//go:build windows

package ls

import (
	"os"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

func GetPermission(finfo os.FileInfo) structs.FilePermission {
	perms := structs.FilePermission{}
	perms.Permissions = finfo.Mode().Perm().String()
	if finfo.IsDir() {
		perms.Permissions = "d" + perms.Permissions[1:]
	}
	// Windows doesn't have Unix-style UID/GID
	perms.UID = 0
	perms.GID = 0
	perms.User = ""
	perms.Group = ""
	return perms
}
