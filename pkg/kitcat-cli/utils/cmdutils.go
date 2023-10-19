package utils

import (
	"fmt"
	"os"
	"os/exec"
)

// HasBinary checks if the given binary is available on the user's machine.
func HasBinary(binaryName string) bool {
	cmd := exec.Command("which", binaryName)
	err := cmd.Run()

	// If there is an error, it means the binary is not found.
	return err == nil
}

func ExecShellCommandAt(dir, command string) (string, error) {
	fmt.Println("Executing command:", command)
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func ExecShellCommandInTerm(dir, command string) error {
	fmt.Println("Executing command:", command)
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()

}
