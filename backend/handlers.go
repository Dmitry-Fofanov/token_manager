package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret       = []byte(os.Getenv("SECRET_KEY"))
	refreshTokenTTL = time.Hour * 24
	accessTokenTTL  = time.Hour
	smtpUser        = os.Getenv("SMTP_USER")
	smtpAuth        = smtp.PlainAuth(
		"",
		smtpUser,
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_HOST"),
	)
	smtpAddress = os.Getenv("SMTP_HOST") + ":" + os.Getenv("SMTP_PORT")
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AccessTokenClaims struct {
	TokenId string `json:"token_id"`
	UserId  string `json:"user_id"`
	IP      string `json:"ip"`
	jwt.RegisteredClaims
}

func SendEmailToUser(userId, message string) error {
	var email string
	err := db.QueryRow(`
		SELECT email
		FROM users
		WHERE id = $1`,
		userId).
		Scan(&email)
	if err != nil {
		return err
	}

	if debug {
		log.Printf("Отправляю сообщение \"%s\" по почтовому адресу %s", message, email)
	} else {
		smtp.SendMail(smtpAddress, smtpAuth, smtpUser, []string{email}, []byte(message))
	}

	return nil
}

func generateTokenPair(ip, userId string) (*TokenPair, error) {
	refreshTokenBytes := make([]byte, 32)
	_, err := rand.Read(refreshTokenBytes)
	if err != nil {
		return nil, err
	}

	encodedRefreshToken := base64.StdEncoding.EncodeToString(refreshTokenBytes)

	claims := AccessTokenClaims{
		TokenId: uuid.New().String(),
		UserId:  userId,
		IP:      ip,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{time.Now().Add(accessTokenTTL)},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	accessToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshTokenHash, err := bcrypt.GenerateFromPassword(
		[]byte(encodedRefreshToken),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		INSERT INTO refresh_tokens
		(token_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		claims.TokenId,
		refreshTokenHash,
		time.Now().Add(refreshTokenTTL),
	)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: encodedRefreshToken,
	}, nil
}

func RetrieveTokensHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var credentials struct {
			UserId string `json:"user_id"`
		}

		err := json.NewDecoder(r.Body).Decode(&credentials)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var userExists bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1
				FROM users
				WHERE id = $1
			 )`,
			credentials.UserId).
			Scan(&userExists)

		if err != nil || !userExists {
			http.Error(w, "Неверный ID пользователя", http.StatusBadRequest)
			return
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		tokenPair, err := generateTokenPair(ip, credentials.UserId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(tokenPair)
	}
}

func RefreshTokensHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenPair TokenPair

		err := json.NewDecoder(r.Body).Decode(&tokenPair)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		token, err := jwt.ParseWithClaims(
			tokenPair.AccessToken,
			&AccessTokenClaims{},
			func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			},
		)

		if err != nil {
			http.Error(w, "Неверный Access-токен", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*AccessTokenClaims)
		if !ok || !token.Valid {
			http.Error(w, "Неверный Access-токен", http.StatusUnauthorized)
			return
		}

		var (
			expiresAt        time.Time
			refreshTokenHash string
		)
		err = db.QueryRow(`
			SELECT token_hash, expires_at
			FROM refresh_tokens
			WHERE token_id = $1`,
			claims.TokenId).
			Scan(&refreshTokenHash, &expiresAt)

		if err != nil {
			http.Error(w, "Неверный Refresh-токен", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword(
			[]byte(refreshTokenHash),
			[]byte(tokenPair.RefreshToken),
		)

		if err != nil {
			http.Error(w, "Неверный Refresh-токен", http.StatusUnauthorized)
			return
		}

		if time.Now().After(expiresAt) {
			http.Error(w, "Неверный Refresh-токен", http.StatusUnauthorized)
			return
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		newTokenPair, err := generateTokenPair(ip, claims.UserId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if claims.IP != ip {
			SendEmailToUser(
				claims.UserId,
				fmt.Sprintf(
					"ВНИМАНИЕ, выполнен вход с нового IP-адреса: %s",
					ip,
				),
			)
		} else if debug {
			SendEmailToUser(
				claims.UserId,
				fmt.Sprintf(
					"Проверка сервиса почты, IP-адрес: %s",
					ip,
				),
			)
		}

		db.Exec(`
			DELETE FROM refresh_tokens
			WHERE token_hash = $1`,
			claims.TokenId,
		)

		json.NewEncoder(w).Encode(newTokenPair)
	}
}
