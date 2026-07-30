package client

import (
	"context"

	"github.com/Foresee-Security/hrobot-go/models"
)

// RobotClient is the Hetzner Robot Webservice surface this package implements.
//
// Every method that performs I/O takes a context as its first parameter, so a
// caller can cancel a request and impose a deadline. Requests made with a
// context carrying no deadline still fall back to the client's own timeout.
type RobotClient interface {
	SetBaseURL(baseURL string)
	SetUserAgent(userAgent string)
	GetVersion() string
	SetCredentials(username, password string) error
	ValidateCredentials(ctx context.Context) error

	ServerGetList(ctx context.Context) ([]models.Server, error)
	ServerGet(ctx context.Context, id int) (*models.Server, error)
	ServerSetName(ctx context.Context, id int, input *models.ServerSetNameInput) (*models.Server, error)
	ServerReverse(ctx context.Context, id int) (*models.Cancellation, error)
	KeyGetList(ctx context.Context) ([]models.Key, error)
	KeySet(ctx context.Context, input *models.KeySetInput) (*models.Key, error)
	IPGetList(ctx context.Context) ([]models.IP, error)
	RDnsGetList(ctx context.Context) ([]models.Rdns, error)
	RDnsGet(ctx context.Context, ip string) (*models.Rdns, error)
	BootLinuxGet(ctx context.Context, id int) (*models.Linux, error)
	BootLinuxDelete(ctx context.Context, id int) (*models.Linux, error)
	BootLinuxSet(ctx context.Context, id int, input *models.LinuxSetInput) (*models.Linux, error)
	BootRescueGet(ctx context.Context, id int) (*models.Rescue, error)
	BootRescueDelete(ctx context.Context, id int) (*models.Rescue, error)
	BootRescueSet(ctx context.Context, id int, input *models.RescueSetInput) (*models.Rescue, error)
	ResetGet(ctx context.Context, id int) (*models.Reset, error)
	ResetSet(ctx context.Context, id int, input *models.ResetSetInput) (*models.ResetPost, error)
	FailoverGetList(ctx context.Context) ([]models.Failover, error)
	FailoverGet(ctx context.Context, ip string) (*models.Failover, error)
}
