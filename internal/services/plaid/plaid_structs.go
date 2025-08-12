package plaid


type PlaidOpts struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"secret"`
	Environment  string `json:"environment"`
	Version      string `json:"version"`
}

type ExchangeTokenRequest struct {
    PublicToken string `json:"public_token"`
    UserID      string `json:"user_id"`
}

type ExchangeTokenResponse struct {
    AccessToken string `json:"access_token"`
    ItemID      string `json:"item_id"`
    AccountID   string `json:"account_id"`
}

type Account struct {
    Name        string  `json:"name"`
    Subtype     string  `json:"subtype"`
    Mask        string  `json:"mask"`
    Type        string  `json:"type"`
    Institution *string `json:"institution"`
}

type AccountWithBalance struct {
    *Account
    Balance float64 `json:"balance"`
}

type CreatePlaidBankAccountResponse struct {
    AccountID   string `json:"AccountID"`
    AccessToken string `json:"AccessToken"`
    ItemID      string `json:"ItemID"`
}