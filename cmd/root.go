package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/syntropynet/data-layer-sdk/pkg/options"
	"gorm.io/gorm"

	"github.com/syntropynet/osmosis-publisher/internal/repository"
	"github.com/syntropynet/osmosis-publisher/internal/repository/pg"
	"github.com/syntropynet/osmosis-publisher/internal/repository/sqlite"
)

var (
	flagVerbose         *bool
	flagTelemetryPeriod *time.Duration
	flagNatsPubUrls     *string
	flagUserPubCreds    *string
	flagNkeyPub         *string
	flagNatsAccNkey     *string
	flagJWTPub          *string
	flagNatsSubUrls     *string
	flagUserSubCreds    *string
	flagNkeySub         *string
	flagJWTSub          *string
	flagTLSClientCert   *string
	flagTLSKey          *string
	flagCACert          *string
	flagPrefixName      *string
	flagPemFile         *string

	flagDbHost     *string
	flagDbPort     *uint
	flagDbUser     *string
	flagDbPassword *string
	flagDbName     *string

	natsPubConnection *nats.Conn
	natsSubConnection *nats.Conn
	database          *repository.Repository
)

func setErrorHandlers(conn *nats.Conn) {
	if conn == nil {
		return
	}

	conn.SetErrorHandler(func(c *nats.Conn, s *nats.Subscription, err error) {
		slog.Error("NATS error", err)
	})
	conn.SetDisconnectHandler(func(c *nats.Conn) {
		slog.Error("NATS disconnected", c.LastError())
	})
}

