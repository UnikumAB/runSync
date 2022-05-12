// Package lockfile implements a lock to limit a binary to one process per
// anchor file.
package lockfile

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

const lockSuffix = ".lock"

// Lock is lockfile to limit a binary to one process per anchor file.
type Lock string

type LockError struct {
	Filename string
	Hostname string
	Pid      int
}

func (e LockError) Error() string {
	return fmt.Sprintf("lockfile: %s: already exists (Host %v, PID %d)", e.Filename, e.Hostname, e.Pid)
}

// Create a lock for the given binary anchorFile.
// Returns an error if the lock already exists.
func Create(anchorFile string) (Lock, error) {
	filename := anchorFile + lockSuffix
	_, err := os.Stat(filename)

	if err == nil {
		// file exists
		line, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", errors.Wrapf(err, "lockfile: %s", filename)
		}

		hostname, pid, err := parseFileContent(line)
		if err != nil {
			return "", errors.Wrapf(err, "lockfile: %s: already exists", filename)
		}

		currentHostname, err := os.Hostname()

		if err != nil {
			currentHostname = "unknown"
		}

		if hostname == currentHostname {
			alive := checkIfPidIsAlive(pid)
			if !alive {
				_ = os.Remove(filename)
			} else {
				le := LockError{
					Filename: filename,
					Hostname: hostname,
					Pid:      pid,
				}
				return "", le
			}
		} else {
			le := LockError{
				Filename: filename,
				Hostname: hostname,
				Pid:      pid,
			}
			return "", le
		}
	}

	pid := os.Getpid()

	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		hostname = "unknown"
	}

	err = createLockfile(filename, fmt.Sprintf("%s:%d\n", hostname, pid))

	if err != nil {
		return "", err
	}

	return Lock(filename), nil
}

func checkIfPidIsAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process could not be found on non-unix system
		return false
	} else {
		// On Unix system sending signal 0 checks if process is alive
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			// Process could not be found
			return false
		} else {
			// Process was found on system and lockfile is from this system
			return true
		}
	}
}

func parseFileContent(line []byte) (hostname string, pid int, err error) {
	lineString := strings.TrimSpace(string(line))
	parts := strings.Split(lineString, ":")

	if len(parts) == 2 {
		hostname = parts[0]
		pid, err = strconv.Atoi(parts[1])

		return
	}

	return "", 0, fmt.Errorf("lockfile not parsable. content: %q", line)
}
func createLockfile(filename string, content string) error {
	fp, err := os.Create(filename)

	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer fp.Close()

	if _, err := io.WriteString(fp, content); err != nil {
		return err
	}

	return nil
}

// Release the lock.
// The protected process should call this method during shutdown.
func (l *Lock) Release() error {
	s := string(*l)
	if s == "" {
		return nil
	}

	err := os.Remove(s)

	if err != nil {
		*l = ""
	}

	return err
}
