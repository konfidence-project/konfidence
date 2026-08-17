package auth_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"

	authcmd "github.com/konfidence-project/konfidence/cmd/kden/cmd/auth"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

type logoutRequest struct {
	method      string
	path        string
	sessionName string
	sessionID   string
}

func executeLogoutCommand(appConfig *cfg.AppConfig) error {
	GinkgoHelper()

	logoutCommand, err := authcmd.NewLogoutCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	rootCommand := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCommand.SetOut(io.Discard)
	rootCommand.SetErr(io.Discard)
	rootCommand.AddCommand(logoutCommand)
	rootCommand.SetArgs([]string{"logout"})

	return rootCommand.Execute()
}

var _ = Describe("logout command", func() {
	It("logs out and removes the persisted session cookie", func() {
		requests := make(chan logoutRequest, 1)

		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			observation := logoutRequest{
				method: request.Method,
				path:   request.URL.Path,
			}

			cookie, err := request.Cookie("kden-session")
			if err == nil {
				observation.sessionName = cookie.Name
				observation.sessionID = cookie.Value
			}

			requests <- observation
			writer.WriteHeader(http.StatusOK)
		}))
		DeferCleanup(server.Close)

		store := &recordingCookieStore{
			loaded: &http.Cookie{
				Name:     "kden-session",
				Value:    "session-id",
				Path:     "/",
				HttpOnly: true,
			},
		}
		client := newTestAuthClient(server.URL+"/api", store)

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return client, nil
				},
			),
		}

		Expect(executeLogoutCommand(appConfig)).To(Succeed())

		request := <-requests
		Expect(request.method).To(Equal(http.MethodPost))
		Expect(request.path).To(Equal("/api/v1/logout"))
		Expect(request.sessionName).To(Equal("kden-session"))
		Expect(request.sessionID).To(Equal("session-id"))
		Expect(store.deleteCalls).To(Equal(1))
		Expect(store.loaded).To(BeNil())
	})

	It("invalidates the local cookie when the API returns unauthorized", func() {
		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			writer.WriteHeader(http.StatusUnauthorized)
		}))
		DeferCleanup(server.Close)

		store := &recordingCookieStore{
			loaded: &http.Cookie{
				Name:  "kden-session",
				Value: "expired-session",
				Path:  "/",
			},
		}
		client := newTestAuthClient(server.URL+"/api", store)

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return client, nil
				},
			),
		}

		Expect(executeLogoutCommand(appConfig)).To(Succeed())
		Expect(store.deleteCalls).To(Equal(1))
		Expect(store.loaded).To(BeNil())
	})

	It("wraps API client initialization errors", func() {
		initializationErr := errors.New("failed creating API client")

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return nil, initializationErr
				},
			),
		}

		err := executeLogoutCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"failed initializing API client",
		))
		Expect(errors.Is(err, initializationErr)).To(BeTrue())
	})

	It("wraps API transport errors", func() {
		server := httptest.NewServer(http.NotFoundHandler())
		endpoint := server.URL + "/api"
		server.Close()

		client := newTestAuthClient(
			endpoint,
			&recordingCookieStore{},
		)

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return client, nil
				},
			),
		}

		err := executeLogoutCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"logging out of Konfidence API failed",
		))
		Expect(err.Error()).To(ContainSubstring(
			"logout of Konfidence API failed",
		))
	})

	It("wraps cookie deletion errors", func() {
		deleteErr := errors.New("keyring unavailable")

		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			writer.WriteHeader(http.StatusOK)
		}))
		DeferCleanup(server.Close)

		store := &recordingCookieStore{
			loaded: &http.Cookie{
				Name:  "kden-session",
				Value: "session-id",
				Path:  "/",
			},
			deleteErr: deleteErr,
		}
		client := newTestAuthClient(server.URL+"/api", store)

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return client, nil
				},
			),
		}

		err := executeLogoutCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"logging out of Konfidence API failed",
		))
		Expect(err.Error()).To(ContainSubstring(
			"removing session cookie failed",
		))
		Expect(errors.Is(err, deleteErr)).To(BeTrue())
		Expect(store.deleteCalls).To(Equal(1))
	})
})
