package vectorpromotion_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vectorpromotion"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testApprovePath = "/api/v1/projects/" + testProjectID + "/vectorPromotions/" + testVectorPromotionID + "/approve"

func executeApproveCommand(appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()

	approveCmd, err := vectorpromotion.NewApproveCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())

	approveCmd.Flags().StringP("projectId", "p", "", "")
	_ = approveCmd.MarkFlagRequired("projectId")

	rootCmd := &cobra.Command{
		Use:           "kden",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(approveCmd)
	rootCmd.SetArgs(append([]string{"approve"}, args...))

	return rootCmd.Execute()
}

var _ = Describe("approve command", func() {
	var server *httptest.Server
	var appConfig *cfg.AppConfig
	var requests chan *http.Request

	BeforeEach(func() {
		requests = make(chan *http.Request, 1)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusNoContent)
		}))
		DeferCleanup(server.Close)

		appConfig = &cfg.AppConfig{
			APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
				return newTestAuthClient(server.URL + "/api"), nil
			}),
		}
	})

	It("succeeds on 204", func() {
		err := executeApproveCommand(appConfig, testVectorPromotionID, flagProjectID, testProjectID)

		Expect(err).NotTo(HaveOccurred())
		expectRequest(requests, http.MethodPost, testApprovePath)
	})

	It("fails when no positional argument is provided", func() {
		err := executeApproveCommand(appConfig, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("accepts 1 arg(s), received 0"))
	})

	It("fails when projectId flag is missing", func() {
		err := executeApproveCommand(appConfig, testVectorPromotionID)

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

		err := executeApproveCommand(appConfig, testVectorPromotionID, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed initializing API client"))
		Expect(err.Error()).To(ContainSubstring(initErr.Error()))
		Expect(errors.Is(err, initErr)).To(BeTrue())
	})

	It("wraps transport errors", func() {
		server.Close()

		err := executeApproveCommand(appConfig, testVectorPromotionID, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("approving vector promotion failed"))
	})

	It("returns an error on 403", func() {
		errorMsg := fmt.Sprintf(`access to project %q is not allowed`, testProjectID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(errorResponseBody("forbidden", errorMsg))
		})

		err := executeApproveCommand(appConfig, testVectorPromotionID, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		expectRequest(requests, http.MethodPost, testApprovePath)
	})

	It("returns an error on 404", func() {
		errorMsg := fmt.Sprintf(`vectorPromotion %q not found`, testVectorPromotionID)
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(errorResponseBody("not_found", errorMsg))
		})

		err := executeApproveCommand(appConfig, testVectorPromotionID, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(errorMsg))
		expectRequest(requests, http.MethodPost, testApprovePath)
	})

	It("returns an error on 409", func() {
		errorMsg := "promotion was superseded and is locked"
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write(errorResponseBody("conflict", errorMsg))
		})

		err := executeApproveCommand(appConfig, testVectorPromotionID, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errorMsg))
		expectRequest(requests, http.MethodPost, testApprovePath)
	})

	It("returns an error on unexpected status code", func() {
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(errorResponseBody("internal_server_error", unexpectedErrorMsg))
		})

		err := executeApproveCommand(appConfig, testVectorPromotionID, flagProjectID, testProjectID)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("approving vector promotion returned HTTP %d", http.StatusInternalServerError)))
		Expect(err.Error()).To(ContainSubstring(unexpectedErrorMsg))
		expectRequest(requests, http.MethodPost, testApprovePath)
	})

})
