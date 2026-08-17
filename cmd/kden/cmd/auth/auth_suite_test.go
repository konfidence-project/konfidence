package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/adrg/xdg"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Cmd Suite")
}

// The kden config file lives in the XDG config dir. Point it at a per-suite
// temp dir so parallel test packages never race on creating the real one.
var _ = BeforeSuite(func() {
	GinkgoT().Setenv("XDG_CONFIG_HOME", GinkgoT().TempDir())
	xdg.Reload()
})

type recordingCookieStore struct {
	loaded      *http.Cookie
	saved       *http.Cookie
	loadErr     error
	saveErr     error
	deleteErr   error
	deleteCalls int
}

func (s *recordingCookieStore) Load(string) (*http.Cookie, error) {
	return s.loaded, s.loadErr
}

func (s *recordingCookieStore) Save(
	_ string,
	cookie *http.Cookie,
) error {
	s.saved = cookie
	return s.saveErr
}

func (s *recordingCookieStore) Delete(string) error {
	s.deleteCalls++
	s.loaded = nil
	return s.deleteErr
}

func newTestAuthClient(
	endpoint string,
	store kdenauth.CookieStore,
) *kdenauth.Client {
	GinkgoHelper()

	client, err := kdenauth.NewClient(
		endpoint,
		store,
		time.Second,
		time.Second,
	)
	Expect(err).NotTo(HaveOccurred())

	return client
}
