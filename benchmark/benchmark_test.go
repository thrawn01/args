package benchmark_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/thrawn01/args/benchmark"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test-api Suite")
}

const benchCount = 1000000

var _ = Describe("ParserBench", func() {
	var server http.Handler
	var req *http.Request
	var resp *httptest.ResponseRecorder
	var api *Api

	BeforeEach(func() {
		api = &Api{NewParserBench(OptValues{"test-value": "value"})}
		// And init the server
		server = api.NewServer()
		// Record HTTP responses.
		resp = httptest.NewRecorder()
	})

	Describe("SetOpts", func() {
		Context("when setting new values in a go routine", func() {
			It("should result in data race", func() {
				/*go func() {
					api.Parser.SetOpts(OptValues{"test-value": "new"})
				}()*/
				req, _ = http.NewRequest("GET", "/", nil)
				server.ServeHTTP(resp, req)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body.String()).To(Equal("TestValue: value"))
			})
		})
	})
	Measure("should run efficiently", func(b Benchmarker) {
		b.Time("runtime", func() {
			for i := 0; i < benchCount; i++ {
				resp = httptest.NewRecorder()
				req, _ = http.NewRequest("GET", "/", nil)
				server.ServeHTTP(resp, req)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body.String()).To(Equal("TestValue: value"))
			}
		})
	}, 1)
})

var _ = Describe("ParserBenchMutex", func() {
	var server http.Handler
	var req *http.Request
	var resp *httptest.ResponseRecorder
	var api *Api

	BeforeEach(func() {
		api = &Api{NewParserBenchMutex(OptValues{"test-value": "value"})}
		// And init the server
		server = api.NewServer()
		// Record HTTP responses.
		resp = httptest.NewRecorder()
	})

	Describe("SetOpts", func() {
		Context("when setting new values in a go routine", func() {
			It("should NOT result in data race", func() {
				/*go func() {
					api.Parser.SetOpts(OptValues{"test-value": "new"})
				}()*/
				req, _ = http.NewRequest("GET", "/", nil)
				server.ServeHTTP(resp, req)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body.String()).To(Equal("TestValue: value"))
			})
		})
	})
	Measure("should run efficiently", func(b Benchmarker) {
		b.Time("runtime", func() {
			for i := 0; i < benchCount; i++ {
				resp = httptest.NewRecorder()
				req, _ = http.NewRequest("GET", "/", nil)
				server.ServeHTTP(resp, req)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body.String()).To(Equal("TestValue: value"))
			}
		})
	}, 1)
})

var _ = Describe("ParserBenchRWMutex", func() {
	var server http.Handler
	var req *http.Request
	var resp *httptest.ResponseRecorder
	var api *Api

	BeforeEach(func() {
		api = &Api{NewParserBenchRWMutex(OptValues{"test-value": "value"})}
		// And init the server
		server = api.NewServer()
		// Record HTTP responses.
		resp = httptest.NewRecorder()
	})

	Describe("SetOpts", func() {
		Context("when setting new values in a go routine", func() {
			It("should NOT result in data race", func() {
				/*go func() {
					api.Parser.SetOpts(OptValues{"test-value": "new"})
				}()*/
				req, _ = http.NewRequest("GET", "/", nil)
				server.ServeHTTP(resp, req)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body.String()).To(Equal("TestValue: value"))
			})
		})
	})
	Measure("should run efficiently", func(b Benchmarker) {
		b.Time("runtime", func() {
			for i := 0; i < benchCount; i++ {
				resp = httptest.NewRecorder()
				req, _ = http.NewRequest("GET", "/", nil)
				server.ServeHTTP(resp, req)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body.String()).To(Equal("TestValue: value"))
			}
		})
	}, 1)
})

var _ = Describe("ParserBenchChannel", func() {
	var server http.Handler
	var req *http.Request
	var resp *httptest.ResponseRecorder
	var api *Api

	BeforeSuite(func() {
		api = &Api{NewParserBenchChannel(OptValues{"test-value": "value"})}
		// And init the server
		server = api.NewServer()
	})

	BeforeEach(func() {
		// Record HTTP responses.
		resp = httptest.NewRecorder()
	})

	Describe("SetOpts", func() {
		Context("when setting new values in a go routine", func() {
			It("should NOT result in data race", func() {
				/*go func() {
					api.Parser.SetOpts(OptValues{"test-value": "new"})
				}()*/
				req, _ = http.NewRequest("GET", "/", nil)
				server.ServeHTTP(resp, req)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body.String()).To(Equal("TestValue: value"))
			})
		})
		Measure("should run efficiently", func(b Benchmarker) {
			b.Time("runtime", func() {
				for i := 0; i < benchCount; i++ {
					resp = httptest.NewRecorder()
					req, _ = http.NewRequest("GET", "/", nil)
					server.ServeHTTP(resp, req)
					Expect(resp.Code).To(Equal(200))
					Expect(resp.Body.String()).To(Equal("TestValue: value"))
				}
			})
		}, 1)
	})
})
