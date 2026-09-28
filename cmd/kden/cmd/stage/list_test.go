package stage_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/stage"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/konfidence-project/konfidence/internal/kden/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testListPath = "/api/v1/projects/" + TestProjectID + "/stages"

var testStageList = apiclient.StageList{
	Data: []apiclient.Stage{
		{
			Id:          "test-stage-id",
			LandscapeId: "test-landscape-id",
			Name:        "test-stage",
		},
	},
}

func executeListCommand(appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()

	listCmd, err := stage.NewListCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	listCmd.Flags().StringP("projectId", "p", "", "")
	_ = listCmd.MarkFlagRequired("projectId")
	listCmd.Flags().StringP("landscapeId", "l", "", "")

	rootCmd := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(listCmd)
	rootCmd.SetArgs(append([]string{"list"}, args...))

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
			_ = json.NewEncoder(w).Encode(testStageList)
		}))
		DeferCleanup(server.Close)

		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return NewTestAuthClient(server.URL + "/api"), nil
			}),
		}
	})

	It("succeeds on 200 and prints the response", func() {
		printed := CaptureOutput(func() {
			err := executeListCommand(appConfig, FlagProjectID, TestProjectID)
			Expect(err).NotTo(HaveOccurred())
		})

		ExpectRequest(requests, http.MethodGet, testListPath)
		Expect(printed).To(Equal(ToJSON(&testStageList)))
	})

	It("succeeds with optional landscapeId filter", func() {
		printed := CaptureOutput(func() {
			err := executeListCommand(appConfig, FlagProjectID, TestProjectID, "--landscapeId", "test-landscape-id")
			Expect(err).NotTo(HaveOccurred())
		})

		request := <-requests
		Expect(request.Method).To(Equal(http.MethodGet))
		Expect(request.URL.Path).To(Equal(testListPath))
		Expect(request.URL.Query().Get("landscapeId")).To(Equal("test-landscape-id"))
		Expect(printed).To(Equal(ToJSON(&testStageList)))
	})

	It("returns an error when 200 response has no body", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusOK)
		})

		err := executeListCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("stages response did not contain a body"))
		ExpectRequest(requests, http.MethodGet, testListPath)
	})

	It("returns an error when output formatting fails", func() {
		cfg.Config.Output = "invalid"
		DeferCleanup(func() { cfg.Config.Output = JSONOutputFormat })

		err := executeListCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("formatting stages failed"))
	})

	It("fails when projectId flag is missing", func() {
		err := executeListCommand(appConfig)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "projectId" not set`))
	})

	It("wraps API client initialization errors", func() {
		initErr := errors.New("failed creating API client")
		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return nil, initErr
			}),
		}

		err := executeListCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed initializing API client"))
		Expect(err.Error()).To(ContainSubstring(initErr.Error()))
		Expect(errors.Is(err, initErr)).To(BeTrue())
	})

	It("wraps transport errors", func() {
		server.Close()

		err := executeListCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listing stages failed"))
	})

	It("returns an error on 403", func() {
		errorMsg := fmt.Sprintf(`access to project %q is not allowed`, TestProjectID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(ErrorResponseBody("forbidden", errorMsg))
		})

		err := executeListCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testListPath)
	})

	It("returns an error on 404", func() {
		errorMsg := fmt.Sprintf(`project %q not found`, TestProjectID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(ErrorResponseBody("not_found", errorMsg))
		})

		err := executeListCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testListPath)
	})

	It("returns an error on unexpected status code", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(ErrorResponseBody("internal_server_error", UnexpectedErrorMsg))
		})

		err := executeListCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("listing stages returned HTTP %d", http.StatusInternalServerError)))
		Expect(err.Error()).To(ContainSubstring(UnexpectedErrorMsg))
		ExpectRequest(requests, http.MethodGet, testListPath)
	})
})
