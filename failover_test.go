package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"

	client "github.com/Foresee-Security/hrobot-go"
	. "gopkg.in/check.v1"
)

func (s *ClientSuite) TestFailoverGetListSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/failover_list.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	failoverList, err := robotClient.FailoverGetList(context.Background())
	c.Assert(err, IsNil)
	c.Assert(len(failoverList), Equals, 1)
	c.Assert(failoverList[0].IP, Equals, testIP)
}

func (s *ClientSuite) TestFailoverGetListInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.FailoverGetList(context.Background())
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestFailoverGetListServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.FailoverGetList(context.Background())
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestFailoverGetSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/failover_get.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	failover, err := robotClient.FailoverGet(context.Background(), testIP)
	c.Assert(err, IsNil)
	c.Assert(failover.IP, Equals, testIP)
}

func (s *ClientSuite) TestFailoverGetInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.FailoverGet(context.Background(), testIP)
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestFailoverGetServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.FailoverGet(context.Background(), testIP)
	c.Assert(err, Not(IsNil))
}
