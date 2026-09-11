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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testListPath = "/api/v1/projects/" + testProjectID + "/vectorPromotionConfigs"

var testVectorPromotionConfigList = apiclient.VectorPromotionConfigList{
	Data: []apiclient.VectorPromotionConfig{
		{
			Id:         testVectorPromotionConfigID,
			Promotions: []apiclient.VectorPromotion{},
			Source:     apiclient.PromotionSourceReference{},
			Target:     apiclient.PromotionTargetReference{},
		},
	},
}

func executeListCommand(appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()

	listCmd, err := vectorpromotion.NewListCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	listCmd.Flags().StringP("projectId", "p", "", "")
	_ = listCmd.MarkFlagRequired("projectId")

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
			_ = json.NewEncoder(w).Encode(testVectorPromotionConfigList)
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
			err := executeListCommand(appConfig, flagProjectID, testProjectID)
			Expect(err).NotTo(HaveOccurred())
		})

		expectRequest(requests, http.MethodGet, testListPath)
		Expect(printed).To(Equal(toJSON(&testVectorPromotionConfigList)))
	})

	It("returns an error when 200 response has no body", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusOK)
		})

		err := executeListCommand(appConfig, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("vector promotion configs response did not contain a body"))
		expectRequest(requests, http.MethodGet, testListPath)
	})

	It("returns an error when output formatting fails", func() {
		cfg.Config.Output = "invalid"
		DeferCleanup(func() { cfg.Config.Output = jsonOutputFormat })

		err := executeListCommand(appConfig, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("formatting vector promotion configs failed"))
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

		err := executeListCommand(appConfig, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed initializing API client"))
		Expect(err.Error()).To(ContainSubstring(initErr.Error()))
		Expect(errors.Is(err, initErr)).To(BeTrue())
	})

	It("wraps transport errors", func() {
		server.Close()

		err := executeListCommand(appConfig, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listing vector promotion configs failed"))
	})

	It("returns an error on unexpected status code", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(errorResponseBody("internal_server_error", unexpectedErrorMsg))
		})

		err := executeListCommand(appConfig, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("listing vector promotion configs returned HTTP %d", http.StatusInternalServerError)))
		Expect(err.Error()).To(ContainSubstring(unexpectedErrorMsg))
		expectRequest(requests, http.MethodGet, testListPath)
	})
})
