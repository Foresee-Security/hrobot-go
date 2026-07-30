package client

import (
	"context"
	"encoding/json"

	"github.com/Foresee-Security/hrobot-go/models"
)

func (c *Client) IPGetList(ctx context.Context) ([]models.IP, error) {
	url := c.baseURL + "/ip"
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var ips []models.IPResponse
	err = json.Unmarshal(bytes, &ips)
	if err != nil {
		return nil, err
	}

	data := make([]models.IP, 0, len(ips))
	for i := range ips {
		data = append(data, ips[i].IP)
	}

	return data, nil
}
