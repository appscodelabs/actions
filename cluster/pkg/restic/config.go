package restic

import (
	shell "github.com/codeskyblue/go-sh"
)
type ResticWrapper struct {
	sh          *shell.Session
	scratchDir  string
	enableCache bool
	hostname    string
	cacertFile  string
	secretDir string
}

func New(scratchDir string, enableCache bool, hostname string) *ResticWrapper {
	ctrl := &ResticWrapper{
		sh:          shell.NewSession(),
		scratchDir:  scratchDir,
		enableCache: enableCache,
		hostname:    hostname,
	}
	ctrl.sh.SetDir(scratchDir)
	ctrl.sh.ShowCMD = true
	return ctrl
}
