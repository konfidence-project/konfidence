package vector_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vector"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/konfidence-project/konfidence/internal/kden/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testDiffPath = "/api/v1/projects/" + TestProjectID + "/vectors/diff"

const (
	flagLeft  = "--left"
	flagRight = "--right"

	testDiffLeftRef  = "registry.example.com//test/vector:1.0.0"
	testDiffRightRef = "registry.example.com//test/vector:2.0.0"
)

var testVectorDiff = func() apiclient.VectorDiff {
	leftVersion := "1.0.0"
	rightVersion := "2.0.0"
	return apiclient.VectorDiff{
		Left:  testDiffLeftRef,
		Right: testDiffRightRef,
		Components: []apiclient.VectorComponentDiff{
			{
				ComponentName: "comp-a",
				ChangeType:    apiclient.Changed,
				LeftVersion:   &leftVersion,
				RightVersion:  &rightVersion,
			},
		},
	}
}()

func executeDiffCommand(appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()

	diffCmd, err := vector.NewDiffCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	diffCmd.Flags().StringP("projectId", "p", "", "")
	_ = diffCmd.MarkFlagRequired("projectId")
	diffCmd.Flags().Bool("include-unchanged", false, "")
	diffCmd.Flags().String("left", "", "")
	_ = diffCmd.MarkFlagRequired("left")
	diffCmd.Flags().String("right", "", "")
	_ = diffCmd.MarkFlagRequired("right")

	rootCmd := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(diffCmd)
	rootCmd.SetArgs(append([]string{"diff"}, args...))

	return rootCmd.Execute()
}

var _ = Describe("diff command", func() {
	var server *httptest.Server
	var appConfig *cfg.AppConfig
	var requests chan *http.Request

	BeforeEach(func() {
		requests = make(chan *http.Request, 1)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(testVectorDiff)
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
			err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)
			Expect(err).NotTo(HaveOccurred())
		})

		request := <-requests
		Expect(request.Method).To(Equal(http.MethodGet))
		Expect(request.URL.Path).To(Equal(testDiffPath))
		Expect(request.URL.Query().Get("left")).To(Equal(testDiffLeftRef))
		Expect(request.URL.Query().Get("right")).To(Equal(testDiffRightRef))
		Expect(printed).To(Equal(ToJSON(&testVectorDiff)))
	})

	It("passes includeUnchanged when the flag is set", func() {
		printed := CaptureOutput(func() {
			err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef, "--include-unchanged")
			Expect(err).NotTo(HaveOccurred())
		})

		request := <-requests
		Expect(request.URL.Path).To(Equal(testDiffPath))
		Expect(request.URL.Query().Get("includeUnchanged")).To(Equal("true"))
		Expect(printed).To(Equal(ToJSON(&testVectorDiff)))
	})

	It("omits includeUnchanged when the flag is not set", func() {
		CaptureOutput(func() {
			err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)
			Expect(err).NotTo(HaveOccurred())
		})

		request := <-requests
		Expect(request.URL.Query().Has("includeUnchanged")).To(BeFalse())
	})

	It("returns an error when 200 response has no body", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusOK)
		})

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("vector diff response did not contain a body"))
		ExpectRequest(requests, http.MethodGet, testDiffPath)
	})

	It("returns an error when output formatting fails", func() {
		cfg.Config.Output = "invalid"
		DeferCleanup(func() { cfg.Config.Output = JSONOutputFormat })

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("formatting vector diff response failed"))
	})

	It("fails when projectId flag is missing", func() {
		err := executeDiffCommand(appConfig, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "projectId" not set`))
	})

	It("fails when left and right flags are missing", func() {
		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "left", "right" not set`))
	})

	It("wraps API client initialization errors", func() {
		initErr := errors.New("failed creating API client")
		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return nil, initErr
			}),
		}

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed initializing API client"))
		Expect(err.Error()).To(ContainSubstring(initErr.Error()))
		Expect(errors.Is(err, initErr)).To(BeTrue())
	})

	It("wraps transport errors", func() {
		server.Close()

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("getting vector diff failed"))
	})

	It("returns an error on 400", func() {
		errorMsg := fmt.Sprintf("invalid ref %q", testDiffLeftRef)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(ErrorResponseBody("bad_request", errorMsg))
		})

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testDiffPath)
	})

	It("returns an error on 403", func() {
		errorMsg := fmt.Sprintf(`access to project %q is not allowed`, TestProjectID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(ErrorResponseBody("forbidden", errorMsg))
		})

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testDiffPath)
	})

	It("returns an error on 404", func() {
		errorMsg := fmt.Sprintf(`vector %q not found`, testDiffLeftRef)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(ErrorResponseBody("not_found", errorMsg))
		})

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testDiffPath)
	})

	It("returns an error on unexpected status code", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(ErrorResponseBody("internal_server_error", UnexpectedErrorMsg))
		})

		err := executeDiffCommand(appConfig, FlagProjectID, TestProjectID, flagLeft, testDiffLeftRef, flagRight, testDiffRightRef)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("getting vector diff returned HTTP %d", http.StatusInternalServerError)))
		Expect(err.Error()).To(ContainSubstring(UnexpectedErrorMsg))
		ExpectRequest(requests, http.MethodGet, testDiffPath)
	})
})
