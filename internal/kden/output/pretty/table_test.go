package pretty_test

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"

	"charm.land/bubbles/v2/table"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// captureStdout runs fn and returns whatever it wrote to os.Stdout. FormatTable
// prints the rendered table directly, so we intercept the real stdout.
func captureStdout(fn func()) string {
	orig := os.Stdout
	r, w, err := os.Pipe()
	Expect(err).NotTo(HaveOccurred())
	os.Stdout = w

	fn()

	Expect(w.Close()).To(Succeed())
	os.Stdout = orig
	out, err := io.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())
	return string(out)
}

var _ = Describe("FormatTable", func() {
	twoCol := func(_ interface{}) *pretty.TableData {
		return &pretty.TableData{
			Columns: []table.Column{{Title: "Field", Width: 12}, {Title: "Value", Width: 20}},
			Rows:    []table.Row{{"Version", "v1.2.3"}, {"Commit", "abc1234"}},
			Footer:  "a faint hint",
		}
	}

	It("renders the rows and footer to stdout", func() {
		var out string
		var err error
		out = captureStdout(func() { err = pretty.FormatTable(twoCol, nil) })

		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("Version"))
		Expect(out).To(ContainSubstring("v1.2.3"))
		Expect(out).To(ContainSubstring("a faint hint"))
	})

	It("does not emit a scroll-help line", func() {
		out := captureStdout(func() { _ = pretty.FormatTable(twoCol, nil) })
		Expect(out).NotTo(ContainSubstring("↑/k")) // static table: no navigation help
	})

	It("keeps rows tight, not padded to a fixed max width", func() {
		out := captureStdout(func() { _ = pretty.FormatTable(twoCol, nil) })
		for _, line := range strings.Split(out, "\n") {
			Expect(len(line)).To(BeNumerically("<", 100)) // 12+20 cols, not 500
		}
	})

	It("surfaces the model func error", func() {
		bad := func(_ interface{}) *pretty.TableData {
			return &pretty.TableData{Err: errors.New("boom")}
		}
		err := pretty.FormatTable(bad, nil)
		Expect(err).To(MatchError(ContainSubstring("boom")))
	})
})
