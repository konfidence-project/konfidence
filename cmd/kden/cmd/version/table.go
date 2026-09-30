package version

import (
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	"github.com/konfidence-project/konfidence/pkg/build"
)

// updateHint is the footer under the version table. It mirrors installCommand;
// keep the two in sync if the URL changes.
const updateHint = "To update, re-run: " + installCommand

func table(info build.Info) *pretty.TableData {
	builder := pretty.NewTable().Columns("Field", "Value").Footer(updateHint)
	builder.Row().String("Field", "Version").String("Value", info.Version)
	builder.Row().String("Field", "Commit").String("Value", info.Commit)
	builder.Row().String("Field", "Go").String("Value", info.GoVersion)
	builder.Row().String("Field", "Platform").String("Value", info.Platform)
	builder.Row().String("Field", "built").String("Value", info.Date)
	return builder.TableData()
}
