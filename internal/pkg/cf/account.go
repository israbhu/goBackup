package cf

import (
	"fmt"
	"strings"
)

//******* This struct contains the data needed to access the cloudflare infrastructure. It is stored on drive in the file preferences.toml *****
type Account struct {
	//cloudflare account information
	// namespace is called the "namespace id" on the cloudflare website for Workers KV
	// account is called "account id" on the cloudflare dashboard
	// key is also called the "global api key" on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	// Token is used instead of the key and created on cloudflare at https://dash.cloudflare.com/profile/api-tokens
	// email is the email associated with your cloudflare account

	AccountID, Data, Email, Namespace, Key, Token, Location, Zip, Backup string

	// Whether to disable all network interactions.
	LocalOnly bool
}

//validate that the account information has all the correct fields
func (a *Account) Validate() error {
	msgs := []string{}

	//check the required fields are not blank
	if a.Email == "" {
		msgs = append(msgs, "Email information is empty. Please specify in preferences file or command line flag.")
	}
	if a.Namespace == "" {
		msgs = append(msgs, "Namespace information is empty. Please specify in preferences file or command line flag.")
	}
	if a.AccountID == "" {
		msgs = append(msgs, "Account information is empty. Please specify in preferences file or command line flag.")
	}
	if a.Key == "" && a.Token == "" {
		msgs = append(msgs, "Key and Token are empty. Please specify in preferences file or command line flag.")
	}

	if len(msgs) > 0 {
		return fmt.Errorf("Account Settings did not validate: \n%s", strings.Join(msgs, "\n"))
	}

	return nil
}
