package rpc_test

import (
	"errors"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	. "github.com/cppforlife/bosh-cpi-go/rpc"
	"github.com/cppforlife/bosh-cpi-go/rpc/rpcfakes"
)

var _ = Describe("JSONDispatcher", func() {
	var (
		actionFactory *rpcfakes.FakeActionFactory
		caller        *rpcfakes.FakeCaller
		dispatcher    JSONDispatcher
	)

	BeforeEach(func() {
		actionFactory = &rpcfakes.FakeActionFactory{}

		actionFactory.CreateStub = func(method string, ctx apiv1.CallContext, versions apiv1.ApiVersions) (interface{}, error) {
			Expect(method).To(Equal("fake-action"))
			Expect(ctx).ToNot(BeNil())

			Expect(versions.Contract).To(BeNumerically(">", 0))
			Expect(versions.Stemcell).To(BeNumerically(">", 0))
			Expect(versions.Contract).To(BeNumerically("<=", apiv1.MaxSupportedApiVersion))
			return nil, nil
		}

		caller = &rpcfakes.FakeCaller{}
		logger := boshlog.NewLogger(boshlog.LevelNone)
		dispatcher = NewJSONDispatcher(actionFactory, caller, logger)
	})

	Describe("Dispatch", func() {
		Context("when method is known", func() {
			It("runs action with provided simple arguments", func() {
				dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":["fake-arg"]}`))

				method, ctx, _ := actionFactory.CreateArgsForCall(0)
				Expect(method).To(Equal("fake-action"))
				Expect(ctx).To(Equal(apiv1.CloudPropsImpl{}))

				_, args := caller.CallArgsForCall(0)
				Expect(args).To(Equal([]interface{}{"fake-arg"}))
			})

			It("runs action with provided more complex arguments", func() {
				dispatcher.Dispatch([]byte(`{
          "method":"fake-action",
          "arguments":[
            123,
            "fake-arg",
            [123, "fake-arg"],
            {"fake-arg2-key":"fake-arg2-value"}
          ]
        }`))

				method, ctx, _ := actionFactory.CreateArgsForCall(0)
				Expect(method).To(Equal("fake-action"))
				Expect(ctx).To(Equal(apiv1.CloudPropsImpl{}))

				_, args := caller.CallArgsForCall(0)
				Expect(args).To(Equal([]interface{}{
					float64(123),
					"fake-arg",
					[]interface{}{float64(123), "fake-arg"},
					map[string]interface{}{"fake-arg2-key": "fake-arg2-value"},
				}))
			})

			Context("when the context is specified", func() {
				It("runs action with provided context (without stemcell version)", func() {
					dispatcher.Dispatch([]byte(`{
			  "method":"fake-action",
			  "arguments":[],
			  "context":{"ctx1": "ctx1-val"}
			}`))

					type TestCtx struct {
						Ctx1 string
					}

					method, ctx, _ := actionFactory.CreateArgsForCall(0)
					Expect(method).To(Equal("fake-action"))

					var parsedCtx TestCtx
					err := ctx.As(&parsedCtx)
					Expect(err).ToNot(HaveOccurred())
					Expect(parsedCtx).To(Equal(TestCtx{Ctx1: "ctx1-val"}))

					_, args := caller.CallArgsForCall(0)
					Expect(args).To(Equal([]interface{}{}))
				})

				It("runs action with provided context (with stemcell version)", func() {
					dispatcher.Dispatch([]byte(`{
			  "method":"fake-action",
			  "arguments":[],
			  "context":{"ctx1": "ctx1-val", "vm": {"stemcell": {"api_version": 95}}}
			}`))

					_, _, apiVersion := actionFactory.CreateArgsForCall(0)

					Expect(apiVersion.Stemcell).To(Equal(95))

					_, args := caller.CallArgsForCall(0)
					Expect(args).To(Equal([]interface{}{}))
				})
			})

			Context("when the contract api version is between [1,MaxSupportedApiVersion]", func() {
				testWithApiVersion := func(apiVersion int) {
					It(fmt.Sprintf("runs the action and returns contract version %d", apiVersion), func() {
						dispatcher.Dispatch([]byte(fmt.Sprintf(`{
			  "method":"fake-action",
			  "arguments":[],
			  "api_version": %d
			}`, apiVersion)))

						_, _, apiVersions := actionFactory.CreateArgsForCall(0)
						Expect(apiVersions.Contract).To(Equal(apiVersion))

						_, args := caller.CallArgsForCall(0)
						Expect(args).To(Equal([]interface{}{}))
					})
				}

				for apiVersion := 1; apiVersion <= apiv1.MaxSupportedApiVersion; apiVersion++ {
					testWithApiVersion(apiVersion)
				}
			})

			Context("when running action succeeds", func() {
				It("returns serialized result without including error when result can be serialized", func() {
					caller.CallReturns("fake-result", nil)

					resp := dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":["fake-arg"]}`))
					Expect(resp).To(MatchJSON(`{
            "result": "fake-result",
            "error": null,
            "log": ""
          }`))
				})

				It("returns Bosh::Clouds::CpiError when result cannot be serialized", func() {
					caller.CallReturns(func() {}, nil) // funcs do not serialize

					resp := dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":["fake-arg"]}`))
					Expect(resp).To(MatchJSON(`{
            "result": null,
            "error": {
              "type":"Bosh::Clouds::CpiError",
              "message":"Failed to serialize result",
              "ok_to_retry": false
            },
            "log": ""
          }`))
				})
			})

			Context("when running action fails", func() {
				It("returns error without result when action error is a CloudError", func() {
					caller.CallReturns(nil, &rpcfakes.FakeCloudError{
						TypeStub:  func() string { return "fake-type" },
						ErrorStub: func() string { return "fake-message" },
					})

					resp := dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":["fake-arg"]}`))
					Expect(resp).To(MatchJSON(`{
            "result": null,
            "error": {
              "type":"fake-type",
              "message":"fake-message",
              "ok_to_retry": false
            },
            "log": ""
          }`))
				})

				It("returns error with ok_to_retry=true when action error is a RetryableError and it can be retried", func() {
					caller.CallReturns(nil, &rpcfakes.FakeRetryableError{
						ErrorStub:    func() string { return "fake-message" },
						CanRetryStub: func() bool { return true },
					})

					resp := dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":["fake-arg"]}`))
					Expect(resp).To(MatchJSON(`{
            "result": null,
            "error": {
              "type":"Bosh::Clouds::CloudError",
              "message":"fake-message",
              "ok_to_retry": true
            },
            "log": ""
          }`))
				})

				It("returns error with ok_to_retry=false when action error is a RetryableError and it cannot be retried", func() {
					caller.CallReturns(nil, &rpcfakes.FakeRetryableError{
						ErrorStub:    func() string { return "fake-message" },
						CanRetryStub: func() bool { return false },
					})

					resp := dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":["fake-arg"]}`))
					Expect(resp).To(MatchJSON(`{
            "result": null,
            "error": {
              "type":"Bosh::Clouds::CloudError",
              "message":"fake-message",
              "ok_to_retry": false
            },
            "log": ""
          }`))
				})

				It("returns error without result when action error is neither CloudError or RetryableError", func() {
					caller.CallReturns(nil, errors.New("fake-run-err"))

					resp := dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":["fake-arg"]}`))
					Expect(resp).To(MatchJSON(`{
            "result": null,
            "error": {
              "type":"Bosh::Clouds::CloudError",
              "message":"fake-run-err",
              "ok_to_retry": false
            },
            "log": ""
          }`))
				})
			})
		})

		Context("when method is unknown", func() {
			It("responds with Bosh::Clouds::NotImplemented error", func() {
				actionFactory.CreateReturns(nil, errors.New("fake-err"))

				resp := dispatcher.Dispatch([]byte(`{"method":"fake-action","arguments":[]}`))
				Expect(resp).To(MatchJSON(`{
          "result": null,
          "error": {
            "type":"Bosh::Clouds::NotImplemented",
            "message":"Must call implemented method",
            "ok_to_retry": false
          },
          "log": ""
        }`))
			})
		})

		Context("when method key is missing", func() {
			It("responds with Bosh::Clouds::CpiError error", func() {
				resp := dispatcher.Dispatch([]byte(`{}`))
				Expect(resp).To(MatchJSON(`{
          "result": null,
          "error": {
            "type":"Bosh::Clouds::CpiError",
            "message":"Must provide 'method' key",
            "ok_to_retry": false
          },
          "log": ""
        }`))
			})
		})

		Context("when arguments key is missing", func() {
			It("responds with Bosh::Clouds::CpiError error", func() {
				resp := dispatcher.Dispatch([]byte(`{"method":"fake-action"}`))
				Expect(resp).To(MatchJSON(`{
          "result": null,
          "error": {
            "type":"Bosh::Clouds::CpiError",
            "message":"Must provide 'arguments' key",
            "ok_to_retry": false
          },
          "log": ""
        }`))
			})
		})

		Context("when payload cannot be deserialized", func() {
			It("responds with Bosh::Clouds::CpiError error", func() {
				resp := dispatcher.Dispatch([]byte(`{-}`))
				Expect(resp).To(MatchJSON(`{
          "result": null,
          "error": {
            "type":"Bosh::Clouds::CpiError",
            "message":"Must provide valid JSON payload",
            "ok_to_retry": false
          },
          "log": ""
        }`))
			})
		})

		Context("when the stemcell version is not deserializable", func() {
			It("responds with Bosh::Clouds::CpiError error", func() {
				resp := dispatcher.Dispatch([]byte(`
{
	"method":"fake-action",
	"arguments":[],
	"context":{
		"ctx1": "ctx1-val",
		"vm": {
			"stemcell": {
				"api_version": "FooBar"
			}
		}
	}
}`))
				Expect(resp).To(MatchJSON(`{
          "result": null,
          "error": {
            "type":"Bosh::Clouds::CpiError",
            "message":"Unable to parse stemcell version",
            "ok_to_retry": false
          },
          "log": ""
        }`))
			})
		})

		Context("when the cpi api version is not deserializable", func() {
			It("responds with Bosh::Clouds::CpiError error", func() {
				resp := dispatcher.Dispatch([]byte(`
					{
						"method":"fake-action",
						"arguments":[],
						"api_version": "hello"
					}`))
				Expect(resp).To(MatchJSON(`
					{
						"result": null,
						"error": {
							"type":"Bosh::Clouds::CpiError",
							"message":"Must provide valid JSON payload",
							"ok_to_retry": false
						},
						"log": ""
					}`))
			})
		})

		Context("when the cpi api version is not higher than the max supported version", func() {
			It("responds with Bosh::Clouds::CpiError error", func() {
				requestedApiVersion := apiv1.MaxSupportedApiVersion + 1
				resp := dispatcher.Dispatch([]byte(fmt.Sprintf(`
					{
						"method":"fake-action",
						"arguments":[],
						"api_version": %d
					}`, requestedApiVersion)))
				Expect(resp).To(MatchJSON(fmt.Sprintf(`
					{
						"result": null,
						"error": {
							"type":"Bosh::Clouds::CpiError",
							"message":"Api version specified is %d, max supported is %d",
							"ok_to_retry": false
						},
						"log": ""
					}`, requestedApiVersion, apiv1.MaxSupportedApiVersion)))
			})
		})
	})
})
