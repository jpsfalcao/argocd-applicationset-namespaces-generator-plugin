package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	ListenAddress string `mapstructure:"listen-address"`
	ListenToken   string `mapstructure:"listen-token"`

	Local bool `mapstructure:"local"`
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

		http.HandleFunc("/api/v1/getparams.execute", config.secretsHandler(ctx))

		slog.Info("Server starting...", "listenAddress", config.ListenAddress)
		if err := http.ListenAndServe(config.ListenAddress, nil); err != nil {
			slog.Error("Server Failure", "err", err)
			return err
		}
		return nil
	},
}
