package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

var serverPort int

const (
	requestIDKey = "requestID"
	loggerKey    = "logger"
)

func loggingMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		requestID := uuid.New().String()
		ctx.SetUserValue(requestIDKey, requestID)
		next(ctx)
		duration := time.Since(start)
		log.Debug().
			Str("method", string(ctx.Method())).
			Str("path", string(ctx.Path())).
			Str("remote_ip", ctx.RemoteIP().String()).
			Int("status", ctx.Response.StatusCode()).
			Dur("latency", duration).
			Str("request_id", requestID).
			Msg("HTTP request")
	}
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		log.Trace().Msg("This is a top-level trace log from serverCmd.Run")
		handler := func(ctx *fasthttp.RequestCtx) {
			reqLogger, ok := ctx.UserValue(loggerKey).(zerolog.Logger)
			if !ok {
				reqLogger = log.Logger
			}
			reqLogger.Trace().Msg("Handler entered")
			fmt.Fprintf(ctx, "Hello from FastHTTP! Your request ID: %s", ctx.UserValue(requestIDKey))
			reqLogger.Trace().Msg("Handler exiting")
		}
		wrappedHandler := loggingMiddleware(handler)
		addr := fmt.Sprintf(":%d", serverPort)
		log.Info().Msgf("Starting FastHTTP server on %s", addr)
		if err := fasthttp.ListenAndServe(addr, wrappedHandler); err != nil {
			log.Error().Err(err).Msg("Error starting FastHTTP server")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the server on")
}
