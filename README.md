# RunSync
Runsync is a tool for cronjobs that are not supposed to run on multiple machines at the same time, but it doesn't matter which machine runs it.

One example is a database backup that can be run on any of the DB servers but shouldn't run twice.
In the case of multiple servers they need to share some storage where the sync and lock file can live.

## Usage

```
Usage of runSync:
runSync <options> <command>
    -debug
        Run with debug output
    -lockFile string
        The file used for the lock. Defaults to sync file name with .lock appended.
    -maxInterval string
        Minimum time between runs i.e. 5h30m40s (default "12h")
    -syncFile string
        The file used for the timestamp (default "./sync.ts")
    -verbose
        Run with verbose output
```
the output of the command will be visible.
