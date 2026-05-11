package state

import (
	"os"

	"github.com/git-logs/client/webserver/config"
	"github.com/git-logs/client/webserver/mapofmu"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/genconfig"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var (
	TableEventModifiers = "event_modifiers"
	TableRepos          = "repos"
	TableGuilds         = "guilds"
	TableWebhooks       = "webhooks"
	TableWebhookLogs    = "webhook_logs"

	TableList = []*string{
		&TableEventModifiers,
		&TableRepos,
		&TableGuilds,
		&TableWebhooks,
		&TableWebhookLogs,
	}
)

func Setup() {
	MapMutex = mapofmu.New[string]()

	// Initialize logger
	w := zapcore.AddSync(os.Stdout)

	var level = zap.InfoLevel
	if os.Getenv("DEBUG") == "true" {
		level = zap.DebugLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		level,
	)

	Logger = zap.New(core)

	Logger.Info("Generating config")
	genconfig.SampleFileName = configFile + ".sample"

	genconfig.GenConfig(config.Config{})

	Logger.Info("Loading config", zap.String("file", configFile))
	cfg, err := os.ReadFile(configFile)

	if err != nil {
		Logger.Fatal("Could not read config file", zap.Error(err), zap.String("file", configFile))
	}

	err = yaml.Unmarshal(cfg, &Config)

	if err != nil {
		panic(err)
	}

	Logger.Info("Validating config")
	err = Validator.Struct(Config)

	if err != nil {
		Logger.Fatal("Config validation failed", zap.Error(err))
	}

	Logger.Info("Connecting to service [postgres]")
	Pool, err = pgxpool.New(Context, Config.PostgresURL)

	if err != nil {
		Logger.Fatal("Could not connect to postgres", zap.Error(err))
	}

	Logger.Info("Connecting to service [discord]")
	Discord, err = discordgo.New("Bot " + Config.Token)

	if err != nil {
		Logger.Fatal("Could not open discord connection", zap.Error(err))
	}

	Logger.Info("Connected to all services successfully")

	ApplyMigrations()
}

// Must be called when embedding, PrepareForEmbedding creates the table names from config and may do other setup
// in the future
//
// Note that config.GetTable must be set before calling this function
func PrepareForEmbedding() {
	for _, table := range TableList {
		*table = Config.GetTable(*table)
	}

	IsEmbedded = true
}

func ApplyMigrations() {
	/*
		webhooks.created_by TEXT NOT NULL [set unfilled to '']
		webhooks.last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		webhooks.last_updated_by TEXT NOT NULL [set unfilled to '']

		repos.created_by TEXT NOT NULL [set unfilled to '']
		repos.last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		repos.last_updated_by TEXT NOT NULL [set unfilled to '']

		event_modifiers.created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		event_modifiers.created_by TEXT NOT NULL [set unfilled to '']
		event_modifiers.last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		event_modifiers.last_updated_by TEXT NOT NULL [set unfilled to '']

		webhook_logs.webhook_id text not null references webhooks (id) ON UPDATE CASCADE ON DELETE CASCADE [drop all if webhook_id unset]
	*/

	tx, err := Pool.Begin(Context)

	if err != nil {
		Logger.Fatal("Could not start migration transaction", zap.Error(err))
	}

	defer tx.Rollback(Context)

	var countOfWebhookId int64
	err = tx.QueryRow(Context, "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = 'webhook_id'", TableWebhookLogs).Scan(&countOfWebhookId)

	if err != nil {
		Logger.Fatal("Could not check for webhook_id column", zap.Error(err))
	}

	var countOfGuildId int64
	err = tx.QueryRow(Context, "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = 'guild_id'", TableWebhookLogs).Scan(&countOfGuildId)

	if err != nil {
		Logger.Fatal("Could not check for guild_id column", zap.Error(err))
	}

	if countOfGuildId == 0 || countOfWebhookId == 0 {
		_, err = tx.Exec(Context, `
			DELETE FROM `+TableWebhookLogs+`
		`)

		if err != nil {
			Logger.Fatal("Could not delete webhook_logs", zap.Error(err))
		}
	}

	_, err = tx.Exec(Context, `
		ALTER TABLE `+TableWebhooks+` ADD COLUMN IF NOT EXISTS created_by TEXT NOT NULL DEFAULT '0';
		ALTER TABLE `+TableWebhooks+` ALTER COLUMN created_by DROP DEFAULT;
		ALTER TABLE `+TableWebhooks+` ADD COLUMN IF NOT EXISTS last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
		ALTER TABLE `+TableWebhooks+` ADD COLUMN IF NOT EXISTS last_updated_by TEXT NOT NULL DEFAULT '0';
		ALTER TABLE `+TableWebhooks+` ALTER COLUMN last_updated_by DROP DEFAULT;

		ALTER TABLE `+TableRepos+` ADD COLUMN IF NOT EXISTS created_by TEXT NOT NULL DEFAULT '0';
		ALTER TABLE `+TableRepos+` ALTER COLUMN created_by DROP DEFAULT;
		ALTER TABLE `+TableRepos+` ADD COLUMN IF NOT EXISTS last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
		ALTER TABLE `+TableRepos+` ADD COLUMN IF NOT EXISTS last_updated_by TEXT NOT NULL DEFAULT '0';
		ALTER TABLE `+TableRepos+` ALTER COLUMN last_updated_by DROP DEFAULT;

		ALTER TABLE `+TableEventModifiers+` ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
		ALTER TABLE `+TableEventModifiers+` ADD COLUMN IF NOT EXISTS created_by TEXT NOT NULL DEFAULT '0';
		ALTER TABLE `+TableEventModifiers+` ALTER COLUMN created_by DROP DEFAULT;
		ALTER TABLE `+TableEventModifiers+` ADD COLUMN IF NOT EXISTS last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
		ALTER TABLE `+TableEventModifiers+` ADD COLUMN IF NOT EXISTS last_updated_by TEXT NOT NULL DEFAULT '0';
		ALTER TABLE `+TableEventModifiers+` ALTER COLUMN last_updated_by DROP DEFAULT;

		ALTER TABLE `+TableWebhookLogs+` ADD COLUMN IF NOT EXISTS webhook_id TEXT NOT NULL REFERENCES `+TableWebhooks+` (id) ON UPDATE CASCADE ON DELETE CASCADE;
		ALTER TABLE `+TableWebhookLogs+` ADD COLUMN IF NOT EXISTS guild_id TEXT NOT NULL REFERENCES `+TableGuilds+` (id) ON UPDATE CASCADE ON DELETE CASCADE;

		ALTER TABLE `+TableWebhooks+` ADD COLUMN IF NOT EXISTS broken BOOLEAN NOT NULL DEFAULT false;
		ALTER TABLE `+TableWebhooks+` ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'github';
		ALTER TABLE `+TableWebhooks+` ALTER COLUMN provider DROP NOT NULL;
	`)

	if err != nil {
		Logger.Fatal("Could not apply migrations", zap.Error(err))
	}

	err = tx.Commit(Context)

	if err != nil {
		Logger.Fatal("Could not commit migration transaction", zap.Error(err))
	}
}
