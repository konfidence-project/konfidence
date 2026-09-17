package cmd

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("migrate command", func() {
	var root *cobra.Command

	BeforeEach(func() {
		root = &cobra.Command{Use: "api"}
		root.AddCommand(newMigrateCmd())
	})

	It("registers up and down sub-commands", func() {
		migrate, _, err := root.Find([]string{"migrate"})
		Expect(err).NotTo(HaveOccurred())
		Expect(migrate).NotTo(BeNil())

		names := make([]string, 0, len(migrate.Commands()))
		for _, c := range migrate.Commands() {
			names = append(names, c.Name())
		}
		Expect(names).To(ConsistOf("up", "down"))
	})

	It("requires --db-connection for migrate up", func() {
		buf := &bytes.Buffer{}
		root.SetOut(buf)
		root.SetErr(buf)

		root.SetArgs([]string{"migrate", "up"})
		err := root.Execute()
		Expect(err).To(MatchError(ContainSubstring("db-connection")))
	})

	It("requires --db-connection for migrate down", func() {
		buf := &bytes.Buffer{}
		root.SetOut(buf)
		root.SetErr(buf)

		root.SetArgs([]string{"migrate", "down"})
		err := root.Execute()
		Expect(err).To(MatchError(ContainSubstring("db-connection")))
	})
})
