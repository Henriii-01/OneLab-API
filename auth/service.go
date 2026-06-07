package auth

type AuthService interface {
	ValidateCredentials(clientID, clientSecret string) bool
	GenerateToken(clientID string) (token string, expiresIn int, err error)
	ValidateToken(token string) (clientID string, err error)
}
