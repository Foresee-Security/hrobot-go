package client

import (
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"strconv"

	"github.com/Foresee-Security/hrobot-go/models"
)

func (c *Client) BootRescueGet(ctx context.Context, id int) (*models.Rescue, error) {
	url := c.baseURL + fmt.Sprintf("/boot/%v/rescue", id)
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var rescueResp models.RescueResponse
	err = json.Unmarshal(bytes, &rescueResp)
	if err != nil {
		return nil, err
	}

	return &rescueResp.Rescue, nil
}

func (c *Client) BootRescueSet(ctx context.Context, id int, input *models.RescueSetInput) (*models.Rescue, error) {
	url := c.baseURL + fmt.Sprintf("/boot/%v/rescue", id)

	formData := neturl.Values{}
	formData.Set("os", input.OS)
	if input.Arch > 0 {
		formData.Set("arch", strconv.Itoa(input.Arch))
	}
	if len(input.AuthorizedKey) > 0 {
		formData.Set("authorized_key", input.AuthorizedKey)
	}

	bytes, err := c.doPostFormRequest(ctx, url, formData)
	if err != nil {
		return nil, err
	}

	var rescueResp models.RescueResponse
	err = json.Unmarshal(bytes, &rescueResp)
	if err != nil {
		return nil, err
	}

	return &rescueResp.Rescue, nil
}

func (c *Client) BootRescueDelete(ctx context.Context, id int) (*models.Rescue, error) {
	url := c.baseURL + fmt.Sprintf("/boot/%v/rescue", id)
	bytes, err := c.doDeleteRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var rescueResp models.RescueResponse
	err = json.Unmarshal(bytes, &rescueResp)
	if err != nil {
		return nil, err
	}

	return &rescueResp.Rescue, nil
}

func (c *Client) BootLinuxGet(ctx context.Context, id int) (*models.Linux, error) {
	url := c.baseURL + fmt.Sprintf("/boot/%v/linux", id)
	bytes, err := c.doGetRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var linuxResp models.LinuxResponse
	err = json.Unmarshal(bytes, &linuxResp)
	if err != nil {
		return nil, err
	}

	return &linuxResp.Linux, nil
}

func (c *Client) BootLinuxSet(ctx context.Context, id int, input *models.LinuxSetInput) (*models.Linux, error) {
	url := c.baseURL + fmt.Sprintf("/boot/%v/linux", id)

	formData := neturl.Values{}
	formData.Set("dist", input.Dist)
	if input.Arch > 0 {
		formData.Set("arch", strconv.Itoa(input.Arch))
	}
	if len(input.Lang) > 0 {
		formData.Set("lang", input.Lang)
	}
	if len(input.AuthorizedKey) > 0 {
		formData.Set("authorized_key", input.AuthorizedKey)
	}

	bytes, err := c.doPostFormRequest(ctx, url, formData)
	if err != nil {
		return nil, err
	}

	var linuxResp models.LinuxResponse
	err = json.Unmarshal(bytes, &linuxResp)
	if err != nil {
		return nil, err
	}

	return &linuxResp.Linux, nil
}

func (c *Client) BootLinuxDelete(ctx context.Context, id int) (*models.Linux, error) {
	url := c.baseURL + fmt.Sprintf("/boot/%v/linux", id)
	bytes, err := c.doDeleteRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var linuxResp models.LinuxResponse
	err = json.Unmarshal(bytes, &linuxResp)
	if err != nil {
		return nil, err
	}
	return &linuxResp.Linux, nil
}
