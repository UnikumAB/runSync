package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/nightlyone/lockfile"
)

var syncFile = flag.String("syncFile", "./sync.ts", "The file used for the timestamp")
var lockFile = flag.String("lockFile", "", "The file used for the lock. Defaults to sync file name with .lock appended.")
var verbose = flag.Bool("verbose", false, "Run with verbose output")
var debug = flag.Bool("debug", false, "Run with debug output")
var minIntervalParam = flag.String("maxInterval", "12h", "Minimum time between runs i.e. 5h30m40s")

func main() {
	flag.Parse()
	if *debug {
		err := flag.Set("verbose", "true")
		if err != nil {
			log.Fatalf("Cannot set verbose flag, reason: %v", err)
		}
	}
	if *lockFile == "" {
		*lockFile = filepath.Join(*syncFile) + ".lock"
	}
	absoluteLockFile, err := filepath.Abs(filepath.Join(*lockFile))
	if err != nil {
		log.Fatalf("Failed to clean path to lockFile %q, reason: %v", *lockFile, err)
	}
	if *verbose {
		log.Printf("Using %q as lockfile", absoluteLockFile)
	}
	absoluteSyncFile, err := filepath.Abs(*syncFile)
	if err != nil {
		log.Fatalf("Failed to clean path to lockFile %q, reason: %v", *syncFile, err)
	}
	if *verbose {
		log.Printf("Using %q as sync", absoluteSyncFile)
	}
	lock, err := lockfile.New(absoluteLockFile)

	if err != nil {
		log.Fatalf("Cannot init lock with %q. reason: %v", absoluteLockFile, err)
	}

	// Error handling is essential, as we only try to get the lock.
	if err = lock.TryLock(); err != nil {
		log.Fatalf("Cannot lock %q, reason: %v", lock, err)
	}

	defer func() {
		if err := lock.Unlock(); err != nil {
			log.Fatalf("Cannot unlock %q, reason: %v", lock, err)
		}
	}()
	newFile := false
	stat, err := os.Stat(absoluteSyncFile)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(absoluteSyncFile)
			if err != nil {
				log.Fatalf("Cannot create file %q, reason: %v", absoluteSyncFile, err)
			}
			err = file.Close()
			if err != nil {
				log.Panicf("Cannot close the file %q, reason: %v", absoluteSyncFile, err)
			}
			newFile = true
			stat, err = os.Stat(absoluteSyncFile)
			if err != nil {
				log.Panicf("Cannot get stat for %q, reason: %v", absoluteSyncFile, err)
			}
		} else {
			log.Fatalf("Cannot open %q, reason: %v", absoluteSyncFile, err)
		}
	}

	minInterval, err := time.ParseDuration(*minIntervalParam)
	if err != nil {
		log.Fatalf("Cannot parse minInterval %q, reason: %v", *minIntervalParam, err)
	}
	maxAge := time.Now().Add(minInterval * -1)
	if newFile || stat.ModTime().Before(maxAge) {
		currentTime := time.Now().Local()
		err = os.Chtimes(absoluteSyncFile, currentTime, currentTime)
		if err != nil {
			log.Fatalf("Failed to change timestamp on %q, reason: %v", absoluteSyncFile, err)
		}
		command := exec.Command(flag.Args()[0], flag.Args()[1:]...)
		stdout, _ := command.StdoutPipe()
		stderr, _ := command.StderrPipe()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer func() {
				if *debug {
					log.Printf("Done waiting for stdout")
				}
				wg.Done()
			}()
			if _, err := io.Copy(os.Stdout, stdout); err != nil {
				log.Fatalf("failed to copy stdout, reason: %v", err)
			}
		}()
		wg.Add(1)
		go func() {
			defer func() {
				if *debug {
					log.Printf("Done waiting for stderr")
				}
				wg.Done()
			}()
			if _, err := io.Copy(os.Stderr, stderr); err != nil {
				log.Fatalf("failed to copy stderr, reason: %v", err)
			}
		}()
		if err := command.Start(); err != nil {
			log.Fatalf("Failed to start, reason: %v", err)
		}

		log.Printf("Executing %q", command.String())
		wg.Wait()
		err := command.Wait()
		if err != nil {
			log.Fatalf("Failed to wait on process, reason: %v", err)
		}
	} else {
		fileAge := time.Now().Local().Sub(stat.ModTime())
		if *verbose {
			log.Printf("Don't run. %v < %v", fileAge, minInterval)
		}
	}

}
