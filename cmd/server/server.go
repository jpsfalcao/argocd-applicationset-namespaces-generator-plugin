package server

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

type ServerConfig struct {
	ListenAddress string `mapstructure:"listen-address"`
	ListenToken   string `mapstructure:"listen-token"`

	Local bool `mapstructure:"local"`

	clientOnce sync.Once
	client     kubernetes.Interface
	clientErr  error
}

func init() {
	Cmd.PersistentFlags().String("listen-address", ":8080", "Local address to listen on")
	Cmd.PersistentFlags().String("listen-token", "", "Bearer token to authenticate requests (if needed)")

	Cmd.PersistentFlags().Bool("local", false, "Enable to use local kubectl context (for debugging)")
}

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Start plugin server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		config := ServerConfig{}
		if err := viper.Unmarshal(&config); err != nil {
			return err
		}

		// Fail at startup rather than on the first request.
		if _, err := config.GetClient(nil); err != nil {
			return err
		}

		http.HandleFunc("/api/v1/getparams.execute", config.secretsHandler(ctx))

		server := &http.Server{
			Addr:              config.ListenAddress,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		slog.Info("Server starting...", "listenAddress", config.ListenAddress)
		if err := server.ListenAndServe(); err != nil {
			slog.Error("Server Failure", "err", err)
			return err
		}
		return nil
	},
}
