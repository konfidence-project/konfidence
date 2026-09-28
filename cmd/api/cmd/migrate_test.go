package cmd

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("migrate command", func() {
	BeforeEach(func() {
		// cfg is filled from API_* env vars at init; a developer shell with
		// API_DB_CONNECTION set must not make these specs reach a database.
		saved := cfg
		cfg.Database.Connection = ""
		DeferCleanup(func() {
			cfg = saved
			rootCmd.SetArgs(nil)
		})

		buf := &bytes.Buffer{}
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
	})

	DescribeTable("requires a database connection",
		func(direction string) {
			rootCmd.SetArgs([]string{"migrate", direction})
			Expect(rootCmd.Execute()).To(MatchError(ContainSubstring("--db-connection must not be empty")))
		},
		Entry("up", "up"),
		Entry("down", "down"),
	)

	// The Helm migration Job passes these flags; they are defined on the root
	// command and must be inherited by migrate.
	It("accepts the root command's --log-level and --db-connection flags", func() {
		rootCmd.SetArgs([]string{"migrate", "up", "--log-level=info", "--db-connection="})
		Expect(rootCmd.Execute()).To(MatchError(ContainSubstring("--db-connection must not be empty")))
	})
})
