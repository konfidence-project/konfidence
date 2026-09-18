package vectorpromotion_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vectorpromotion"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/konfidence-project/konfidence/internal/kden/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testVectorPromotionConfigID = "test-vector-promotion-config-id"
const testGetPath = "/api/v1/projects/" + TestProjectID + "/vectorPromotionConfigs/" + testVectorPromotionConfigID

var testVectorPromotionConfig = apiclient.VectorPromotionConfig{
	Id:         testVectorPromotionConfigID,
	Promotions: []apiclient.VectorPromotion{},
	Source:     apiclient.PromotionSourceReference{},
	Target:     apiclient.PromotionTargetReference{},
}

func executeGetCommand(appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()

	getCmd, err := vectorpromotion.NewGetCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	getCmd.Flags().StringP("projectId", "p", "", "")
	_ = getCmd.MarkFlagRequired("projectId")

	getCmd.Flags().StringP("vectorPromotionConfigId", "v", "", "")
	_ = getCmd.MarkFlagRequired("vectorPromotionConfigId")

	rootCmd := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(getCmd)
	rootCmd.SetArgs(append([]string{"get"}, args...))

	return rootCmd.Execute()
}

var _ = Describe("get command", func() {
	var server *httptest.Server
	var appConfig *cfg.AppConfig
	var requests chan *http.Request

	BeforeEach(func() {
		requests = make(chan *http.Request, 1)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(testVectorPromotionConfig)
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
			err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)
			Expect(err).NotTo(HaveOccurred())
		})

		ExpectRequest(requests, http.MethodGet, testGetPath)
		Expect(printed).To(Equal(ToJSON(&testVectorPromotionConfig)))
	})

	It("fails when projectId flag is missing", func() {
		err := executeGetCommand(appConfig, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "projectId" not set`))
	})

	It("fails when vectorPromotionConfigId flag is missing", func() {
		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "vectorPromotionConfigId" not set`))
	})

	It("wraps API client initialization errors", func() {
		initErr := errors.New("failed creating API client")
		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return nil, initErr
			}),
		}

		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed initializing API client"))
		Expect(err.Error()).To(ContainSubstring(initErr.Error()))
		Expect(errors.Is(err, initErr)).To(BeTrue())
	})

	It("wraps transport errors", func() {
		server.Close()

		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("getting vector promotion config failed"))
	})

	It("returns an error when 200 response has no body", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusOK)
		})

		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("vector promotion config response did not contain a body"))
		ExpectRequest(requests, http.MethodGet, testGetPath)
	})

	It("returns an error when output formatting fails", func() {
		cfg.Config.Output = "invalid"
		DeferCleanup(func() { cfg.Config.Output = JSONOutputFormat })

		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("formatting vector promotion config failed"))
	})

	It("returns an error on 403", func() {
		errorMsg := fmt.Sprintf(`access to project %q is not allowed`, TestProjectID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(ErrorResponseBody("forbidden", errorMsg))
		})

		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testGetPath)
	})

	It("returns an error on 404", func() {
		errorMsg := fmt.Sprintf(`project %q not found`, TestProjectID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(ErrorResponseBody("not_found", errorMsg))
		})

		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		ExpectRequest(requests, http.MethodGet, testGetPath)
	})

	It("returns an error on unexpected status code", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(ErrorResponseBody("internal_server_error", UnexpectedErrorMsg))
		})

		err := executeGetCommand(appConfig, FlagProjectID, TestProjectID, flagVectorPromotionConfigID, testVectorPromotionConfigID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("getting vector promotion config returned HTTP %d", http.StatusInternalServerError)))
		Expect(err.Error()).To(ContainSubstring(UnexpectedErrorMsg))
		ExpectRequest(requests, http.MethodGet, testGetPath)
	})
})
