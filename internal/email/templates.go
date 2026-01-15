package email

import "fmt"

func VerificationEmail(link string) (string, string) {
	return "Verify your email",
		fmt.Sprintf(
			"Welcome!\n\nVerify your email by clicking:\n%s\n\nThis link expires in 30 minutes.",
			link,
		)
}

func PasswordResetEmail(link string) (string, string) {
	return "Reset your password",
		fmt.Sprintf(
			"You requested a password reset.\n\nReset here:\n%s\n\nThis link expires in 30 minutes.",
			link,
		)
}
