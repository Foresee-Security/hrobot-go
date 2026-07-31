package hrobot

import "context"

// RobotClient is the Robot Webservice surface [Client] implements.
//
// It exists so a consumer can substitute a fake in tests without hand-writing
// the surface, and as one place to read what this client can do. Prefer
// declaring a narrower interface naming only the calls you actually make, per
// the Go convention of defining interfaces where they are consumed.
// Constructors return the concrete *[Client], which satisfies this and any such
// narrower interface.
//
// It covers what a substitute could meaningfully stand in for, which means
// everything that performs I/O. Deliberately absent are [Client.GetVersion],
// which returns a package constant that a fake could only lie about, and
// [Client.String] and [Client.LogValue], which exist to redact a credential
// rather than to be substituted. TestRobotClientCoversTheIOSurface pins that
// list, so a method added to *Client cannot quietly go missing here.
//
// Adding a method to this interface is a breaking change for anyone
// implementing it outside this module. That cost is accepted, because the
// interface is meant for test doubles in consumers we control rather than as an
// extension point. A consumer who wants to be insulated from that should
// declare its own narrower interface.
//
// Every method that reaches the network takes a context as its first parameter,
// so a caller can cancel a request and impose a deadline. A request made with a
// context carrying no deadline is still bounded by the client's own timeout.
// SetCredentials is the one exception, because it only validates and stores.
type RobotClient interface {
	SetCredentials(username, password string) error
	ValidateCredentials(ctx context.Context) error

	ServerGetList(ctx context.Context) ([]Server, error)
	ServerGet(ctx context.Context, id int) (*Server, error)
	ServerSetName(ctx context.Context, id int, input *ServerSetNameInput) (*Server, error)
	ServerCancellationWithdraw(ctx context.Context, id int) (*Cancellation, error)

	KeyGetList(ctx context.Context) ([]Key, error)
	KeySet(ctx context.Context, input *KeySetInput) (*Key, error)

	IPGetList(ctx context.Context) ([]IP, error)

	RDNSGetList(ctx context.Context) ([]RDNS, error)
	RDNSGet(ctx context.Context, ip string) (*RDNS, error)

	BootLinuxGet(ctx context.Context, id int) (*Linux, error)
	BootLinuxDelete(ctx context.Context, id int) (*Linux, error)
	BootLinuxSet(ctx context.Context, id int, input *LinuxSetInput) (*Linux, error)
	BootRescueGet(ctx context.Context, id int) (*Rescue, error)
	BootRescueDelete(ctx context.Context, id int) (*Rescue, error)
	BootRescueSet(ctx context.Context, id int, input *RescueSetInput) (*Rescue, error)

	ResetGet(ctx context.Context, id int) (*Reset, error)
	ResetSet(ctx context.Context, id int, input *ResetSetInput) (*ResetPost, error)

	FailoverGetList(ctx context.Context) ([]Failover, error)
	FailoverGet(ctx context.Context, ip string) (*Failover, error)
}
