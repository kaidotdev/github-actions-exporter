package cmd

import "github.com/spf13/cobra"

func GetRootCmd(args []string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "github-actions-exporter",
		Short:        "KubeOutdatedImageExporter is Prometheus Exporter that collects all outdated image by querying Docker Registry API against all of container images in cluster.",
		SilenceUsage: true,
	}

	rootCmd.SetArgs(args)
	rootCmd.AddCommand(serverCmd())

	return rootCmd
}
