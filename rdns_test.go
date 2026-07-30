package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"

	client "github.com/Foresee-Security/hrobot-go"
	. "gopkg.in/check.v1"
)

func (s *ClientSuite) TestRDnsGetListSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/rdns_list.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	rdnsList, err := robotClient.RDnsGetList(context.Background())
	c.Assert(err, IsNil)
	c.Assert(len(rdnsList), Equals, 2)
	c.Assert(rdnsList[0].IP, Equals, testIP)
	c.Assert(rdnsList[1].IP, Equals, testIP2)
}

func (s *ClientSuite) TestRDnsGetListInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.RDnsGetList(context.Background())
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestRDnsGetListServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.RDnsGetList(context.Background())
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestRDnsGetSuccess(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		pwd, pwdErr := os.Getwd()
		c.Assert(pwdErr, IsNil)

		data, readErr := os.ReadFile(pwd + "/test/response/rdns_get.json")
		c.Assert(readErr, IsNil)

		_, err := w.Write(data)
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	rdns, err := robotClient.RDnsGet(context.Background(), testIP)
	c.Assert(err, IsNil)
	c.Assert(rdns.IP, Equals, testIP)
}

func (s *ClientSuite) TestRDnsGetInvalidResponse(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("invalid JSON"))
		c.Assert(err, IsNil)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.RDnsGet(context.Background(), testIP)
	c.Assert(err, Not(IsNil))
}

func (s *ClientSuite) TestRDnsGetServerError(c *C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	robotClient := client.NewBasicAuthClient("user", "pass")
	robotClient.SetBaseURL(ts.URL)

	_, err := robotClient.RDnsGet(context.Background(), testIP)
	c.Assert(err, Not(IsNil))
}
