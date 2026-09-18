package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	JSONOutputFormat   = "json"
	UnexpectedErrorMsg = "unexpected error occurred"

	TestProjectID = "test-project"
	FlagProjectID = "--projectId"
)

func NewTestAuthClient(endpoint string) *kdenauth.Client {
	ginkgo.GinkgoHelper()

	client, err := kdenauth.NewClient(
		endpoint,
		"",
		&NoopCookieStore{},
		time.Second,
		time.Second,
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return client
}

type NoopCookieStore struct{}

func (s *NoopCookieStore) Load(string) (*http.Cookie, error)   { return nil, nil }
func (s *NoopCookieStore) Save(_ string, _ *http.Cookie) error { return nil }
func (s *NoopCookieStore) Delete(string) error                 { return nil }

func ExpectRequest(requests <-chan *http.Request, method, path string) {
	ginkgo.GinkgoHelper()

	select {
	case request := <-requests:
		gomega.Expect(request.Method).To(gomega.Equal(method))
		gomega.Expect(request.URL.Path).To(gomega.Equal(path))
	case <-time.After(3 * time.Second):
		ginkgo.Fail(fmt.Sprintf("timed out waiting for %s request to %s", method, path))
	}
}

func ErrorResponseBody(code, message string) []byte {
	ginkgo.GinkgoHelper()

	body, err := json.Marshal(apiclient.ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return body
}

func ToJSON(v interface{}) string {
	ginkgo.GinkgoHelper()

	out, err := json.MarshalIndent(v, "", "  ")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return string(out)
}

func CaptureOutput(fn func()) string {
	ginkgo.GinkgoHelper()

	r, w, err := os.Pipe()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	original := os.Stdout
	os.Stdout = w

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			_ = w.Close()
			os.Stdout = original
		})
	}
	defer cleanup()

	fn()
	cleanup()

	output, err := io.ReadAll(r)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return string(output)
}
