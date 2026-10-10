package main

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
)

func TestChildExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}
	runErr := exec.Command("/bin/sh", "-c", "exit 3").Run()

	t.Run("child exit code is passed through", func(t *testing.T) {
		code, ok := childExitCode(runErr)
		if !ok || code != 3 {
			t.Fatalf("got (%d, %v), want (3, true)", code, ok)
		}
	})

	t.Run("wrapped exit error is detected", func(t *testing.T) {
		code, ok := childExitCode(fmt.Errorf("wrapped: %w", runErr))
		if !ok || code != 3 {
			t.Fatalf("got (%d, %v), want (3, true)", code, ok)
		}
	})

	t.Run("other errors are not exit errors", func(t *testing.T) {
		if _, ok := childExitCode(errors.New("boom")); ok {
			t.Fatal("plain error must not be treated as a child exit")
		}
	})
}

func TestWantsHelp(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"3.14.0"}, false},
		{[]string{"-h"}, true},
		{[]string{"3.14.0", "--help"}, true},
	}
	for _, tc := range cases {
		if got := wantsHelp(tc.args); got != tc.want {
			t.Errorf("wantsHelp(%q) = %v, want %v", tc.args, got, tc.want)
		}
	}
}
