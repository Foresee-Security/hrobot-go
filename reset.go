package client

import (
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"

	"github.com/Foresee-Security/hrobot-go/models"
)

func (c *Client) ResetGet(ctx context.Context, id int) (*models.Reset, error) {
	url := c.baseURL + fmt.Sprintf("/reset/%v", id)
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var resetResp models.ResetResponse
	err = json.Unmarshal(bytes, &resetResp)
	if err != nil {
		return nil, err
	}

	return &resetResp.Reset, nil
}

func (c *Client) ResetSet(ctx context.Context, id int, input *models.ResetSetInput) (*models.ResetPost, error) {
	url := c.baseURL + fmt.Sprintf("/reset/%v", id)

	formData := neturl.Values{}
	formData.Set("type", input.Type)

	bytes, err := c.doPostFormRequest(ctx, url, formData)
	if err != nil {
		return nil, err
	}

	var resetResp models.ResetPostResponse
	err = json.Unmarshal(bytes, &resetResp)
	if err != nil {
		return nil, err
	}

	return &resetResp.Reset, nil
}
