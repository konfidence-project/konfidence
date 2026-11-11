package vectorpromotion_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	. "github.com/konfidence-project/konfidence/internal/kden/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type promotionActionCommand struct {
	action  string
	execute func(*cfg.AppConfig) error
}

var _ = Describe("promotion action responses", func() {
	commands := make([]promotionActionCommand, 0, 2+len(overrideCommands))
	commands = append(commands,
		promotionActionCommand{
			action: "approving vector promotion",
			execute: func(appConfig *cfg.AppConfig) error {
				return executeApproveCommand(appConfig, testVectorPromotionID, FlagProjectID, TestProjectID)
			},
		},
		promotionActionCommand{
			action: "revoking vector promotion approval",
			execute: func(appConfig *cfg.AppConfig) error {
				return executeRevokeApprovalCommand(appConfig, testVectorPromotionID, FlagProjectID, TestProjectID)
			},
		},
	)
	for _, command := range overrideCommands {
		commands = append(commands, promotionActionCommand{
			action: command.action,
			execute: func(appConfig *cfg.AppConfig) error {
				return executeOverrideCommand(command, appConfig, testVectorPromotionID,
					FlagProjectID, TestProjectID, "--reason", "known bad vector")
			},
		})
	}

	for _, command := range commands {
		Describe(command.action, func() {
			var (
				appConfig      *cfg.AppConfig
				responseStatus int
				responseType   string
				responseBody   string
			)

			BeforeEach(func() {
				responseStatus = http.StatusNoContent
				responseType, responseBody = "application/json", ""
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", responseType)
					w.WriteHeader(responseStatus)
					_, _ = w.Write([]byte(responseBody))
				}))
				DeferCleanup(server.Close)
				appConfig = &cfg.AppConfig{
					APIProvider: cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
						return NewTestAuthClient(server.URL + "/api"), nil
					}),
				}
			})

			DescribeTable("reports failures without printing success",
				func(status int, contentType, body, expected string) {
					responseStatus, responseType, responseBody = status, contentType, body
					var err error
					printed := CaptureOutput(func() {
						err = command.execute(appConfig)
					})

					Expect(err).To(MatchError(expected))
					Expect(printed).To(BeEmpty())
				},
				Entry("400 API message", http.StatusBadRequest, "application/json",
					"{\"error\":{\"code\":\"bad_request\",\"message\":\"invalid request\"}}", "invalid request"),
				Entry("JSON envelope with a non-JSON content type", http.StatusForbidden, "text/plain",
					"{\"error\":{\"message\":\"access denied\"}}", "access denied"),
				Entry("non-JSON error body", http.StatusForbidden, "text/plain",
					"access denied", command.action+" returned HTTP 403: access denied"),
				Entry("empty non-JSON body", http.StatusForbidden, "text/plain", "", command.action+" returned HTTP 403: "),
				Entry("missing error envelope", http.StatusNotFound, "application/json", "{}", command.action+" returned HTTP 404: {}"),
				Entry("blank API message", http.StatusConflict, "application/json",
					"{\"error\":{\"message\":\"  \"}}", command.action+" returned HTTP 409: {\"error\":{\"message\":\"  \"}}"),
				Entry("unexpected success status", http.StatusOK, "application/json", "{}", command.action+" returned HTTP 200: {}"),
				Entry("unexpected error status", http.StatusBadGateway, "text/plain",
					"upstream unavailable", command.action+" returned HTTP 502: upstream unavailable"),
			)

			It("preserves authentication errors", func() {
				responseStatus = http.StatusUnauthorized
				var err error
				printed := CaptureOutput(func() {
					err = command.execute(appConfig)
				})

				Expect(errors.Is(err, kdenauth.ErrUnauthorized)).To(BeTrue())
				Expect(err).To(MatchError(command.action + " failed: " + kdenauth.ErrUnauthorized.Error()))
				Expect(printed).To(BeEmpty())
			})

			It("wraps malformed JSON errors from the generated client", func() {
				responseStatus, responseBody = http.StatusForbidden, "{"
				var err error
				printed := CaptureOutput(func() {
					err = command.execute(appConfig)
				})

				var syntaxError *json.SyntaxError
				Expect(errors.As(err, &syntaxError)).To(BeTrue())
				Expect(err).To(MatchError(ContainSubstring(command.action + " failed: ")))
				Expect(printed).To(BeEmpty())
			})
		})
	}
})
