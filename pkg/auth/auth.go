package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

func (a *AuthManager) composeSessionKey(sessionid string) string {
	return fmt.Sprintf("sessionid_%s", sessionid)
}

func (a *AuthManager) composeUserKey(username string) string {
	return fmt.Sprintf("user_%s", username)
}

func (a *AuthManager) cacheGet(ctx context.Context, key string) (User, error) {
	var user User
	res, err := a.cache.Get(ctx, key).Result()
	if err != nil {
		return user, err
	}
	json.Unmarshal([]byte(res), &user)
	return user, err
}

func (a *AuthManager) cacheSetSession(ctx context.Context, session Session, user User) error {
	key := a.composeSessionKey(session.Id)
	now, _ := utils.GetNowTz()
	ttl := session.ValidThrough.Sub(now)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return a.cache.Set(ctx, key, data, ttl).Err()
}

func (a *AuthManager) cacheSetUser(ctx context.Context, user User) error {
	key := a.composeUserKey(user.Username)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return a.cache.Set(ctx, key, data, 0).Err()
}

// GetUserBySessionId
func (a *AuthManager) GetUserBySessionId(ctx context.Context, sessionid string) (User, error) {
	var u User
	u, err := a.cacheGet(ctx, a.composeSessionKey(sessionid))
	if err == nil {
		return u, nil
	}
	query := `SELECT u.username, u.password, s.valid_through
		FROM users u JOIN sessions s 
		ON u.username = s.username 
		WHERE s.id = $1 AND s.valid_through > $2`
	now, err := utils.GetNowTz()
	if err != nil {
		return User{}, err
	}
	s := Session{
		Id:           sessionid,
		ValidThrough: now,
		Username:     "",
	}
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err = a.dbpool.QueryRow(queryCtx, query, sessionid, now).Scan(&u.Username, &u.Password, &s.ValidThrough)
	if err != nil {
		return User{}, err
	}
	s.Username = u.Username
	a.cacheSetSession(ctx, s, u)
	return u, nil
}

func (a *AuthManager) encryptPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	hashedBytes := hash.Sum(nil)
	return hex.EncodeToString(hashedBytes)
}

func (a *AuthManager) CreateUser(ctx context.Context, username, password string) (User, error) {
	u := User{Username: username, Password: a.encryptPassword(password)}
	query := "INSERT INTO users (username, password) VALUES ($1, $2)"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	_, err := a.dbpool.Exec(queryCtx, query, u.Username, u.Password)
	if err != nil {
		return u, err
	}
	a.cacheSetUser(ctx, u)
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
	u, err := a.cacheGet(ctx, a.composeUserKey(username))
	if err == nil {
		return u, nil
	}
	u = User{Username: username}
	query := "SELECT password FROM users WHERE username = $1"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err = a.dbpool.QueryRow(queryCtx, query, username).Scan(&u.Password)
	if err != nil {
		return u, err
	}
	a.cacheSetUser(ctx, u)
	return u, nil
}

func (a *AuthManager) createSesssion(ctx context.Context, u User) (Session, error) {
	now, err := utils.GetNowTz()
	if err != nil {
		return Session{}, err
	}
	expirationDate := now.AddDate(0, 0, 10)
	newSession := Session{ValidThrough: expirationDate, Username: u.Username}
	query := "INSERT INTO sessions (valid_through, username) VALUES ($1, $2) RETURNING id"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	err = a.dbpool.QueryRow(queryCtx, query, newSession.ValidThrough, newSession.Username).Scan(&newSession.Id)
	if err != nil {
		return Session{}, err
	}
	a.cacheSetSession(ctx, newSession, u)
	return newSession, nil
}

func (a *AuthManager) comparePasswords(u User, password string) bool {
	return u.Password == a.encryptPassword(password)
}

func (a *AuthManager) invalidateUserSessions(ctx context.Context, u User) error {
	now, err := utils.GetNowTz()
	if err != nil {
		return err
	}
	query := "UPDATE sessions SET valid_through = $1 WHERE username = $2 RETURNING id"
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	rows, err := a.dbpool.Query(queryCtx, query, now, u.Username)
	if err != nil {
		return err
	}
	defer rows.Close()
	var sessionIds []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		sessionIds = append(sessionIds, a.composeSessionKey(id))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = a.cache.Del(ctx, sessionIds...).Result()
	return err
}

func (a *AuthManager) ChangePassword(ctx context.Context, username, currentPassword, newPassword string) (User, error) {
	user, err := a.getUserByUsername(ctx, username)
	if err != nil {
		return user, err
	}
	if !a.comparePasswords(user, currentPassword) {
		return User{}, errInvalidCredentials
	}
	query := "UPDATE users SET password = $1 WHERE username = $2"
	encryptedPassword := a.encryptPassword(newPassword)
	queryCtx, cancel := context.WithTimeout(ctx, QUERY_TIMEOUT)
	defer cancel()
	_, err = a.dbpool.Exec(queryCtx, query, encryptedPassword, username)
	if err != nil {
		return user, err
	}
	user.Password = encryptedPassword
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
	if err != nil {
		return err
	}
	_, err = a.cache.Del(ctx, a.composeSessionKey(sessionId)).Result()
	return err
}
