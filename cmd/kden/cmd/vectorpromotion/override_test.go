package vectorpromotion_test

import (
	"encoding/json"
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

type overrideCommand struct {
	name    string
	newCmd  func(*cfg.AppConfig) (*cobra.Command, error)
	path    string
	success string
	action  string
}

var overrideCommands = []overrideCommand{
	{
		name:    "reject",
		newCmd:  vectorpromotion.NewRejectCmd,
		path:    "/api/v1/projects/" + TestProjectID + "/vectorPromotions/" + testVectorPromotionID + "/reject",
		success: fmt.Sprintf("Vector promotion %s rejected successfully", testVectorPromotionID),
		action:  "rejecting vector promotion",
	},
	{
		name:    "bypass-gates",
		newCmd:  vectorpromotion.NewBypassGatesCmd,
		path:    "/api/v1/projects/" + TestProjectID + "/vectorPromotions/" + testVectorPromotionID + "/bypassGates",
		success: fmt.Sprintf("Gates of vector promotion %s bypassed successfully", testVectorPromotionID),
		action:  "bypassing vector promotion gates",
	},
}

func executeOverrideCommand(command overrideCommand, appConfig *cfg.AppConfig, args ...string) error {
	GinkgoHelper()

	cmd, err := command.newCmd(appConfig)
	Expect(err).NotTo(HaveOccurred())
	cmd.Flags().StringP("projectId", "p", "", "")
	_ = cmd.MarkFlagRequired("projectId")

	rootCmd := &cobra.Command{Use: "kden", SilenceErrors: true, SilenceUsage: true}
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs(append([]string{command.name}, args...))
	return rootCmd.Execute()
}

var _ = Describe("override commands", func() {
	for _, command := range overrideCommands {
		Describe(command.name, func() {
			var (
				server    *httptest.Server
				appConfig *cfg.AppConfig
				requests  chan *http.Request
				bodies    chan map[string]any
				status    int
				response  []byte
			)

			BeforeEach(func() {
				requests = make(chan *http.Request, 1)
				bodies = make(chan map[string]any, 1)
				status, response = http.StatusNoContent, nil
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body := map[string]any{}
					_ = json.NewDecoder(r.Body).Decode(&body)
					requests <- r
					bodies <- body
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write(response)
				}))
				DeferCleanup(server.Close)

				appConfig = &cfg.AppConfig{
					APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
						return NewTestAuthClient(server.URL + "/api"), nil
					}),
				}
			})

			It("sends the reason and prints a success message on 204", func() {
				var err error
				printed := CaptureOutput(func() {
					err = executeOverrideCommand(command, appConfig, testVectorPromotionID,
						FlagProjectID, TestProjectID, "--reason", "known bad vector")
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(printed).To(Equal(command.success))
				ExpectRequest(requests, http.MethodPost, command.path)
				Expect(<-bodies).To(Equal(map[string]any{"reason": "known bad vector"}))
			})

			It("requires a reason", func() {
				err := executeOverrideCommand(command, appConfig, testVectorPromotionID, FlagProjectID, TestProjectID)

				Expect(err).To(MatchError(ContainSubstring(`required flag(s) "reason" not set`)))
			})

			DescribeTable("returns the API error message",
				func(code int, errorCode string) {
					status, response = code, ErrorResponseBody(errorCode, "promotion is already rejected")

					err := executeOverrideCommand(command, appConfig, testVectorPromotionID,
						FlagProjectID, TestProjectID, "-r", "known bad vector")

					Expect(err).To(MatchError("promotion is already rejected"))
				},
				Entry("400", http.StatusBadRequest, "bad_request"),
				Entry("403", http.StatusForbidden, "forbidden"),
				Entry("404", http.StatusNotFound, "not_found"),
				Entry("409", http.StatusConflict, "conflict"),
			)

			It("reports unexpected status codes with the body", func() {
				status, response = http.StatusInternalServerError, ErrorResponseBody("internal_server_error", UnexpectedErrorMsg)

				err := executeOverrideCommand(command, appConfig, testVectorPromotionID,
					FlagProjectID, TestProjectID, "-r", "known bad vector")

				Expect(err).To(MatchError(ContainSubstring(
					fmt.Sprintf("%s returned HTTP %d", command.action, http.StatusInternalServerError))))
				Expect(err).To(MatchError(ContainSubstring(UnexpectedErrorMsg)))
			})
		})
	}
})
