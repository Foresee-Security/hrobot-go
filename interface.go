package hrobot

import "context"

// RobotClient is the Robot Webservice surface [Client] implements.
//
// It is provided so that a consumer can substitute a fake in tests, and as a
// single place to read the whole surface. Prefer declaring a narrower interface
// naming only the calls you actually make, per the Go convention of defining
// interfaces where they are consumed. Constructors return the concrete
// *[Client], which satisfies this and any such narrower interface.
//
// Every method that performs I/O takes a context as its first parameter, so a
// caller can cancel a request and impose a deadline. A request made with a
// context carrying no deadline is still bounded by the client's own timeout.
//
// Methods that only validate their arguments, such as SetCredentials, take no
// context because they never reach the network.
type RobotClient interface {
	GetVersion() string
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
