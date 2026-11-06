package vectorpromotion_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vectorpromotion"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/konfidence-project/konfidence/internal/kden/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

const testRevokeApprovalPath = "/api/v1/projects/" + TestProjectID + "/vectorPromotions/" +
	testVectorPromotionID + "/revoke-approval"

func executeRevokeApprovalCommand(appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()
	revokeCmd, err := vectorpromotion.NewRevokeApprovalCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())
	revokeCmd.Flags().StringP("projectId", "p", "", "")
	_ = revokeCmd.MarkFlagRequired("projectId")

	rootCmd := &cobra.Command{Use: "kden", SilenceErrors: true, SilenceUsage: true}
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(revokeCmd)
	rootCmd.SetArgs(append([]string{"revoke-approval"}, args...))
	return rootCmd.Execute()
}

var _ = Describe("revoke-approval command", func() {
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
				return NewTestAuthClient(server.URL + "/api"), nil
			}),
		}
	})

	It("succeeds on 204 and prints a success message", func() {
		var err error
		printed := CaptureOutput(func() {
			err = executeRevokeApprovalCommand(appConfig, testVectorPromotionID, FlagProjectID, TestProjectID)
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(printed).To(Equal(fmt.Sprintf(
			"Approval of vector promotion %s revoked successfully", testVectorPromotionID,
		)))
		ExpectRequest(requests, http.MethodPost, testRevokeApprovalPath)
	})

	It("fails when the projectId flag is missing", func() {
		err := executeRevokeApprovalCommand(appConfig, testVectorPromotionID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`required flag(s) "projectId" not set`))
	})

	It("returns the API conflict", func() {
		errorMsg := "promotion gates are already satisfied"
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write(ErrorResponseBody("conflict", errorMsg))
		})

		err := executeRevokeApprovalCommand(appConfig, testVectorPromotionID, FlagProjectID, TestProjectID)
		Expect(err).To(MatchError(errorMsg))
		ExpectRequest(requests, http.MethodPost, testRevokeApprovalPath)
	})

	It("does not print success when there is no approval to revoke", func() {
		errorMsg := "promotion is not approved; nothing to revoke"
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write(ErrorResponseBody("conflict", errorMsg))
		})

		var err error
		printed := CaptureOutput(func() {
			err = executeRevokeApprovalCommand(appConfig, testVectorPromotionID, FlagProjectID, TestProjectID)
		})
		Expect(err).To(MatchError(errorMsg))
		Expect(printed).To(BeEmpty())
	})
})
