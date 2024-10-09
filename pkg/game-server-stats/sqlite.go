package gameserverstats

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/tursodatabase/go-libsql"
	"vim-arcade.theprimeagen.com/pkg/assert"
)

func checkTableExists(db *sqlx.DB) bool {
    query := `SELECT name
FROM sqlite_master
WHERE type='table' AND name='GameServerConfigs';`

    var tableName string
    err := db.Get(&tableName, query)
    assert.NoError(err, "error while checking for the table existing")
    return tableName == "GameServerConfigs"
}

func deleteTable(db *sqlx.DB) error {
    query := `DROP TABLE IF EXISTS GameServerConfigs;`
    _, err := db.Exec(query)
    return err
}

// createTable creates the GameServerConfigs table
func (s *Sqlite) CreateGameServerConfigs() error {
    // TODO validate this: apparently INTEGER for last_updated makes life
    // easier for calculations??
    query := `
    CREATE TABLE GameServerConfigs (
        id TEXT PRIMARY KEY,
        state TEXT,
        connections INTEGER,
        connections_added INTEGER,
        connections_removed INTEGER,
        last_updated INTERGER,
        load REAL,
        host TEXT,
        port INTEGER
    );`

    _, err := s.db.Exec(query)
    if err != nil {
        return err
    }

    var createLoadIndex = `CREATE INDEX idx_load ON GameServerConfigs (Load);`
    _, err = s.db.Exec(createLoadIndex)

    return err
}

type SqliteFile struct {
    Stats []GameServerConfig `json:"stats"`
}

type Sqlite struct {
    db *sqlx.DB
    logger *slog.Logger
}

func getLogger() *slog.Logger {
    return slog.Default().With("area", "Sqlite")
}

func ClearSQLiteFiles(path string) {
    os.Remove(path)
    os.Remove(fmt.Sprintf("%s-shm", path))
    os.Remove(fmt.Sprintf("%s-wal", path))
}

func NewSqlite(path string) *Sqlite {
    logger := getLogger()
    db, err := sqlx.Open("libsql", path)
    assert.NoError(err, "failed to open db")
    logger.Warn("New Sqlite", "path", path)
    return &Sqlite{
        db: db,
        logger: logger,
    }
}

func (s *Sqlite) Close() error {
    return s.db.Close()
}

func (s *Sqlite) setPragma(name string, value string) {
    row := s.db.QueryRowx(fmt.Sprintf("PRAGMA %s=%s;", name, value))
    var v string
    err := row.Scan(&v)
    assert.NoError(err, "could not scan pragma row result", "name", name, "value", value)
    s.logger.Warn(name, "value", v)
}

func (s *Sqlite) SetSqliteModes() {
    s.setPragma("busy_timeout", "3000")
    s.setPragma("journal_mode", "WAL")
}

func (s *Sqlite) GetServerCount() int {
    selectQuery := `SELECT COUNT(*)
FROM GameServerConfigs;`

    var count int
    err := s.db.Get(&count, selectQuery)
    assert.NoError(err, "unable to get server count")

    return count
}

func (s *Sqlite) GetTotalConnectionCount() GameServecConfigConnectionStats {
    sumQuery := `SELECT CAST(TOTAL(connections) AS INT) AS connections,
                     CAST(TOTAL(connections_added) AS INT) AS connections_added,
                     CAST(TOTAL(connections_removed) AS INT) AS connections_removed
FROM GameServerConfigs;`

    var counts GameServecConfigConnectionStats
    err := s.db.Get(&counts, sumQuery)
    assert.NoError(err, "unable to get total connection count")

    return counts
}

func (s *Sqlite) Update(stat GameServerConfig) error {
    s.logger.Info("Updating", "stat", stat)
    query := `INSERT OR REPLACE INTO GameServerConfigs (id, state, connections, connections_added, connections_removed, load, host, port, last_updated)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, strftime('%s', 'now'));`

    // TODO probably don't need to update every
    res, err := s.db.Exec(query, stat.Id, stat.State, stat.Connections, stat.ConnectionsAdded, stat.ConnectionsRemoved, stat.Load, stat.Host, stat.Port)
    n, err := res.RowsAffected()
    s.logger.Info("update complete", "rows affected", n, "error", err)

    return err
}

func EnsureSqliteURI(path string) string {
    if strings.HasPrefix(path, "file:") ||
        strings.HasPrefix(path, "https://") {
        return path
    }

    return "file:" + path
}


func (j *Sqlite) Run(ctx context.Context) {
    <-ctx.Done()
    j.db.Close()
    j.logger.Warn("Sqlite finished running")
}

func (s *Sqlite) GetConfigByHostAndPort(host string, port uint16) (*GameServerConfig, error) {
    query := `SELECT * FROM GameServerConfigs WHERE host = ? AND port = ?;`
    s.logger.Error("GetConfigByHostAndPort", "query", query)
    config := []GameServerConfig{}
    err := s.db.Select(&config, query, host, port)

    s.logger.Error("GetConfigByHostAndPort", "query", query, "config", config, "error", err)

    if len(config) == 1 {
        return &config[0], err
    }
    return nil, err
}

func (s *Sqlite) DeleteGameServerConfig(id string) {
    assert.Assert(os.Getenv("ENV") == "TESTING", "can only delete server configs while testing")
    query := `DELETE FROM table_name WHERE column_name = ?;`
    _, err := s.db.Exec(query)
    assert.NoError(err, "there should be no error when deleting a row.")
}

func (s *Sqlite) GetAllGameServerConfigs() ([]GameServerConfig, error) {
    var configs []GameServerConfig
    query := `SELECT id, state, connections, load, host, port FROM GameServerConfigs;`

    err := s.db.Select(&configs, query)
    if err != nil {
        return nil, err
    }

    return configs, nil
}

func (s *Sqlite) GetById(id string) *GameServerConfig {
    g := []GameServerConfig{}
    s.db.Select(&g, `SELECT *
FROM GameServerConfigs
WHERE id=?;`, id)
    if len(g) == 1 {
        s.logger.Info("GetById", "id", id, "stat", g[0].String())
        return &g[0]
    }
    return nil
}

func (s *Sqlite) GetServersByUtilization(maxLoad float64) []GameServerConfig {
    var g []GameServerConfig
    s.db.Select(&g, `SELECT *
FROM GameServerConfigs
WHERE load < ? AND state == ?
ORDER BY load DESC;`, maxLoad, GSStateReady)
    s.logger.Info("GetServersByUtilization", "maxLoad", maxLoad, "count", len(g))
    return g
}
