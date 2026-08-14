package repository

import (
	"context"

	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/models"
	"go.etcd.io/bbolt"
)

const settingsKey = "settings"

type boltSettingsRepository struct {
	bolt *bbolt.DB
	// defaults is returned (and persisted) the first time Get is called
	// against an empty database, so the app always has usable settings.
	defaults models.Settings
}

// NewSettingsRepository builds a bbolt-backed SettingsRepository seeded with defaults.
func NewSettingsRepository(db *database.DB, defaults models.Settings) SettingsRepository {
	return &boltSettingsRepository{bolt: db.Bolt(), defaults: defaults}
}

func (r *boltSettingsRepository) Get(ctx context.Context) (*models.Settings, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var settings models.Settings
	err := database.Get(r.bolt, database.BucketSettings, settingsKey, &settings)
	if err == database.ErrNotFound {
		settings = r.defaults
		if saveErr := database.Put(r.bolt, database.BucketSettings, settingsKey, &settings); saveErr != nil {
			return nil, saveErr
		}
		return &settings, nil
	}
	if err != nil {
		return nil, err
	}

	// Sync environment/config Telegram credentials and report settings if updated on startup
	if r.defaults.TelegramBotToken != "" && settings.TelegramBotToken != r.defaults.TelegramBotToken {
		settings.TelegramBotToken = r.defaults.TelegramBotToken
		if r.defaults.TelegramChatID != "" {
			settings.TelegramChatID = r.defaults.TelegramChatID
		}
		_ = database.Put(r.bolt, database.BucketSettings, settingsKey, &settings)
	}
	if settings.SpeedReportEnabled != r.defaults.SpeedReportEnabled {
		settings.SpeedReportEnabled = r.defaults.SpeedReportEnabled
		_ = database.Put(r.bolt, database.BucketSettings, settingsKey, &settings)
	}

	return &settings, nil
}

func (r *boltSettingsRepository) Save(ctx context.Context, settings *models.Settings) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return database.Put(r.bolt, database.BucketSettings, settingsKey, settings)
}
