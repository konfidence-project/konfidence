package project_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"os"

	"github.com/adrg/xdg"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Project Cmd Suite")
}

const (
	jsonOutputFormat   = "json"
	unexpectedErrorMsg = "unexpected error occurred"
)

// The kden config file lives in the XDG config dir. Point it at a per-suite
// temp dir so parallel test packages never race on creating the real one.
var _ = BeforeSuite(func() {
	GinkgoT().Setenv("XDG_CONFIG_HOME", GinkgoT().TempDir())
	xdg.Reload()
	cfg.Config.Output = jsonOutputFormat
})

func newTestAuthClient(endpoint string) *kdenauth.Client {
	GinkgoHelper()

	client, err := kdenauth.NewClient(
		endpoint,
		"",
		&noopCookieStore{},
		time.Second,
		time.Second,
	)
	Expect(err).NotTo(HaveOccurred())

	return client
}

type noopCookieStore struct{}

func (s *noopCookieStore) Load(string) (*http.Cookie, error)   { return nil, nil }
func (s *noopCookieStore) Save(_ string, _ *http.Cookie) error { return nil }
func (s *noopCookieStore) Delete(string) error                 { return nil }

func expectRequest(requests <-chan *http.Request, method, path string) {
	GinkgoHelper()

	request := <-requests
	Expect(request.Method).To(Equal(method))
	Expect(request.URL.Path).To(Equal(path))
}

func errorResponseBody(code, message string) []byte {
	return []byte(fmt.Sprintf(`{"error":{"code":%q,"message":%q}}`, code, message))
}

func toJSON(v interface{}) string {
	GinkgoHelper()

	out, err := json.MarshalIndent(v, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	return string(out)
}

func captureOutput(fn func()) string {
	GinkgoHelper()

	r, w, err := os.Pipe()
	Expect(err).NotTo(HaveOccurred())

	original := os.Stdout
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = original

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	return string(buf[:n])
}
