package client_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	client "github.com/Foresee-Security/hrobot-go"
	"github.com/Foresee-Security/hrobot-go/models"
	. "gopkg.in/check.v1"
)

func (s *ClientSuite) TestServerGetListSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/server_list.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	servers, err := robotClient.ServerGetList(context.Background())
	c.Assert(err, IsNil)
	c.Assert(len(servers), Equals, 2)
	c.Assert(servers[0].ServerNumber, Equals, testServerID)
	c.Assert(servers[1].ServerNumber, Equals, testServerID2)
}

func (s *ClientSuite) TestServerGetListInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.ServerGetList(context.Background())
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestServerGetListServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.ServerGetList(context.Background())
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestServerGetSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/server_get.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	server, err := robotClient.ServerGet(context.Background(), testServerID)
	c.Assert(err, IsNil)
	c.Assert(server.ServerNumber, Equals, testServerID)
}

func (s *ClientSuite) TestServerGetInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.ServerGet(context.Background(), testServerID)
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestServerGetServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.ServerGet(context.Background(), testServerID)
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestServerGetNotFound(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/server_get_404.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.ServerGet(context.Background(), testServerID)
	c.Assert(err, NotNil)
}

func (s *ClientSuite) TestServerSetNameSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqContentType := r.Header.Get("Content-Type")
		c.Assert(reqContentType, Equals, "application/x-www-form-urlencoded")

		body, bodyErr := io.ReadAll(r.Body)
		c.Assert(bodyErr, IsNil)
		c.Assert(string(body), Equals, "server_name=mongodb-prod-px62-nvme-hetzner-nbg1-dc1-123456")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/server_get.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	input := &models.ServerSetNameInput{
		Name: "mongodb-prod-px62-nvme-hetzner-nbg1-dc1-123456",
	}

	server, err := robotClient.ServerSetName(context.Background(), testServerID, input)
	c.Assert(err, IsNil)
	c.Assert(server.ServerNumber, Equals, testServerID)
}

func (s *ClientSuite) TestServerSetNameInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqContentType := r.Header.Get("Content-Type")
		c.Assert(reqContentType, Equals, "application/x-www-form-urlencoded")

		body, bodyErr := io.ReadAll(r.Body)
		c.Assert(bodyErr, IsNil)
		c.Assert(string(body), Equals, "server_name=mongodb-prod-px62-nvme-hetzner-nbg1-dc1-123456")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	input := &models.ServerSetNameInput{
		Name: "mongodb-prod-px62-nvme-hetzner-nbg1-dc1-123456",
	}

	_, err := robotClient.ServerSetName(context.Background(), testServerID, input)
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestServerSetNameServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	input := &models.ServerSetNameInput{
		Name: "mongodb-prod-px62-nvme-hetzner-nbg1-dc1-123456",
	}

	_, err := robotClient.ServerSetName(context.Background(), testServerID, input)
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestServerReverseSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/server_reverse.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	cancellation, err := robotClient.ServerReverse(context.Background(), testServerID)
	c.Assert(err, IsNil)
	c.Assert(cancellation.ServerNumber, Equals, testServerID)
	c.Assert(cancellation.CancellationDate, Equals, "2014-04-15")
}

func (s *ClientSuite) TestServerReverseInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.ServerReverse(context.Background(), testServerID)
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestServerReverseServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.ServerReverse(context.Background(), testServerID)
	c.Assert(err, Not(IsNil))
}
