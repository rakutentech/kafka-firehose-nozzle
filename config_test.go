package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cases := []struct {
		in                  string
		success             bool
		dopplerAddr, errStr string
	}{
		{
			in:          "basic.toml",
			success:     true,
			dopplerAddr: "wss://doppler.cloudfoundry.net",
		},

		{
			in:      "not-exist.toml",
			success: false,
			errStr:  "no such file",
		},
	}

	for i, tc := range cases {
		path := filepath.Join("./fixtures/", tc.in)
		config, err := LoadConfig(path)
		if tc.success {
			if err != nil {
				t.Fatalf("#%d expect %q to be nil", i, err)
			}

			// ok
			if config.CF.DopplerAddr != tc.dopplerAddr {
				t.Fatalf("#%d expect %q to be %q", i, config.CF.DopplerAddr, tc.dopplerAddr)
			}

			continue
		}

		if err == nil {
			t.Fatalf("#%d expect to be failed", i)
		}

		if !strings.Contains(err.Error(), tc.errStr) {
			t.Fatalf("#%d expect %q to contain %q", i, err.Error(), tc.errStr)
		}
	}

}
