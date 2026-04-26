//go:build windows

package daemon

import (
	"errors"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
)

func TestStartReturnsUnsupportedOSOnWindows(t *testing.T) {
	err := Start(config.Default())
	if !errors.Is(err, ErrUnsupportedOS) {
		t.Errorf("Start() error = %v, want ErrUnsupportedOS", err)
	}
}

func TestInitReturnsUnsupportedOSOnWindows(t *testing.T) {
	err := Init("bash")
	if !errors.Is(err, ErrUnsupportedOS) {
		t.Errorf("Init() error = %v, want ErrUnsupportedOS", err)
	}
}
