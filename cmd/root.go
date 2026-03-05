package cmd

import (
	"fmt"
	"os"

	"github.com/LotadTech/lazy-fluxcd/internal/k8"
	"github.com/LotadTech/lazy-fluxcd/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var kubeconfig string

var rootCmd = &cobra.Command{
	Use:   "lazy-fluxcd",
	Short: "A TUI for FluxCD",
	Long:  `lazy-fluxcd is an interactive terminal UI for managing FluxCD resources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := k8.DynamicClient(kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to connect to cluster: %w", err)
		}
		clusterName, err := k8.CurrentContext(kubeconfig)
		if err != nil {
			clusterName = "unknown"
		}
		p := tea.NewProgram(tui.New(client, clusterName), tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file (defaults to ~/.kube/config)")
}
