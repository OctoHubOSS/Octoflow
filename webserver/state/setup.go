package state

import (
	"os"

	"github.com/OctoHubOSS/Octoflow/webserver/config"
	"github.com/OctoHubOSS/Octoflow/webserver/mapofmu"

	"github.com/PlexiOSS/Keel/genconfig"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var (
	TableEventModifiers  = "event_modifiers"
	TableRepos           = "repos"
	TableGuilds          = "guilds"
	TableWebhooks        = "webhooks"
	TableWebhookLogs     = "webhook_logs"
	TableBotHeartbeat    = "bot_heartbeat"
	TableStatusSnapshots = "status_snapshots"

	TableList = []*string{
		&TableEventModifiers,
		&TableRepos,
		&TableGuilds,
		&TableWebhooks,
		&TableWebhookLogs,
		&TableBotHeartbeat,
		&TableStatusSnapshots,
	}
)

func Setup() {
	MapMutex = mapofmu.New[string]()

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

	if v := os.Getenv("APPLY_MIGRATIONS"); v == "true" {
		ApplyMigrations()
	}
}

func PrepareForEmbedding() {
	for _, table := range TableList {
		*table = Config.GetTable(*table)
	}

	IsEmbedded = true
}

func ApplyMigrations() {
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

		CREATE TABLE IF NOT EXISTS `+TableBotHeartbeat+` (
			id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			guild_count INTEGER NOT NULL,
			member_count BIGINT NOT NULL,
			shard_count INTEGER NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS `+TableStatusSnapshots+` (
			id BIGSERIAL PRIMARY KEY,
			checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			database_up BOOLEAN NOT NULL,
			discord_up BOOLEAN NOT NULL,
			db_latency_ms INTEGER NOT NULL
		);
	`)

	if err != nil {
		Logger.Fatal("Could not apply migrations", zap.Error(err))
	}

	err = tx.Commit(Context)

	if err != nil {
		Logger.Fatal("Could not commit migration transaction", zap.Error(err))
	}
}
