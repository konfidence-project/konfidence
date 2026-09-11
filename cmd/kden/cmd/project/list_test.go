package project_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/project"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testListPath = "/api/v1/projects"

var testProjectList = apiclient.ProjectList{
	Data: []apiclient.Project{
		{
			Id:   "test-project-id",
			Name: "test-project",
		},
	},
}

func executeListCommand(appConfig *cfg.AppConfig) error {
	GinkgoHelper()

	listCmd, err := project.NewListCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	rootCmd := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(listCmd)
	rootCmd.SetArgs([]string{"list"})

	return rootCmd.Execute()
}

var _ = Describe("list command", func() {
	var server *httptest.Server
	var appConfig *cfg.AppConfig
	var requests chan *http.Request

	BeforeEach(func() {
		requests = make(chan *http.Request, 1)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(testProjectList)
		}))
		DeferCleanup(server.Close)

		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return newTestAuthClient(server.URL + "/api"), nil
			}),
		}
	})

	It("succeeds on 200 and prints the response", func() {
		printed := captureOutput(func() {
			err := executeListCommand(appConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		expectRequest(requests, http.MethodGet, testListPath)
		Expect(printed).To(Equal(toJSON(&testProjectList)))
	})

	It("returns an error when 200 response has no body", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusOK)
		})

		err := executeListCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("projects response did not contain a body"))
		expectRequest(requests, http.MethodGet, testListPath)
	})

	It("returns an error when output formatting fails", func() {
		cfg.Config.Output = "invalid"
		DeferCleanup(func() { cfg.Config.Output = jsonOutputFormat })

		err := executeListCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("formatting projects failed"))
	})

	It("wraps API client initialization errors", func() {
		initErr := errors.New("failed creating API client")
		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return nil, initErr
			}),
		}

		err := executeListCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed initializing API client"))
		Expect(err.Error()).To(ContainSubstring(initErr.Error()))
		Expect(errors.Is(err, initErr)).To(BeTrue())
	})

	It("wraps transport errors", func() {
		server.Close()

		err := executeListCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listing projects failed"))
	})

	It("returns an error on unexpected status code", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(errorResponseBody("internal_server_error", unexpectedErrorMsg))
		})

		err := executeListCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("listing projects returned HTTP %d", http.StatusInternalServerError)))
		Expect(err.Error()).To(ContainSubstring(unexpectedErrorMsg))
		expectRequest(requests, http.MethodGet, testListPath)
	})
})
