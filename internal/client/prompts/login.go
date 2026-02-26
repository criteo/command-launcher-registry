package prompts

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// PromptUsername prompts for username (visible input)
func PromptUsername() (string, error) {
	fmt.Print("Username: ")
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read username: %w", err)
	}
	return strings.TrimSpace(username), nil
}

// PromptPassword prompts for password (hidden input)
func PromptPassword() (string, error) {
	fmt.Print("Password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Print newline after hidden input
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(password), nil
}

// AuthMethod represents the authentication method chosen by the user
type AuthMethod int

const (
	AuthMethodUsernamePassword AuthMethod = 1
	AuthMethodJWT              AuthMethod = 2
)

// PromptAuthMethod prompts the user to choose an authentication method
func PromptAuthMethod() (AuthMethod, error) {
	fmt.Println("How would you like to authenticate?")
	fmt.Println("  1. Username and password")
	fmt.Println("  2. JWT token")
	fmt.Print("Choice [1]: ")

	reader := bufio.NewReader(os.Stdin)
	choice, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("failed to read choice: %w", err)
	}

	choice = strings.TrimSpace(choice)

	// Default to username/password if empty
	if choice == "" {
		return AuthMethodUsernamePassword, nil
	}

	switch choice {
	case "1":
		return AuthMethodUsernamePassword, nil
	case "2":
		return AuthMethodJWT, nil
	default:
		return 0, fmt.Errorf("invalid choice: %s (must be 1 or 2)", choice)
	}
}

// PromptJWTToken prompts for JWT token input (visible input)
func PromptJWTToken() (string, error) {
	fmt.Print("JWT Token: ")
	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read JWT token: %w", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("JWT token cannot be empty")
	}

	return token, nil
}
