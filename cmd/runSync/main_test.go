package main

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func Test_runSync(t *testing.T) {
	type args struct {
		absoluteSyncFile string
		minInterval      time.Duration
		args             []string
		debug            bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "basic test",
			args: args{
				absoluteSyncFile: "sync.ts" + uuid.NewString(),
				minInterval:      time.Second * 5,
				args:             []string{"true"},
				debug:            false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.debug {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
			}
			if err := runSync(tt.args.absoluteSyncFile, tt.args.minInterval, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("runSync() error = %v, wantErr %v", err, tt.wantErr)
			}
			cleanup(tt.args.absoluteSyncFile)
		})
	}
}

func cleanup(syncFile string) {
	_ = os.Remove(syncFile)
	_ = os.Remove(syncFile + ".lock")
}
