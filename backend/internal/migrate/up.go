package migrate

import (
	"fmt"

	"basepro/backend/migrations"
	gomigrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type golangMigrateRunner struct{}

func (golangMigrateRunner) Up(databaseDSN string) error {
	sourceDriver, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := gomigrate.NewWithSourceInstance("iofs", sourceDriver, databaseDSN)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer closeMigrator(m)

	if err := m.Up(); err != nil {
		return err
	}
	return nil
}

func closeMigrator(m *gomigrate.Migrate) {
	sourceErr, dbErr := m.Close()
	_ = sourceErr
	_ = dbErr
}
