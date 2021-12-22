package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runSync/lockfile"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var syncFile = flag.String("syncFile", "./sync.ts", "The file used for the timestamp")
var minIntervalParam = flag.String("maxInterval", "12h", "Minimum time between runs i.e. 5h30m40s")
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	debug := flag.Bool("debug", false, "sets log level to debug")
	verbose := flag.Bool("verbose", false, "Run with verbose output")
	jsonLog := flag.Bool("json", false, "Log as JSON")
	versionFlag := flag.Bool("version", false, "Version of the program")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if !*jsonLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}

	if *versionFlag {
		log.Info().Msgf("runSync %s, commit %s, built at %s by %s", version, commit, date, builtBy)
		return
	}

	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if *debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}

	minInterval, absoluteSyncFile, err := flags2data()

	if err != nil {
		log.Fatal().Err(err)
	}

	err = runSync(absoluteSyncFile, minInterval, flag.Args())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run sync")
	}
}

func runSync(absoluteSyncFile string, minInterval time.Duration, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("failed to run since you failed to provide a command")
	}

	log.Debug().Msgf("Using %q as sync", absoluteSyncFile)

	log.Debug().Msgf("Using %q as lockfile", absoluteSyncFile+".lock")

	lock, err := lockfile.Create(absoluteSyncFile)

	if err != nil {
		var le lockfile.LockError
		if errors.As(err, &le) {
			log.Warn().Err(err).Msgf("Lock not available: %v", err)
			return nil
		}

		log.Warn().Err(err).Msgf("cannot init lock with %q. reason", absoluteSyncFile+".lock")

		return fmt.Errorf("cannot init lock with %q. reason: %w", absoluteSyncFile+".lock", err)
	}

	defer func() {
		if err := lock.Release(); err != nil {
			log.Fatal().Err(err).Msgf("Cannot unlock %q, reason", lock)
		}
	}()

	newFile := false
	stat, err := os.Stat(absoluteSyncFile)

	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(absoluteSyncFile)

			if err != nil {
				return fmt.Errorf("Cannot create file %q, reason: %w", absoluteSyncFile, err)
			}

			err = file.Close()

			if err != nil {
				return fmt.Errorf("Cannot close the file %q, reason: %w", absoluteSyncFile, err)
			}

			newFile = true
			stat, err = os.Stat(absoluteSyncFile)

			if err != nil {
				return fmt.Errorf("Cannot get stat for %q, reason: %w", absoluteSyncFile, err)
			}
		} else {
			return fmt.Errorf("cannot open %q, reason: %w", absoluteSyncFile, err)
		}
	}

	maxAge := time.Now().Add(minInterval * -1)
	if newFile || stat.ModTime().Before(maxAge) {
		return executeCommandAndTouchSyncFile(absoluteSyncFile, args)
	} else {
		fileAge := time.Now().Local().Sub(stat.ModTime())
		log.Debug().Msgf("Don't run. %v < %v", fileAge, minInterval)
	}

	return nil
}

func flags2data() (time.Duration, string, error) {
	minInterval, err := time.ParseDuration(*minIntervalParam)
	if err != nil {
		return 0, "", fmt.Errorf("cannot parse minInterval %q, reason: %w", *minIntervalParam, err)
	}

	absoluteSyncFile, err := filepath.Abs(*syncFile)

	if err != nil {
		return 0, "", fmt.Errorf("failed to clean path to lockFile %q, reason: %w", *syncFile, err)
	}

	return minInterval, absoluteSyncFile, nil
}

func executeCommandAndTouchSyncFile(absoluteSyncFile string, args []string) error {
	currentTime := time.Now().Local()

	command := exec.Command(args[0], args[1:]...)
	stdout, _ := command.StdoutPipe()
	stderr, _ := command.StderrPipe()

	var wg sync.WaitGroup

	wg.Add(2)

	go copyAndWait(&wg, zerolog.InfoLevel, stdout, "stdout")()
	go copyAndWait(&wg, zerolog.ErrorLevel, stderr, "stderr")()

	command.Stdin = os.Stdin

	if err := command.Start(); err != nil {
		return fmt.Errorf("Failed to start, reason: %w", err)
	}

	log.Printf("Executing %q", command.String())
	wg.Wait()
	err := command.Wait()

	if err != nil {
		return fmt.Errorf("Failed to wait on process, reason: %w", err)
	}

	err = os.Chtimes(absoluteSyncFile, currentTime, currentTime)

	if err != nil {
		return fmt.Errorf("Failed to change timestamp on %q, reason: %w", absoluteSyncFile, err)
	}

	return nil
}

func copyAndWait(wg *sync.WaitGroup, level zerolog.Level, r io.ReadCloser, name string) func() {
	return func() {
		defer func() {
			log.Trace().Msgf("Done waiting for %v", name)

			wg.Done()
		}()

		buf := bufio.NewScanner(r)

		for buf.Scan() {
			log.WithLevel(level).Msg(buf.Text())
		}
	}
}
