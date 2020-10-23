package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
	"time"
)

var Version string
var Commit string

func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Show fx tools version",
		Example: "fx version",
		Run: func(*cobra.Command, []string) {
			fmt.Printf("Version: %s\n", Version)
			fmt.Printf("Git Commit: %s\n", Commit)
			fmt.Printf("Go Version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Build Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		},
	}
	return versionCmd
}
