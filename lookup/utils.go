package lookup

func WrapSecret(secret string) (result string) {
	if ShowSecretsInLog {
		return secret
	} else {
		return "***"
	}
}
