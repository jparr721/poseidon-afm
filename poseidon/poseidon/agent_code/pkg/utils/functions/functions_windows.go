//go:build windows

package functions

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"unicode/utf16"
)

func isElevated() bool {
	return false
}

func getArchitecture() string {
	return runtime.GOARCH
}

func getProcessName() string {
	name, err := os.Executable()
	if err != nil {
		return ""
	}
	return name
}

func getDomain() string {
	return ""
}

func getOS() string {
	return "Windows"
}

func getUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return ""
	}
	return currentUser.Username
}

func getEffectiveUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return ""
	}
	return currentUser.Username
}

func getPID() int {
	return os.Getpid()
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}

func getCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

// Helper function to convert DWORD byte counts to
// human readable sizes.
func UINT32ByteCountDecimal(b uint32) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint32(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float32(b)/float32(div), "kMGTPE"[exp])
}

// Helper function to convert LARGE_INTEGER byte
// counts to human readable sizes.
func UINT64ByteCountDecimal(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

// Helper function to build a string from a WCHAR string
func UTF16ToString(s []uint16) []string {
	var results []string
	cut := 0
	for i, v := range s {
		if v == 0 {
			if i-cut > 0 {
				results = append(results, string(utf16.Decode(s[cut:i])))
			}
			cut = i + 1
		}
	}
	if cut < len(s) {
		results = append(results, string(utf16.Decode(s[cut:])))
	}
	return results
}
