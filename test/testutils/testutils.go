package testutils

import (
	"fmt"
	"strings"
	"testing"
)

func SetDebugPortsForConfig(t *testing.T, key string, debugPorts []int) {
	t.Setenv(key, strings.Trim(strings.Join(strings.Fields(fmt.Sprint(debugPorts)), ","), "[]"))
}
