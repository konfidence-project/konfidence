package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
)

var _ = Describe("Client", func() {
	var srv *httptest.Server
	var client *apiclient.LegacyClient

	BeforeEach(func() {
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		client = apiclient.NewLegacy(srv.URL, 5*time.Second)
	})

	AfterEach(func() { srv.Close() })

	Describe("Healthz", func() {
		It("returns status ok from a live server", func() {
			resp, err := client.Healthz(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Status).To(Equal("ok"))
		})
	})

	Describe("Readyz", func() {
		It("returns status ok from a live server", func() {
			resp, err := client.Readyz(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Status).To(Equal("ok"))
		})
	})

	Describe("error handling", func() {
		It("returns an error when the server responds with non-200", func() {
			errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer errSrv.Close()

			c := apiclient.NewLegacy(errSrv.URL, 5*time.Second)
			_, err := c.Healthz(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("503"))
		})

		It("returns an error when the server is unreachable", func() {
			c := apiclient.NewLegacy("http://127.0.0.1:1", 100*time.Millisecond)
			_, err := c.Healthz(context.Background())
			Expect(err).To(HaveOccurred())
		})
	})
})
