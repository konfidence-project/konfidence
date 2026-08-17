package auth_test

import (
	"context"
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

type loginRequest struct {
	method string
	path   string
}

func executeLoginCommand(appConfig *cfg.AppConfig) error {
	GinkgoHelper()

	loginCommand, err := authcmd.NewLoginCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	rootCommand := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCommand.SetOut(io.Discard)
	rootCommand.SetErr(io.Discard)
	rootCommand.AddCommand(loginCommand)
	rootCommand.SetArgs([]string{"login"})

	return rootCommand.Execute()
}

var _ = Describe("login command", func() {
	It("logs in when the existing session is valid", func() {
		requests := make(chan loginRequest, 1)

		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			requests <- loginRequest{
				method: request.Method,
				path:   request.URL.Path,
			}

			writer.Header().Set("Content-Type", "application/json")
			_, err := writer.Write([]byte(`{
				"name": "Test User",
				"givenName": "Test",
				"familyName": "User",
				"email": "test@example.com",
				"projectRoles": {}
			}`))
			Expect(err).NotTo(HaveOccurred())
		}))
		DeferCleanup(server.Close)

		store := &recordingCookieStore{}
		client := newTestAuthClient(server.URL+"/api", store)

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return client, nil
				},
			),
		}

		Expect(executeLoginCommand(appConfig)).To(Succeed())

		request := <-requests
		Expect(request.method).To(Equal(http.MethodGet))
		Expect(request.path).To(Equal("/api/v1/identity"))
		Expect(store.deleteCalls).To(BeZero())
		Expect(store.saved).To(BeNil())
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

		err := executeLoginCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"failed initializing API client",
		))
		Expect(errors.Is(err, initializationErr)).To(BeTrue())
	})

	It("wraps errors returned by the authentication client", func() {
		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			writer.WriteHeader(http.StatusInternalServerError)
			_, err := writer.Write([]byte("identity unavailable"))
			Expect(err).NotTo(HaveOccurred())
		}))
		DeferCleanup(server.Close)

		client := newTestAuthClient(
			server.URL+"/api",
			&recordingCookieStore{},
		)

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return client, nil
				},
			),
		}

		err := executeLoginCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"authenticating with Konfidence API failed",
		))
		Expect(err.Error()).To(ContainSubstring(
			"checking current session returned HTTP 500",
		))
		Expect(err.Error()).To(ContainSubstring(
			"identity unavailable",
		))
	})

	It("cancels logout when the command context is canceled", func() {
		requestStarted := make(chan struct{}, 1)

		server := httptest.NewServer(http.HandlerFunc(func(
			_ http.ResponseWriter,
			request *http.Request,
		) {
			requestStarted <- struct{}{}
			<-request.Context().Done()
		}))
		DeferCleanup(server.Close)

		client := newTestAuthClient(
			server.URL+"/api",
			&recordingCookieStore{},
		)

		appConfig := &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(
				func() (*kdenauth.Client, error) {
					return client, nil
				},
			),
		}

		loginCommand, err := authcmd.NewLoginCmd(appConfig)
		Expect(err).NotTo(HaveOccurred())

		rootCommand := &cobra.Command{
			Use:           "kden",
			SilenceErrors: true,
			SilenceUsage:  true,
		}
		rootCommand.SetOut(io.Discard)
		rootCommand.SetErr(io.Discard)
		rootCommand.AddCommand(loginCommand)
		rootCommand.SetArgs([]string{"login"})

		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)

		go func() {
			result <- rootCommand.ExecuteContext(ctx)
		}()

		<-requestStarted
		cancel()

		err = <-result
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"authenticating with Konfidence API failed",
		))
		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
	})
})
