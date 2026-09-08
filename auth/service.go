package auth

type AuthService interface {
	ValidateCredentials(clientID, clientSecret string) bool
	GenerateToken(username, password string) (token string, expiresIn int, err error)
	ValidateToken(token string) (clientID string, err error)
}
