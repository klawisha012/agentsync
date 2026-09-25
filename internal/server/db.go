package server

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const sessionTTL = 14 * 24 * time.Hour

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *store) migrate() error {
	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return err
	}
	driver, err := pgmigrate.WithInstance(s.db, &pgmigrate.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (s *store) load(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, email_key, name, name_key, password_hash, verified,
		       confirm_token, confirm_expires, reset_token, reset_expires
		FROM accounts`)
	if err != nil {
		return err
	}
	defer rows.Close()
	byID := map[string]*account{}
	for rows.Next() {
		item, err := scanAccount(rows)
		if err != nil {
			return err
		}
		s.byEmail[item.emailKey] = item
		s.byName[item.nameKey] = item
		byID[item.id] = item
	}
	if err := rows.Err(); err != nil {
		return err
	}

	sessions, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, expires_at FROM sessions WHERE expires_at > now()`)
	if err != nil {
		return err
	}
	defer sessions.Close()
	for sessions.Next() {
		var id, accountID string
		var exp time.Time
		if err := sessions.Scan(&id, &accountID, &exp); err != nil {
			return err
		}
		item := byID[accountID]
		if item == nil {
			continue
		}
		s.sessions[id] = item
		s.sessionExpiry[id] = exp
	}
	if err := sessions.Err(); err != nil {
		return err
	}

	spent, err := s.db.QueryContext(ctx, `
		SELECT token, used_at FROM spent_confirms WHERE used_at > now() - interval '24 hours'`)
	if err != nil {
		return err
	}
	defer spent.Close()
	for spent.Next() {
		var token string
		var used time.Time
		if err := spent.Scan(&token, &used); err != nil {
			return err
		}
		s.spentConfirms[token] = used
	}
	if err := spent.Err(); err != nil {
		return err
	}
	if err := s.loadViews(ctx, byID); err != nil {
		return err
	}
	if err := s.loadLikes(ctx, byID); err != nil {
		return err
	}
	if err := s.loadMachines(ctx, byID); err != nil {
		return err
	}
	return s.loadPublications(ctx, byID)
}

func (s *store) loadViews(ctx context.Context, byID map[string]*account) error {
	rows, err := s.db.QueryContext(ctx, `SELECT account_id, viewer_key FROM page_views`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var accountID, key string
		if err := rows.Scan(&accountID, &key); err != nil {
			return err
		}
		page := byID[accountID]
		if page == nil {
			continue
		}
		page.viewers[key] = struct{}{}
		page.views = len(page.viewers)
	}
	return rows.Err()
}

func (s *store) loadLikes(ctx context.Context, byID map[string]*account) error {
	rows, err := s.db.QueryContext(ctx, `SELECT account_id, liker_id FROM page_likes`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var accountID, likerID string
		if err := rows.Scan(&accountID, &likerID); err != nil {
			return err
		}
		page, liker := byID[accountID], byID[likerID]
		if page == nil || liker == nil {
			continue
		}
		page.likedBy[liker.nameKey] = struct{}{}
		page.likes = len(page.likedBy)
	}
	return rows.Err()
}

func (s *store) loadMachines(ctx context.Context, byID map[string]*account) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, account_id, host, listener, token FROM machines`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var bound machine
		var accountID string
		if err := rows.Scan(&bound.id, &accountID, &bound.host, &bound.listener, &bound.token); err != nil {
			return err
		}
		bound.account = byID[accountID]
		if bound.account == nil {
			continue
		}
		s.machines[bound.id] = &bound
		s.byAgentToken[bound.token] = &bound
	}
	return rows.Err()
}

func (s *store) loadPublications(ctx context.Context, byID map[string]*account) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, agent, version, packed, created_at FROM publications`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var accountID string
		item := &publication{}
		if err := rows.Scan(&item.id, &accountID, &item.agent, &item.version, &item.packed, &item.created); err != nil {
			return err
		}
		files, err := unpackFiles(item.packed)
		if err != nil {
			return err
		}
		item.files = files
		page := byID[accountID]
		if page == nil {
			continue
		}
		page.publications = append(page.publications, item)
		if page.publishedAt == nil || item.created.After(*page.publishedAt) {
			created := item.created
			page.publishedAt = &created
		}
	}
	return rows.Err()
}

