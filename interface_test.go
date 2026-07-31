package hrobot_test

import (
	"reflect"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

// notInRobotClient is every exported method on *Client that RobotClient
// deliberately leaves out, with the reason.
//
// The reason is the point. A bare list would be a place to silence this test,
// whereas an entry that has to justify itself is a decision someone made.
var notInRobotClient = map[string]string{
	"GetVersion": "returns a package constant, so a substitute could only lie about it",
	"String":     "redacts a credential for formatting, it is not behaviour to stand in for",
	"LogValue":   "redacts a credential for slog, same reason as String",
}

// TestRobotClientCoversTheIOSurface catches drift in the direction the compiler
// cannot.
//
// The assertion at client.go proves *Client implements everything RobotClient
// declares. It says nothing about the reverse: a method added to *Client and
// forgotten here compiles perfectly, and the interface silently becomes a stale
// partial copy of the surface. That is the standing cost of keeping a
// hand-maintained interface, and this is what stops it accruing.
func TestRobotClientCoversTheIOSurface(t *testing.T) {
	t.Parallel()

	clientType := reflect.TypeFor[*hrobot.Client]()
	interfaceType := reflect.TypeFor[hrobot.RobotClient]()

	declared := make(map[string]bool, interfaceType.NumMethod())
	for method := range interfaceType.Methods() {
		declared[method.Name] = true
	}

	for method := range clientType.Methods() {
		name := method.Name
		if declared[name] {
			continue
		}
		if _, excluded := notInRobotClient[name]; excluded {
			continue
		}
		t.Errorf("*Client has exported method %s that RobotClient does not declare. "+
			"Add it to the interface, or add it to notInRobotClient with the reason it does not belong.", name)
	}

	// The exclusion list must not outlive the methods it names, or it becomes
	// a licence for a future method of the same name to skip the check.
	for name := range notInRobotClient {
		if _, ok := clientType.MethodByName(name); !ok {
			t.Errorf("notInRobotClient names %s, which *Client no longer has. Remove the entry.", name)
		}
		if declared[name] {
			t.Errorf("%s is both declared in RobotClient and listed as excluded. Pick one.", name)
		}
	}
}
