package main

import (
	"fmt"
	"github.com/codecrafters-io/docker-starter-go/app/docker"
	"github.com/codecrafters-io/docker-starter-go/app/util"
	"os"
	"os/exec"
	"syscall"
)

func chroot(dir string) {
	err := syscall.Chroot(dir)
	util.ExitIfErr(err)
	os.Chdir("/")
}
func createNullDevice() {
	os.Mkdir("/dev", 0755)
	devNull, _ := os.Create("/dev/null")
	devNull.Close()
}

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	image := os.Args[2]
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	containerName := fmt.Sprintf("%s-*", image)
	tempDir, err := os.MkdirTemp("", containerName)
	util.ExitIfErr(err)
	docker.Pull(image, tempDir)
	chroot(tempDir)
	createNullDevice()
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID,
	}
	cmd.Run()
	exitCode := cmd.ProcessState.ExitCode()
	os.Exit(exitCode)
}
