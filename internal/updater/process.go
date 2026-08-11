package updater

import (
	"os"
	"os/exec"
)

func replacementCommand(executable, source, target string) *exec.Cmd {
	return exec.Command(executable, "-apply-update="+source, "-update-target="+target)
}

func Start(source, target string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return replacementCommand(executable, source, target).Start()
}
