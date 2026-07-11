package adminauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"

	appcrypto "github.com/djalmajr/tenancit/server/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const credentialBytes = 32

type sessionStore interface {
	CreateAdminSession(context.Context, CreateSessionParams) (StoredSession, error)
	AuthenticateAdminSession(context.Context, AuthenticateSessionParams) (StoredSession, error)
	RevokeAdminSession(context.Context, uuid.UUID, time.Time) error
}

type CreateSessionParams struct {
	TokenHash, CSRFTokenHash, ActorIssuer, ActorSubject string
	CSRFTokenCipher, CSRFNonce                          []byte
	CSRFKeyVersion                                      int16
	ActorLabel                                          *string
	Roles, Permissions                                  []string
	CreatedAt, ExpiresAt, IdleExpiresAt                 time.Time
}

type AuthenticateSessionParams struct {
	TokenHash   string
	UsedAt      time.Time
	IdleSeconds int32
}

type StoredSession struct {
	ID                                                  uuid.UUID
	TokenHash, CSRFTokenHash, ActorIssuer, ActorSubject string
	CSRFTokenCipher, CSRFNonce                          []byte
	CSRFKeyVersion                                      int16
	ActorLabel                                          *string
	Roles, Permissions                                  []string
	CreatedAt, LastUsedAt, ExpiresAt, IdleExpiresAt     time.Time
	RevokedAt                                           pgtype.Timestamptz
}

type SessionIdentity struct {
	Issuer        string
	Subject       string
	Label         string
	SessionID     string
	Roles         []Role
	Permissions   []string
	CSRFTokenHash string
	CSRFToken     string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
}

type CreatedSession struct {
	Token     string
	CSRFToken string
	Identity  SessionIdentity
}

type SessionManager struct {
	store       sessionStore
	cryptor     *appcrypto.Cryptor
	random      io.Reader
	now         func() time.Time
	absoluteTTL time.Duration
	idleTTL     time.Duration
}

func NewSessionManager(store sessionStore, cryptor *appcrypto.Cryptor, random io.Reader, now func() time.Time, absoluteTTL, idleTTL time.Duration) *SessionManager {
	if random == nil {
		random = rand.Reader
	}
	if now == nil {
		now = time.Now
	}
	return &SessionManager{store: store, cryptor: cryptor, random: random, now: now, absoluteTTL: absoluteTTL, idleTTL: idleTTL}
}

func (m *SessionManager) Create(ctx context.Context, identity SessionIdentity) (CreatedSession, error) {
	if m.store == nil || m.cryptor == nil || strings.TrimSpace(identity.Issuer) == "" || strings.TrimSpace(identity.Subject) == "" || len(identity.Roles) == 0 || len(identity.Permissions) == 0 {
		return CreatedSession{}, errors.New("complete session identity and store are required")
	}
	if m.absoluteTTL <= 0 || m.idleTTL <= 0 || m.idleTTL > m.absoluteTTL {
		return CreatedSession{}, errors.New("invalid session expiry policy")
	}
	token, err := randomCredential(m.random)
	if err != nil {
		return CreatedSession{}, err
	}
	csrfToken, err := randomCredential(m.random)
	if err != nil {
		return CreatedSession{}, err
	}
	csrfEncrypted, err := m.cryptor.Encrypt(csrfToken)
	if err != nil {
		return CreatedSession{}, err
	}
	now := m.now().UTC()
	roles := make([]string, 0, len(identity.Roles))
	for _, role := range identity.Roles {
		roles = append(roles, string(role))
	}
	label := optionalLabel(identity.Label)
	stored, err := m.store.CreateAdminSession(ctx, CreateSessionParams{
		TokenHash: HashCredential(token), CSRFTokenHash: HashCredential(csrfToken),
		CSRFTokenCipher: csrfEncrypted.Cipher, CSRFNonce: csrfEncrypted.Nonce,
		CSRFKeyVersion: int16(csrfEncrypted.KeyVersion),
		ActorIssuer:    identity.Issuer, ActorSubject: identity.Subject, ActorLabel: label,
		Roles: roles, Permissions: append([]string(nil), identity.Permissions...), CreatedAt: now,
		ExpiresAt: now.Add(m.absoluteTTL), IdleExpiresAt: now.Add(m.idleTTL),
	})
	if err != nil {
		return CreatedSession{}, err
	}
	createdIdentity, err := m.identityFromSession(stored)
	if err != nil {
		return CreatedSession{}, err
	}
	return CreatedSession{Token: token, CSRFToken: csrfToken, Identity: createdIdentity}, nil
}

func (m *SessionManager) Authenticate(ctx context.Context, token string) (SessionIdentity, error) {
	if m.store == nil || strings.TrimSpace(token) == "" || m.idleTTL <= 0 {
		return SessionIdentity{}, errors.New("invalid session credential")
	}
	stored, err := m.store.AuthenticateAdminSession(ctx, AuthenticateSessionParams{
		TokenHash: HashCredential(token), UsedAt: m.now().UTC(), IdleSeconds: int32(m.idleTTL / time.Second),
	})
	if err != nil {
		return SessionIdentity{}, err
	}
	return m.identityFromSession(stored)
}

func (m *SessionManager) Revoke(ctx context.Context, sessionID string) error {
	id, err := uuid.Parse(sessionID)
	if err != nil || m.store == nil {
		return errors.New("invalid admin session")
	}
	return m.store.RevokeAdminSession(ctx, id, m.now().UTC())
}

func (m *SessionManager) identityFromSession(stored StoredSession) (SessionIdentity, error) {
	roles := make([]Role, 0, len(stored.Roles))
	for _, role := range stored.Roles {
		roles = append(roles, Role(role))
	}
	label := ""
	if stored.ActorLabel != nil {
		label = *stored.ActorLabel
	}
	csrfToken, err := m.cryptor.Decrypt(appcrypto.Encrypted{
		Cipher: stored.CSRFTokenCipher, Nonce: stored.CSRFNonce, KeyVersion: int(stored.CSRFKeyVersion),
	})
	if err != nil {
		return SessionIdentity{}, err
	}
	return SessionIdentity{
		Issuer: stored.ActorIssuer, Subject: stored.ActorSubject, Label: label,
		SessionID: stored.ID.String(), Roles: roles,
		Permissions:   append([]string(nil), stored.Permissions...),
		CSRFTokenHash: stored.CSRFTokenHash, CSRFToken: csrfToken,
		ExpiresAt: stored.ExpiresAt, IdleExpiresAt: stored.IdleExpiresAt,
	}, nil
}

func HashCredential(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func ValidateCSRF(expectedHash, presented string) bool {
	if expectedHash == "" || presented == "" {
		return false
	}
	actualHash := HashCredential(presented)
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(actualHash)) == 1
}

func randomCredential(source io.Reader) (string, error) {
	buffer := make([]byte, credentialBytes)
	if _, err := io.ReadFull(source, buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func optionalLabel(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
