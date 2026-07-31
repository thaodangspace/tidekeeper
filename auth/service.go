package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

var (
	// ErrInvalidInput means an email address or password did not meet the public policy.
	ErrInvalidInput = errors.New("invalid authentication input")
	// ErrEmailTaken means an account has already claimed an email address.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidCredentials deliberately covers unknown emails and bad passwords.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

const maxEmailLength = 254

// Session is a newly issued raw opaque session credential.
type Session struct {
	Token     string
	ExpiresAt time.Time
}

// Service persists first-party account credentials and opaque sessions.
type Service struct {
	pool       *pgxpool.Pool
	sessionTTL time.Duration
}

// NewService creates an account-authentication service using the supplied pool.
func NewService(pool *pgxpool.Pool, sessionTTL time.Duration) *Service {
	return &Service{pool: pool, sessionTTL: sessionTTL}
}

// Register creates an account and its player record, then returns an active session.
func (s *Service) Register(ctx context.Context, email, password string) (Session, error) {
	email, err := normalizeCredentials(email, password)
	if err != nil {
		return Session{}, err
	}
	passwordHash, err := HashPassword(password)
	if err != nil {
		return Session{}, err
	}

	session, err := newSession(s.sessionTTL)
	if err != nil {
		return Session{}, err
	}
	accountID, err := randomUUID()
	if err != nil {
		return Session{}, err
	}
	playerID, err := randomUUID()
	if err != nil {
		return Session{}, err
	}
	publicID, err := newPlayerPublicID()
	if err != nil {
		return Session{}, err
	}
	sessionID, err := randomUUID()
	if err != nil {
		return Session{}, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin account registration: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := query.New(tx)
	if err := queries.CreateAccount(ctx, query.CreateAccountParams{
		ID: accountID, Email: email, PasswordHash: passwordHash,
	}); err != nil {
		if isUniqueViolation(err) {
			return Session{}, ErrEmailTaken
		}
		return Session{}, fmt.Errorf("create account: %w", err)
	}
	if err := queries.CreatePlayer(ctx, query.CreatePlayerParams{
		ID: playerID, AccountID: accountID, PublicID: publicID,
	}); err != nil {
		return Session{}, fmt.Errorf("create player: %w", err)
	}
	if err := queries.CreateSession(ctx, query.CreateSessionParams{
		ID: sessionID, AccountID: accountID, PlayerID: playerID,
		TokenDigest: session.digest.Bytes(), ExpiresAt: timestamptz(session.ExpiresAt),
	}); err != nil {
		return Session{}, fmt.Errorf("create registration session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit account registration: %w", err)
	}
	return session.Session, nil
}

// Authenticate resolves a persisted, active session digest to its principal.
func (s *Service) Authenticate(ctx context.Context, digest Digest) (Principal, error) {
	session, err := query.New(s.pool).GetAuthenticatedSession(ctx, digest.Bytes())
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrSessionExpired
	}
	if err != nil {
		return Principal{}, fmt.Errorf("get authenticated session: %w", err)
	}
	return Principal{
		AccountID: session.AccountID.String(),
		PlayerID:  session.PlayerID.String(),
	}, nil
}

// Login verifies a credential pair and returns a newly issued active session.
func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	email, err := normalizeCredentials(email, password)
	if err != nil {
		return Session{}, err
	}

	account, err := query.New(s.pool).GetAccountForLogin(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, fmt.Errorf("get login account: %w", err)
	}
	if !VerifyPassword(account.PasswordHash, password) {
		return Session{}, ErrInvalidCredentials
	}

	session, err := newSession(s.sessionTTL)
	if err != nil {
		return Session{}, err
	}
	sessionID, err := randomUUID()
	if err != nil {
		return Session{}, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin login session: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := query.New(tx).CreateSession(ctx, query.CreateSessionParams{
		ID: sessionID, AccountID: account.AccountID, PlayerID: account.PlayerID,
		TokenDigest: session.digest.Bytes(), ExpiresAt: timestamptz(session.ExpiresAt),
	}); err != nil {
		return Session{}, fmt.Errorf("create login session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit login session: %w", err)
	}
	return session.Session, nil
}

type issuedSession struct {
	Session
	digest Digest
}

func newSession(ttl time.Duration) (issuedSession, error) {
	if ttl <= 0 {
		return issuedSession{}, fmt.Errorf("session TTL must be positive")
	}
	token, digest, err := NewToken()
	if err != nil {
		return issuedSession{}, err
	}
	return issuedSession{
		Session: Session{Token: token, ExpiresAt: time.Now().UTC().Add(ttl)},
		digest:  digest,
	}, nil
}

func normalizeCredentials(email, password string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if len(normalized) > maxEmailLength || !validEmail(normalized) || !validPassword(password) {
		return "", ErrInvalidInput
	}
	return normalized, nil
}

func validEmail(email string) bool {
	at := strings.LastIndexByte(email, '@')
	return at > 0 && at < len(email)-3 &&
		strings.Count(email, "@") == 1 &&
		!strings.ContainsAny(email, " \t\r\n") &&
		strings.Contains(email[at+1:], ".")
}

func newPlayerPublicID() (string, error) {
	for {
		bytes := make([]byte, 12)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate player public ID: %w", err)
		}
		id := "plr_" + base64.RawURLEncoding.EncodeToString(bytes)
		if c := id[len("plr_")]; c == '-' || c == '_' {
			continue
		}
		return id, nil
	}
}

func randomUUID() (pgtype.UUID, error) {
	var value pgtype.UUID
	if _, err := rand.Read(value.Bytes[:]); err != nil {
		return pgtype.UUID{}, fmt.Errorf("generate authentication UUID: %w", err)
	}
	value.Bytes[6] = (value.Bytes[6] & 0x0f) | 0x40
	value.Bytes[8] = (value.Bytes[8] & 0x3f) | 0x80
	value.Valid = true
	return value, nil
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}
