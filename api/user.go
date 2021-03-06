package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// User represents a user account known to Section
type User struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	CompanyName string `json:"company_name"`
	PhoneNumber string `json:"phone_number"`
	Verified    bool   `json:"verified"`
	Requires2FA bool   `json:"requires2fa"`
	Enforce2FA  bool   `json:"enforce2fa"`
}

// CurrentUser returns details for the currently authenticated user
func CurrentUser() (u User, err error) {
	ur := BaseURL()
	ur.Path += "/user"

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	resp, err := request(ctx, http.MethodGet, ur, nil)
	if err != nil {
		return u, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case 401:
			return u, ErrStatusUnauthorized
		case 403:
			return u, ErrStatusForbidden
		default:
			return u, prettyTxIDError(resp)
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return u, err
	}

	err = json.Unmarshal(body, &u)
	if err != nil {
		return u, err
	}
	return u, err
}
