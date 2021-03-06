package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/config/runtime"
	"github.com/containous/traefik/v2/pkg/config/static"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_TCP(t *testing.T) {
	type expected struct {
		statusCode int
		nextPage   string
		jsonFile   string
	}

	testCases := []struct {
		desc     string
		path     string
		conf     runtime.Configuration
		expected expected
	}{
		{
			desc: "all TCP routers, but no config",
			path: "/api/tcp/routers",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/tcprouters-empty.json",
			},
		},
		{
			desc: "all TCP routers",
			path: "/api/tcp/routers",
			conf: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"test@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar.other`)",
							TLS: &dynamic.RouterTCPTLSConfig{
								Passthrough: false,
							},
						},
						Status: runtime.StatusEnabled,
					},
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
						Status: runtime.StatusWarning,
					},
					"foo@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/tcprouters.json",
			},
		},
		{
			desc: "all TCP routers, pagination, 1 res per page, want page 2",
			path: "/api/tcp/routers?page=2&per_page=1",
			conf: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
					"baz@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`toto.bar`)",
						},
					},
					"test@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar.other`)",
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "3",
				jsonFile:   "testdata/tcprouters-page2.json",
			},
		},
		{
			desc: "one TCP router by id",
			path: "/api/tcp/routers/bar@myprovider",
			conf: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				jsonFile:   "testdata/tcprouter-bar.json",
			},
		},
		{
			desc: "one TCP router by id, that does not exist",
			path: "/api/tcp/routers/foo@myprovider",
			conf: runtime.Configuration{
				TCPRouters: map[string]*runtime.TCPRouterInfo{
					"bar@myprovider": {
						TCPRouter: &dynamic.TCPRouter{
							EntryPoints: []string{"web"},
							Service:     "foo-service@myprovider",
							Rule:        "Host(`foo.bar`)",
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "one TCP router by id, but no config",
			path: "/api/tcp/routers/bar@myprovider",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "all tcp services, but no config",
			path: "/api/tcp/services",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/tcpservices-empty.json",
			},
		},
		{
			desc: "all tcp services",
			path: "/api/tcp/services",
			conf: runtime.Configuration{
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"bar@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
						Status: runtime.StatusEnabled,
					},
					"baz@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusWarning,
					},
					"foz@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
						Status: runtime.StatusDisabled,
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "1",
				jsonFile:   "testdata/tcpservices.json",
			},
		},
		{
			desc: "all tcp services, 1 res per page, want page 2",
			path: "/api/tcp/services?page=2&per_page=1",
			conf: runtime.Configuration{
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"bar@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
					},
					"baz@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.2:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider"},
					},
					"test@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.3:2345",
									},
								},
							},
						},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				nextPage:   "3",
				jsonFile:   "testdata/tcpservices-page2.json",
			},
		},
		{
			desc: "one tcp service by id",
			path: "/api/tcp/services/bar@myprovider",
			conf: runtime.Configuration{
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"bar@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusOK,
				jsonFile:   "testdata/tcpservice-bar.json",
			},
		},
		{
			desc: "one tcp service by id, that does not exist",
			path: "/api/tcp/services/nono@myprovider",
			conf: runtime.Configuration{
				TCPServices: map[string]*runtime.TCPServiceInfo{
					"bar@myprovider": {
						TCPService: &dynamic.TCPService{
							LoadBalancer: &dynamic.TCPLoadBalancerService{
								Servers: []dynamic.TCPServer{
									{
										Address: "127.0.0.1:2345",
									},
								},
							},
						},
						UsedBy: []string{"foo@myprovider", "test@myprovider"},
					},
				},
			},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
		{
			desc: "one tcp service by id, but no config",
			path: "/api/tcp/services/foo@myprovider",
			conf: runtime.Configuration{},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			rtConf := &test.conf
			handler := New(static.Configuration{API: &static.API{}, Global: &static.Global{}}, rtConf)
			router := mux.NewRouter()
			handler.Append(router)

			server := httptest.NewServer(router)

			resp, err := http.DefaultClient.Get(server.URL + test.path)
			require.NoError(t, err)

			assert.Equal(t, test.expected.nextPage, resp.Header.Get(nextPageHeader))

			require.Equal(t, test.expected.statusCode, resp.StatusCode)

			if test.expected.jsonFile == "" {
				return
			}

			assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")

			contents, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)

			err = resp.Body.Close()
			require.NoError(t, err)

			if *updateExpected {
				var results interface{}
				err := json.Unmarshal(contents, &results)
				require.NoError(t, err)

				newJSON, err := json.MarshalIndent(results, "", "\t")
				require.NoError(t, err)

				err = ioutil.WriteFile(test.expected.jsonFile, newJSON, 0644)
				require.NoError(t, err)
			}

			data, err := ioutil.ReadFile(test.expected.jsonFile)
			require.NoError(t, err)
			assert.JSONEq(t, string(data), string(contents))
		})
	}
}