var rootCmd = &cobra.Command{
	Use:   "osmosis-publisher",
	Short: "",
	Long:  ``,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Sacrifice some security for the sake of user experience by allowing to
		// supply NATS account NKey instead of passing created user NKey and user JWS.
		if *flagNatsAccNkey != "" {
			nkey, jwt, err := CreateUser(*flagNatsAccNkey)
			flagNkeyPub = nkey
			flagJWTPub = jwt

			if err != nil {
				panic(fmt.Errorf("failed to generate user JWT: %w", err))
			}
		}

		conn, err := options.MakeNats("Osmosis Publisher", *flagNatsPubUrls, *flagUserPubCreds, *flagNkeyPub, *flagJWTPub, *flagCACert, *flagTLSClientCert, *flagTLSKey)
		if err != nil {
			panic(fmt.Errorf("failed to connect to publisher NATS %s: %w", *flagNatsPubUrls, err))
		}
		natsPubConnection = conn
		setErrorHandlers(conn)

		conn, err = options.MakeNats("Osmosis Subscriber", *flagNatsSubUrls, *flagUserSubCreds, *flagNkeySub, *flagJWTSub, *flagCACert, *flagTLSClientCert, *flagTLSKey)
		if err != nil {
			panic(fmt.Errorf("failed to connect to subscriber NATS %s: %w", *flagNatsSubUrls, err))
		}
		natsSubConnection = conn
		setErrorHandlers(conn)

		var db *gorm.DB
		if *flagDbName == "sqlite" {
			db, err = sqlite.New(*flagDbHost)
			if err != nil {
				panic(err)
			}
		} else {
			db, err = pg.New(*flagDbHost, *flagDbPort, *flagDbUser, *flagDbPassword, *flagDbName)
			if err != nil {
				panic(err)
			}
		}
		repo, err := repository.New(db, slog.Default())
		if err != nil {
			panic(err)
		}
		database = repo
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if natsPubConnection == nil {
			return
		}
		natsPubConnection.Close()
		database.Close()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	const (
		PUBLISHER_IDENTITY = "IDENTITY"
		PUBLISHER_PREFIX   = "PREFIX"
		DB_HOST            = "DB_HOST"
		DB_PORT            = "DB_PORT"
		DB_USER            = "DB_USER"
		DB_PASSWORD        = "DB_PASSW"
		DB_NAME            = "DB_NAME"
	)
	setDefault(PUBLISHER_PREFIX, "syntropy")
	setDefault(DB_HOST, "postgres")
	setDefault(DB_PORT, "5432")
	setDefault(DB_USER, "osmopub_user")
	setDefault(DB_NAME, "osmopub")

	flagNatsPubUrls = rootCmd.PersistentFlags().StringP("nats-url", "n", os.Getenv("NATS_URL"), "NATS server URLs (separated by comma)")
	flagNatsAccNkey = rootCmd.PersistentFlags().StringP("nats-acc-nkey", "", os.Getenv("NATS_ACC_NKEY"), "NATS account NKey (seed)")
	flagUserPubCreds = rootCmd.PersistentFlags().StringP("nats-creds", "c", os.Getenv("NATS_CREDS"), "NATS User Credentials File (combined JWT and NKey file) ")
	flagJWTPub = rootCmd.PersistentFlags().StringP("nats-jwt", "w", os.Getenv("NATS_JWT"), "NATS JWT")
	flagNkeyPub = rootCmd.PersistentFlags().StringP("nats-nkey", "k", os.Getenv("NATS_NKEY"), "NATS NKey")

	flagNatsSubUrls = rootCmd.PersistentFlags().StringP("nats-sub-url", "", os.Getenv("NATS_SUB_URL"), "NATS server URLs (separated by comma) for Subscribing only")
	flagUserSubCreds = rootCmd.PersistentFlags().StringP("nats-sub-creds", "", os.Getenv("NATS_SUB_CREDS"), "NATS User Credentials File (combined JWT and NKey file) for Subscribing only")
	flagJWTSub = rootCmd.PersistentFlags().StringP("nats-sub-jwt", "", os.Getenv("NATS_SUB_JWT"), "NATS JWT for Subscribing only")
	flagNkeySub = rootCmd.PersistentFlags().StringP("nats-sub-nkey", "", os.Getenv("NATS_SUB_NKEY"), "NATS NKey for Subscribing only")

	flagTLSKey = rootCmd.PersistentFlags().StringP("client-key", "", os.Getenv("CLIENT_KEY"), "NATS Private key file for client certificate")
	flagTLSClientCert = rootCmd.PersistentFlags().StringP("client-cert", "", os.Getenv("CLIENT_CERT"), "NATS TLS client certificate file")
	flagCACert = rootCmd.PersistentFlags().StringP("ca-cert", "", os.Getenv("CA_CERT"), "NATS CA certificate file")

	flagDbHost = rootCmd.PersistentFlags().StringP("db-host", "", os.Getenv(DB_HOST), "Database Host (filepath in case of `sqlite` `db-name`)")

	envPort := os.Getenv(DB_PORT)
	port, err := strconv.ParseUint(envPort, 10, 64)
	if err != nil {
		port = 5432
		slog.Warn("Bad database port format, switching to default", "error", err, "port", port)
	}

	flagDbPort = rootCmd.PersistentFlags().UintP("db-port", "", uint(port), "Database Port")
	flagDbUser = rootCmd.PersistentFlags().StringP("db-user", "", os.Getenv(DB_USER), "Database User")
	flagDbName = rootCmd.PersistentFlags().StringP("db-name", "", os.Getenv(DB_NAME), "Database Name (specify `sqlite` for SQLite database)")
	flagDbPassword = rootCmd.PersistentFlags().StringP("db-passw", "", os.Getenv(DB_PASSWORD), "Database Password")

	flagPemFile = rootCmd.PersistentFlags().StringP("identity", "i", os.Getenv(PUBLISHER_IDENTITY), "Identity as a PEM file containing a private key File")
	flagPrefixName = rootCmd.PersistentFlags().StringP("prefix", "", os.Getenv(PUBLISHER_PREFIX), "NATS topic prefix name as in {prefix}.solana")

	_, verbosePresent := os.LookupEnv("VERBOSE")

	flagVerbose = rootCmd.PersistentFlags().BoolP("verbose", "v", verbosePresent, "Verbose output")

	envTelemetryPeriod := os.Getenv("TELEMETRY_PERIOD")
	var telemetryPeriod time.Duration
	if envTelemetryPeriod != "" {
		var err error
		telemetryPeriod, err = time.ParseDuration(envTelemetryPeriod)
		if err != nil {
			telemetryPeriod = time.Second * 3
			slog.Warn("Invalid format for TELEMETRY_PERIOD environment variable.", "error", err, "default", telemetryPeriod)
		}
	} else {
		telemetryPeriod = time.Second * 3
	}

	flagTelemetryPeriod = rootCmd.PersistentFlags().DurationP("telemetry-period", "T", telemetryPeriod, "Telemetry report period")
}
