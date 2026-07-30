package client

import (
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"

	"github.com/Foresee-Security/hrobot-go/models"
)

func (c *Client) ServerGetList(ctx context.Context) ([]models.Server, error) {
	url := c.baseURL + "/server"
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var servers []models.ServerResponse
	err = json.Unmarshal(bytes, &servers)
	if err != nil {
		return nil, err
	}

	data := make([]models.Server, 0, len(servers))
	for i := range servers {
		data = append(data, servers[i].Server)
	}

	return data, nil
}

func (c *Client) ServerGet(ctx context.Context, id int) (*models.Server, error) {
	url := c.baseURL + fmt.Sprintf("/server/%v", id)
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var serverResp models.ServerResponse
	err = json.Unmarshal(bytes, &serverResp)
	if err != nil {
		return nil, err
	}

	return &serverResp.Server, nil
}

func (c *Client) ServerSetName(ctx context.Context, id int, input *models.ServerSetNameInput) (*models.Server, error) {
	url := c.baseURL + fmt.Sprintf("/server/%v", id)

	formData := neturl.Values{}
	formData.Set("server_name", input.Name)

	bytes, err := c.doPostFormRequest(ctx, url, formData)
	if err != nil {
		return nil, err
	}

	var serverResp models.ServerResponse
	err = json.Unmarshal(bytes, &serverResp)
	if err != nil {
		return nil, err
	}

	return &serverResp.Server, nil
}

func (c *Client) ServerReverse(ctx context.Context, id int) (*models.Cancellation, error) {
	url := c.baseURL + fmt.Sprintf("/server/%v/reversal", id)

	bytes, err := c.doPostFormRequest(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	var cancelResp models.CancellationResponse
	err = json.Unmarshal(bytes, &cancelResp)
	if err != nil {
		return nil, err
	}

	return &cancelResp.Cancellation, nil
}
