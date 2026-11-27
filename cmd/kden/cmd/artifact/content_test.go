package artifact_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/artifact"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/konfidence-project/konfidence/internal/kden/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testContentPath = "/api/v1/projects/" + TestProjectID + "/artifacts/content"

const (
	flagRef = "--ref"

	testArtifactRef = "registry.example.com//test/artifact:1.0.0"
)

var testArtifactContent = apiclient.ArtifactContent{
	Ref:        testArtifactRef,
	Type:       "helm-chart",
	AllowReuse: true,
	Resources: &[]apiclient.ArtifactResource{
		{Name: "artifact-image", Type: "ociImage"},
	},
}

func executeContentCommand(appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()

	contentCmd, err := artifact.NewContentCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	contentCmd.Flags().StringP("projectId", "p", "", "")
	_ = contentCmd.MarkFlagRequired("projectId")
	contentCmd.Flags().String("ref", "", "")
	_ = contentCmd.MarkFlagRequired("ref")

	rootCmd := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(contentCmd)
	rootCmd.SetArgs(append([]string{"content"}, args...))

	return rootCmd.Execute()
}

var _ = Describe("content command", func() {
	var server *httptest.Server
	var appConfig *cfg.AppConfig
	var requests chan *http.Request

	BeforeEach(func() {
		requests = make(chan *http.Request, 1)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(testArtifactContent)
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
			err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)
			Expect(err).NotTo(HaveOccurred())
		})

		request := <-requests
		Expect(request.Method).To(Equal(http.MethodGet))
		Expect(request.URL.Path).To(Equal(testContentPath))
		Expect(request.URL.Query().Get("ref")).To(Equal(testArtifactRef))
		Expect(printed).To(Equal(ToJSON(&testArtifactContent)))
	})

	It("returns an error when 200 response has no body", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusOK)
		})

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("artifact content response did not contain a body"))
		ExpectRequest(requests, http.MethodGet, testContentPath)
	})

	It("returns an error when output formatting fails", func() {
		cfg.Config.Output = "invalid"
		DeferCleanup(func() { cfg.Config.Output = JSONOutputFormat })

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("formatting artifact content response failed"))
	})

	It("fails when projectId flag is missing", func() {
		err := executeContentCommand(appConfig, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "projectId" not set`))
	})

	It("fails when ref flag is missing", func() {
		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "ref" not set`))
	})

	It("wraps API client initialization errors", func() {
		initErr := errors.New("failed creating API client")
		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return nil, initErr
			}),
		}

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed initializing API client"))
		Expect(err.Error()).To(ContainSubstring(initErr.Error()))
		Expect(errors.Is(err, initErr)).To(BeTrue())
	})

	It("wraps transport errors", func() {
		server.Close()

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("getting artifact content failed"))
	})

	It("returns an error on 400", func() {
		errorMsg := fmt.Sprintf("invalid ref %q", testArtifactRef)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(ErrorResponseBody("bad_request", errorMsg))
		})

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testContentPath)
	})

	It("returns an error on 403", func() {
		errorMsg := fmt.Sprintf(`access to project %q is not allowed`, TestProjectID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(ErrorResponseBody("forbidden", errorMsg))
		})

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testContentPath)
	})

	It("returns an error on 404", func() {
		errorMsg := fmt.Sprintf(`artifact %q not found`, testArtifactRef)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(ErrorResponseBody("not_found", errorMsg))
		})

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testContentPath)
	})

	It("returns an error on unexpected status code", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(ErrorResponseBody("internal_server_error", UnexpectedErrorMsg))
		})

		err := executeContentCommand(appConfig, FlagProjectID, TestProjectID, flagRef, testArtifactRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("getting artifact content returned HTTP %d", http.StatusInternalServerError)))
		Expect(err.Error()).To(ContainSubstring(UnexpectedErrorMsg))
		ExpectRequest(requests, http.MethodGet, testContentPath)
	})
})
