package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/JustDean/sam/pkg/postgres"
	redis_utils "github.com/JustDean/sam/pkg/redis"
	"github.com/JustDean/sam/pkg/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	QUERY_TIMEOUT = 10 * time.Second
)

var errInvalidCredentials = errors.New("invalid credentials")

func SetAuthManager(c AuthManagerConfig) (*AuthManager, error) {
	dbpool, err := postgres.SetPostgresPool(c.Db)
	if err != nil {
		return nil, err
	}
	cache, err := redis_utils.SetRedisPool(c.Cache)
	if err != nil {
		return nil, err
	}
	return &AuthManager{
		dbpool, cache,
	}, nil
}

type AuthManager struct {
	dbpool *pgxpool.Pool
	cache  *redis.Client
}

func (a *AuthManager) Run(ctx context.Context) {
	log.Println("Starting Auth Manager")
	<-ctx.Done()
	log.Println("Stopping Auth Manager")
	a.dbpool.Close()
	a.cache.Close()
	log.Println("Auth Manager is stopped")
}

// GetUserBySessionId
func (a *AuthManager) GetUserBySessionId(ctx context.Context, sessionid string) (User, error) {
	// TODO impliment cache
	var u User
	query := `SELECT u.id, u.username, u.password 
		FROM users u JOIN sessions s 
		ON u.id = s.user_id 
		WHERE s.id = $1 AND s.valid_through > $2`
	now, err := utils.GetNowTz()
	if err != nil {
		return User{}, err
	}
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err = a.dbpool.QueryRow(queryCtx, query, sessionid, now).Scan(&u.Id, &u.Username, &u.password)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (a *AuthManager) encryptPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	hashedBytes := hash.Sum(nil)
	return hex.EncodeToString(hashedBytes)
}

func (a *AuthManager) CreateUser(ctx context.Context, username, password string) (User, error) {
	u := User{Username: username, password: a.encryptPassword(password)}
	query := "INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err := a.dbpool.QueryRow(queryCtx, query, u.Username, u.password).Scan(&u.Id)
	if err != nil {
		return u, err
	}
	return u, nil
}

func (a *AuthManager) LoginUser(ctx context.Context, username, password string) (Session, error) {
	user, err := a.getUserByUsername(ctx, username)
	if err != nil {
		return Session{}, err
	}
	if !a.comparePasswords(user, password) {
		return Session{}, errInvalidCredentials
	}
	s, err := a.createSesssion(ctx, user)
	if err != nil {
		return Session{}, err
	}
	return s, nil

}

func (a *AuthManager) getUserByUsername(ctx context.Context, username string) (User, error) {
	// TODO cache
	u := User{Username: username}
	query := "SELECT id, password FROM users WHERE username = $1"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err := a.dbpool.QueryRow(queryCtx, query, username).Scan(&u.Id, &u.password)
	if err != nil {
		return u, err
	}
	return u, nil
}

func (a *AuthManager) getUserById(ctx context.Context, id string) (User, error) {
	// TODO cache
	u := User{Id: id}
	query := "SELECT username, password FROM users WHERE id = $1"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err := a.dbpool.QueryRow(queryCtx, query, id).Scan(&u.Username, &u.password)
	if err != nil {
		return u, err
	}
	return u, nil
}

func (a *AuthManager) createSesssion(ctx context.Context, u User) (Session, error) {
	now, err := utils.GetNowTz()
	if err != nil {
		return Session{}, err
	}
	expirationDate := now.AddDate(0, 0, 10)
	newSession := Session{ValidThrough: expirationDate, UserId: u.Id}
	query := "INSERT INTO sessions (valid_through, user_id) VALUES ($1, $2) RETURNING id"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err = a.dbpool.QueryRow(queryCtx, query, newSession.ValidThrough, newSession.UserId).Scan(&newSession.Id)
	if err != nil {
		return Session{}, err
	}
	return newSession, nil
}

func (a *AuthManager) comparePasswords(u User, password string) bool {
	return u.password == a.encryptPassword(password)
}

func (a *AuthManager) invalidateUserSessions(ctx context.Context, u User) error {
	now, err := utils.GetNowTz()
	if err != nil {
		return err
	}
	query := "UPDATE sessions SET valid_through = $1 WHERE user_id = $2"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	_, err = a.dbpool.Exec(queryCtx, query, now, u.Id)
	return err
}

func (a *AuthManager) ChangePassword(ctx context.Context, userId, currentPassword, newPassword string) (User, error) {
	user, err := a.getUserById(ctx, userId)
	if err != nil {
		return user, err
	}
	if !a.comparePasswords(user, currentPassword) {
		return User{}, errInvalidCredentials
	}
	query := "UPDATE users SET password = $1 WHERE id = $2"
	encryptedPassword := a.encryptPassword(newPassword)
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	_, err = a.dbpool.Exec(queryCtx, query, encryptedPassword, userId)
	if err != nil {
		return user, err
	}
	user.password = encryptedPassword
	a.invalidateUserSessions(ctx, user)
	return user, nil
}

func (a *AuthManager) InvalidateSession(ctx context.Context, sessionId string) error {
	now, err := utils.GetNowTz()
	if err != nil {
		return err
	}
	query := "UPDATE sessions SET valid_through = $1 WHERE id = $2"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	_, err = a.dbpool.Exec(queryCtx, query, now, sessionId)
	return err
}
