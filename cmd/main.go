package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/world-fish/emoji-transfer/internal/platform"
	"github.com/world-fish/emoji-transfer/internal/platform/qq"
	"github.com/world-fish/emoji-transfer/internal/platform/wechat"
	"github.com/world-fish/emoji-transfer/internal/store"
)

func defaultLibDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "emoji-transfer")
}

func allPlatforms() map[string]platform.Platform {
	return map[string]platform.Platform{
		"wechat": wechat.New(),
		"qq":     qq.New(),
	}
}

func main() {
	root := &cobra.Command{
		Use:   "emoji-transfer",
		Short: "跨平台表情包导入导出工具",
	}

	var libDir string
	root.PersistentFlags().StringVar(&libDir, "lib", defaultLibDir(), "local emoji library directory")

	root.AddCommand(
		newExportCmd(&libDir),
		newImportCmd(&libDir),
		newListCmd(&libDir),
		newDetectCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newExportCmd(libDir *string) *cobra.Command {
	var from string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export emojis from a platform into the local library",
		Example: `  emoji-transfer export --from wechat
  emoji-transfer export --from qq`,
		RunE: func(cmd *cobra.Command, args []string) error {
			platforms := allPlatforms()
			p, ok := platforms[from]
			if !ok {
				return fmt.Errorf("unknown platform %q, available: wechat, qq", from)
			}

			ok, err := p.Detect()
			if err != nil {
				return fmt.Errorf("detect %s: %w", from, err)
			}
			if !ok {
				return fmt.Errorf("%s not found on this system", from)
			}

			s, err := store.Open(*libDir)
			if err != nil {
				return fmt.Errorf("open library: %w", err)
			}
			defer s.Close()

			fmt.Printf("Exporting from %s...\n", from)
			added, err := p.Export(s)
			if err != nil {
				return fmt.Errorf("export: %w", err)
			}
			fmt.Printf("Done. %d new emoji(s) added to library.\n", added)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source platform (wechat, qq)")
	cmd.MarkFlagRequired("from")
	return cmd
}

func newImportCmd(libDir *string) *cobra.Command {
	var to string
	var source string
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import emojis from the local library into a platform",
		Example: `  emoji-transfer import --to wechat
  emoji-transfer import --to wechat --source qq`,
		RunE: func(cmd *cobra.Command, args []string) error {
			platforms := allPlatforms()
			p, ok := platforms[to]
			if !ok {
				return fmt.Errorf("unknown platform %q, available: wechat, qq", to)
			}

			s, err := store.Open(*libDir)
			if err != nil {
				return fmt.Errorf("open library: %w", err)
			}
			defer s.Close()

			emojis, err := s.List(source)
			if err != nil {
				return fmt.Errorf("list library: %w", err)
			}
			if len(emojis) == 0 {
				fmt.Println("No emojis found in library.")
				return nil
			}

			fmt.Printf("Importing %d emoji(s) into %s...\n", len(emojis), to)
			if err := p.Import(emojis); err != nil {
				return fmt.Errorf("import: %w", err)
			}
			fmt.Println("Done.")
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "target platform (wechat, qq)")
	cmd.Flags().StringVar(&source, "source", "", "only import emojis from this source platform (optional)")
	cmd.MarkFlagRequired("to")
	return cmd
}

func newListCmd(libDir *string) *cobra.Command {
	var source string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List emojis in the local library",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.Open(*libDir)
			if err != nil {
				return fmt.Errorf("open library: %w", err)
			}
			defer s.Close()

			emojis, err := s.List(source)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			if len(emojis) == 0 {
				fmt.Println("Library is empty.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "HASH\tFORMAT\tSOURCE\tADDED")
			for _, e := range emojis {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					e.Hash[:8],
					e.Format,
					e.Source,
					e.AddedAt.Format("2006-01-02 15:04"),
				)
			}
			w.Flush()
			fmt.Printf("\nTotal: %d emoji(s)\n", len(emojis))
			return nil
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "filter by source platform")
	return cmd
}

func newDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Check which platforms are available on this system",
		RunE: func(cmd *cobra.Command, args []string) error {
			platforms := allPlatforms()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PLATFORM\tSTATUS")
			for name, p := range platforms {
				ok, err := p.Detect()
				status := "found"
				if err != nil {
					status = "error: " + err.Error()
				} else if !ok {
					status = "not found"
				}
				fmt.Fprintf(w, "%s\t%s\n", name, status)
			}
			w.Flush()
			return nil
		},
	}
}
