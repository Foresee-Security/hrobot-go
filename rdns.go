package client

import (
	"context"
	"encoding/json"

	"github.com/Foresee-Security/hrobot-go/models"
)

func (c *Client) RDnsGetList(ctx context.Context) ([]models.Rdns, error) {
	url := c.baseURL + "/rdns"
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var rdnsList []models.RdnsResponse
	err = json.Unmarshal(bytes, &rdnsList)
	if err != nil {
		return nil, err
	}

	var data []models.Rdns
	for _, rdns := range rdnsList {
		data = append(data, rdns.Rdns)
	}

	return data, nil
}

func (c *Client) RDnsGet(ctx context.Context, ip string) (*models.Rdns, error) {
	url := c.baseURL + "/rdns/" + ip
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var rDNSResp models.RdnsResponse
	err = json.Unmarshal(bytes, &rDNSResp)
	if err != nil {
		return nil, err
	}

	return &rDNSResp.Rdns, nil
}
