package client

import (
	"context"
	"encoding/json"

	"github.com/Foresee-Security/hrobot-go/models"
)

func (c *Client) FailoverGetList(ctx context.Context) ([]models.Failover, error) {
	url := c.baseURL + "/failover"
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var failoverList []models.FailoverResponse
	err = json.Unmarshal(bytes, &failoverList)
	if err != nil {
		return nil, err
	}

	var data []models.Failover
	for _, failover := range failoverList {
		data = append(data, failover.Failover)
	}

	return data, nil
}

func (c *Client) FailoverGet(ctx context.Context, ip string) (*models.Failover, error) {
	url := c.baseURL + "/failover/" + ip
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var failoverResp models.FailoverResponse
	err = json.Unmarshal(bytes, &failoverResp)
	if err != nil {
		return nil, err
	}

	return &failoverResp.Failover, nil
}