func scanAccount(rows *sql.Rows) (*account, error) {
	var item account
	var confirmToken, resetToken sql.NullString
	var confirmExpires, resetExpires sql.NullTime
	if err := rows.Scan(
		&item.id, &item.email, &item.emailKey, &item.name, &item.nameKey, &item.password, &item.verified,
		&confirmToken, &confirmExpires, &resetToken, &resetExpires,
	); err != nil {
		return nil, err
	}
	item.confirmToken = confirmToken.String
	if confirmExpires.Valid {
		item.confirmExpires = confirmExpires.Time
	}
	item.resetToken = resetToken.String
	if resetExpires.Valid {
		item.resetExpires = resetExpires.Time
	}
	item.chains = map[string]string{}
	item.viewers = map[string]struct{}{}
	item.likedBy = map[string]struct{}{}
	item.publications = []*publication{}
	return &item, nil
}

func (s *store) insertAccount(ctx context.Context, item *account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (
			id, email, email_key, name, name_key, password_hash, verified,
			confirm_token, confirm_expires, reset_token, reset_expires
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		item.id, item.email, item.emailKey, item.name, item.nameKey, item.password, item.verified,
		nullString(item.confirmToken), nullTime(item.confirmExpires),
		nullString(item.resetToken), nullTime(item.resetExpires),
	)
	return err
}

func (s *store) saveAccount(ctx context.Context, item *account) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE accounts SET
			password_hash = $2, verified = $3,
			confirm_token = $4, confirm_expires = $5,
			reset_token = $6, reset_expires = $7
		WHERE id = $1`,
		item.id, item.password, item.verified,
		nullString(item.confirmToken), nullTime(item.confirmExpires),
		nullString(item.resetToken), nullTime(item.resetExpires),
	)
	return err
}

func (s *store) insertSession(ctx context.Context, id string, item *account) error {
	exp := time.Now().Add(sessionTTL)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, account_id, expires_at) VALUES ($1, $2, $3)`,
		id, item.id, exp)
	if err != nil {
		return err
	}
	s.sessions[id] = item
	s.sessionExpiry[id] = exp
	return nil
}

func (s *store) deleteSession(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id); err != nil {
		return err
	}
	delete(s.sessions, id)
	delete(s.sessionExpiry, id)
	return nil
}

func (s *store) saveSpent(ctx context.Context, token string, used time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO spent_confirms (token, used_at) VALUES ($1, $2)
		ON CONFLICT (token) DO UPDATE SET used_at = EXCLUDED.used_at`, token, used)
	return err
}

func (s *store) accountBySession(id string) *account {
	item := s.sessions[id]
	if item == nil {
		return nil
	}
	exp, ok := s.sessionExpiry[id]
	if ok && !time.Now().Before(exp) {
		delete(s.sessions, id)
		delete(s.sessionExpiry, id)
		return nil
	}
	return item
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func (s *store) putView(ctx context.Context, page *account, key string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO page_views (account_id, viewer_key) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, page.id, key)
	return err
}

func (s *store) dropView(ctx context.Context, page *account, key string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM page_views WHERE account_id = $1 AND viewer_key = $2`, page.id, key)
	return err
}

func (s *store) putLike(ctx context.Context, page, liker *account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO page_likes (account_id, liker_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, page.id, liker.id)
	return err
}

func (s *store) dropLike(ctx context.Context, page, liker *account) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM page_likes WHERE account_id = $1 AND liker_id = $2`, page.id, liker.id)
	return err
}

func (s *store) saveMachine(ctx context.Context, bound *machine) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO machines (id, account_id, host, listener, token)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			account_id = EXCLUDED.account_id,
			host = EXCLUDED.host,
			listener = EXCLUDED.listener,
			token = EXCLUDED.token`,
		bound.id, bound.account.id, bound.host, bound.listener, bound.token)
	return err
}

func (s *store) deleteMachine(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM machines WHERE id = $1`, id)
	return err
}

func (s *store) insertPublication(ctx context.Context, owner *account, item *publication) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO publications (id, account_id, agent, version, packed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		item.id, owner.id, item.agent, item.version, item.packed, item.created)
	return err
}

func uniqueMessage(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return "", false
	}
	switch pgErr.ConstraintName {
	case "accounts_email_key_unique":
		return "Аккаунт с этой почтой уже есть.", true
	case "accounts_name_key_unique":
		return "Это имя уже занято.", true
	default:
		return "Это имя уже занято.", true
	}
}
