//go:build linux || darwin

package ls

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

func GetPermission(finfo os.FileInfo) structs.FilePermission {
	perms := structs.FilePermission{}
	perms.Permissions = finfo.Mode().Perm().String()
	if finfo.Mode()&os.ModeSetuid != 0 {
		perms.SetUID = true
		if perms.Permissions[3] == 'x' {
			perms.Permissions = perms.Permissions[0:3] + "s" + perms.Permissions[4:]
		} else {
			perms.Permissions = perms.Permissions[0:3] + "S" + perms.Permissions[4:]
		}
	}
	if finfo.Mode()&os.ModeSetgid != 0 {
		perms.SetGID = true
		if perms.Permissions[6] == 'x' {
			perms.Permissions = perms.Permissions[0:6] + "s" + perms.Permissions[7:]
		} else {
			perms.Permissions = perms.Permissions[0:6] + "S" + perms.Permissions[7:]
		}
	}
	if finfo.Mode()&os.ModeSticky != 0 {
		perms.Sticky = true
		perms.Permissions = perms.Permissions[0:8] + "t"
	}
	if finfo.IsDir() {
		perms.Permissions = "d" + perms.Permissions[1:]
	}
	systat := finfo.Sys().(*syscall.Stat_t)
	if systat != nil {
		perms.UID = int(systat.Uid)
		perms.GID = int(systat.Gid)
		tmpUser, err := user.LookupId(strconv.Itoa(perms.UID))
		if err == nil {
			perms.User = tmpUser.Username
		}
		tmpGroup, err := user.LookupGroupId(strconv.Itoa(perms.GID))
		if err == nil {
			perms.Group = tmpGroup.Name
		}
	}
	return perms
}
