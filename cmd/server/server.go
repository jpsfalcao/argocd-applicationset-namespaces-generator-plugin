package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
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

func (c *ServerConfig) validate() error {
	if strings.TrimSpace(c.ListenToken) == "" && !c.Local {
		return errors.New("listen-token is required (unless --local)")
	}
	return nil
}

func init() {
	Cmd.PersistentFlags().String("listen-address", ":8080", "Local address to listen on")
	Cmd.PersistentFlags().String("listen-token", "", "Bearer token to authenticate requests (required unless --local)")

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
		if err := config.validate(); err != nil {
			return err
		}
		if config.ListenToken == "" {
			slog.Warn("No listen token set: accepting unauthenticated requests (--local)", "listenAddress", config.ListenAddress)
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
