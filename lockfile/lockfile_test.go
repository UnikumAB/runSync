package lockfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLock(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "lockfile_test")

	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}

	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpdir)

	anchor := filepath.Join(tmpdir, "testfile")
	l1, err := Create(anchor)

	if err != nil {
		t.Fatalf("l1 Create() failed: %v", err)
	}

	if _, err = Create(anchor); err == nil {
		t.Error("second Create() should fail")
	}

	if err := l1.Release(); err != nil {
		t.Fatalf("l1.Release() failed: %v", err)
	}

	l2, err := Create(anchor)

	if err != nil {
		t.Fatalf("l2 Create() failed: %v", err)
	}

	if err := l2.Release(); err != nil {
		t.Fatalf("l2.Release() failed: %v", err)
	}
}

func TestPidCheck(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "lockfile_test")

	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}

	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpdir)

	anchor := filepath.Join(tmpdir, "testfile")
	pid := os.Getpid()

	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		hostname = "unknown"
	}

	err = createLockfile(anchor+lockSuffix, fmt.Sprintf("%s:%d\n", hostname, pid))

	if err != nil {
		t.Fatalf("Cannot create lockfile for test: %v", err)
	}

	l1, err := Create(anchor)

	if err == nil {
		t.Fatalf("l1 Create() didn't fail")
	}

	defer func(l1 *Lock) {
		err := l1.Release()
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	}(&l1)
}

func TestHostnameCheck(t *testing.T) {
	type args struct {
		content string
	}

	host, _ := os.Hostname()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Fail to lock because pid exists on same host",
			args: args{
				content: host + ":" + strconv.Itoa(os.Getpid()) + "\n",
			},
			wantErr: true,
		},
		{
			name: "Fail to lock because different hostname",
			args: args{
				content: "not" + host + ":" + strconv.Itoa(os.Getpid()) + "\n",
			},
			wantErr: true,
		},
		{
			name: "Succeed locking because pid DOES NOT exist on same host",
			args: args{
				content: host + ":" + strconv.Itoa(-1) + "\n",
			},
			wantErr: false,
		},
		{
			name: "Fail to lock because file exists and is not parsable",
			args: args{
				content: strconv.Itoa(1234) + "\n",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir, err := ioutil.TempDir("", "lockfile_test")

			if err != nil {
				t.Fatalf("TempDir() failed: %v", err)
			}

			defer func(path string) {
				_ = os.RemoveAll(path)
			}(tmpdir)
			anchor := filepath.Join(tmpdir, "testfile")
			err = createLockfile(anchor+lockSuffix, tt.args.content)
			if err != nil {
				t.Fatalf("Cannot create lockfile for test: %v", err)
			}
			l1, err := Create(anchor)
			if (err != nil) != tt.wantErr {
				t.Errorf("runSync() error = %v, wantErr %v", err, tt.wantErr)
			}
			defer func(l1 *Lock) {
				err := l1.Release()
				if err != nil {
					t.Fatalf("Failed to release lock: %v", err)
				}
			}(&l1)

		})
	}
}
