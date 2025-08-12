package plaid

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plaid/plaid-go/v12/plaid"
	"go.uber.org/zap"
)

const (
	PlaidTokenProcessorIdentifier = ""
)

type Plaid struct {
	client *plaid.APIClient
}

const (
	PaymentMethodStatusPendingAutomaticVerification = "pending_automatic_verification"
	PaymentMethodStatusPendingManualVerification    = "pending_manual_verification"
	PaymentMethodStatusManuallyVerified             = "manually_verified"
	PaymentMethodStatusAutomaticallyVerified        = "automatically_verified"
)

type PlaidService interface {
	CreateLinkToken(ctx context.Context, userID string) (string, error)
	ExchangePublicToken(ctx context.Context, publicToken string) (*ExchangeTokenResponse, error)
	CreateProcessorToken(ctx context.Context, accessToken, accountID string) (string, error)
	GetAccount(ctx context.Context, accessToken, accountID string) (*Account, error)
	GetAccountWithBalance(ctx context.Context, accessToken, accountID string) (*AccountWithBalance, error)
	CreatePlaidBankAccount(ctx context.Context) (*CreatePlaidBankAccountResponse, error)
	DeletePlaidBankAccount(ctx context.Context, accessToken string) (*string, error)
	CreateStripeToken(ctx context.Context, accessToken, accountID string) (*string, error)
	GetWebhookVerification(ctx context.Context, plaid *plaid.WebhookVerificationKeyGetRequest) (*plaid.JWKPublicKey, error)
	VerifyWebhook(webhookBody string, headers map[string]string) (bool, error)
	IsBalanceCheckSupported(ctx context.Context, accessToken, accountID string) (bool, error)
}

func New(opts *PlaidOpts) PlaidService {
	config := plaid.NewConfiguration()
	config.AddDefaultHeader("PLAID-CLIENT-ID", opts.ClientID)
	config.AddDefaultHeader("PLAID-SECRET", opts.ClientSecret)

	switch opts.Environment {
	case "production":
		config.UseEnvironment(plaid.Production)
	case "development":
		config.UseEnvironment(plaid.Development)
	default:
		config.UseEnvironment(plaid.Sandbox)
	}

	client := plaid.NewAPIClient(config)

	return &Plaid{client}
}

// CreateLinkToken generates a new Plaid Link token for the specified user ID.
// The link token is used by the frontend to initialize the Plaid Link widget,
// allowing the user to securely connect their bank account.
//
// This method should be called before launching Plaid Link on the client side.
func (p *Plaid) CreateLinkToken(ctx context.Context, userID string) (string, error) {
	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: userID,
	}

	req := plaid.NewLinkTokenCreateRequest(
		"Plaid Stripe App",
		"en",
		[]plaid.CountryCode{plaid.COUNTRYCODE_US},
		user,
	)

	req.SetProducts([]plaid.Products{plaid.PRODUCTS_AUTH})

	res, _, err := p.client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*req).Execute()
	if err != nil {
		return "", err
	}

	return res.GetLinkToken(), nil
}

// ExchangePublicToken exchanges a short-lived public_token received from the Plaid Link
// frontend widget for a long-lived access_token and item_id. The access_token can be
// used to retrieve account data, balances, and create processor tokens for payments.
//
// This method is typically called after a user successfully links a bank account in Plaid Link.
func (p *Plaid) ExchangePublicToken(ctx context.Context, publicToken string) (*ExchangeTokenResponse, error) {
	exchangeReq := plaid.NewItemPublicTokenExchangeRequest(publicToken)
	res, _, err := p.client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*exchangeReq).Execute()
	if err != nil {
		return nil, err
	}

	return &ExchangeTokenResponse{
		AccessToken: res.GetAccessToken(),
		ItemID:      res.GetItemId(),
		AccountID:   "",
	}, nil
}

// * Creates a Plaid processor token for a linked bank account, used to generate a Stripe bank account token.
func (p *Plaid) CreateProcessorToken(ctx context.Context, accessToken, accountID string) (string, error) {
	request := plaid.NewProcessorTokenCreateRequest(accessToken, accountID, PlaidTokenProcessorIdentifier)
	processorTokenCreateResp, _, err := p.client.PlaidApi.ProcessorTokenCreate(ctx).ProcessorTokenCreateRequest(*request).Execute()
	if err != nil {
		return "", err
	}
	return processorTokenCreateResp.ProcessorToken, nil
}

// * Fetches basic account info (like name, mask, type) from Plaid using the access token and account ID.
func (p *Plaid) GetAccount(ctx context.Context, accessToken, accountID string) (*Account, error) {
	accountRequest := plaid.NewAccountsGetRequest(accessToken)
	accountsGetResp, _, err := p.client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*accountRequest).Execute()
	if err != nil {
		return nil, err
	}
	accounts := accountsGetResp.GetAccounts()
	institutionID := accountsGetResp.Item.InstitutionId.Get()
	var institution *string
	if institutionID != nil {
		institutionRequest := plaid.NewInstitutionsGetByIdRequest(*institutionID, []plaid.CountryCode{plaid.COUNTRYCODE_US})
		instiutionGetResp, _, err := p.client.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(*institutionRequest).Execute()
		if err != nil {
			return nil, err
		}
		institution = &instiutionGetResp.Institution.Name
	}

	return &Account{Name: accounts[0].Name, Subtype: string(*accounts[0].Subtype.Get()), Mask: string(*accounts[0].Mask.Get()), Type: string(*accounts[0].Type.Ptr()), Institution: institution}, nil
}

// GetAccountWithBalance fetches a Plaid bank account and returns the current available balance
func (p *Plaid) GetAccountWithBalance(ctx context.Context, accessToken, accountID string) (*AccountWithBalance, error) {
	request := plaid.NewAccountsBalanceGetRequest(accessToken)
	resp, _, err := p.client.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(*request).Execute()
	if err != nil {
		return nil, err
	}

	accounts := resp.GetAccounts()

	if len(accounts) == 0 {
		return nil, errors.New("no accounts found for accessToken")
	}

	account := accounts[0]
	a := &Account{Name: account.Name, Subtype: string(*accounts[0].Subtype.Get())}

	balance := account.Balances.GetAvailable()
	if balance == 0 {
		balance = account.Balances.GetCurrent()
	}

	return &AccountWithBalance{
		Account: a,
		Balance: balance,
	}, nil
}

// *TESTING ONLY* CreatePlaidBankAccount is a helper method for seeding a bank account for testing.
// * this method will automatically generate a public token that is usually handled by the plaid modal
// * on the UI, and convert that into an AccountID and AccessToken for use.
func (p *Plaid) CreatePlaidBankAccount(ctx context.Context) (*CreatePlaidBankAccountResponse, error) {
	// Simulate fetching a public token
	sandboxInstitution := "ins_128026" // First Platypus
	testProducts := []plaid.Products{"auth"}
	testPassword := `{
		"override_accounts": [
		  {
		    "type": "depository",
		    "subtype": "checking",
		    "currency": "USD",
		    "numbers": {
			"ach_routing": "051405515"
		    },
		    "starting_balance": 10000,
		    "force_available_balance": 10000,
		    "meta": {
			"name": "checking 1"
		    }
		  }
		]
	      }`
	request := *plaid.NewSandboxPublicTokenCreateRequest(
		sandboxInstitution,
		testProducts,
	)

	options := *plaid.NewSandboxPublicTokenCreateRequestOptionsWithDefaults()
	user := "user_custom"
	type PlaidCustomerPassword struct {
		OverrideAccounts []plaid.OverrideAccounts `json:"override_accounts"`
	}

	options.OverrideUsername = *plaid.NewNullableString(&user)
	options.OverridePassword = *plaid.NewNullableString(&testPassword)
	request.Options = &options

	sandboxPublicTokenResp, _, err := p.client.PlaidApi.SandboxPublicTokenCreate(ctx).SandboxPublicTokenCreateRequest(
		request,
	).Execute()
	if err != nil {
		log.Printf("error calling out to plaid, %v", err)
		return nil, err
	}

	publicToken := sandboxPublicTokenResp.GetPublicToken()

	// Exchange the publicToken for an accessToken
	exchangePublicTokenResp, _, err := p.client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*plaid.NewItemPublicTokenExchangeRequest(publicToken),
	).Execute()
	if err != nil {
		return nil, err
	}

	accessToken := exchangePublicTokenResp.GetAccessToken()

	// Get Accounts
	accountsGetResp, _, err := p.client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()
	if err != nil {
		return nil, err
	}

	// Get Items
	itemGetResp, _, err := p.client.PlaidApi.ItemGet(ctx).ItemGetRequest(
		*plaid.NewItemGetRequest(accessToken),
	).Execute()
	if err != nil {
		return nil, err
	}

	accountID := accountsGetResp.GetAccounts()[0].GetAccountId()
	itemID := itemGetResp.GetItem().ItemId

	return &CreatePlaidBankAccountResponse{
		AccessToken: accessToken,
		AccountID:   accountID,
		ItemID:      itemID,
	}, nil
}

// * Removes a linked bank account by calling Plaid’s ItemRemove.
func (p *Plaid) DeletePlaidBankAccount(ctx context.Context, accessToken string) (*string, error) {
	request := plaid.NewItemRemoveRequest(accessToken)

	itemRemoveResp, _, err := p.client.PlaidApi.ItemRemove(ctx).ItemRemoveRequest(*request).Execute()
	if err != nil {
		return nil, err
	}
	return &itemRemoveResp.RequestId, nil
}

// * Checks whether the account supports Plaid’s balance product.
func (p *Plaid) IsBalanceCheckSupported(ctx context.Context, accessToken, accountID string) (bool, error) {
	accountRequest := plaid.NewAccountsGetRequest(accessToken)
	accountsGetResp, _, err := p.client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*accountRequest).Execute()
	if err != nil {
		return false, err
	}

	if len(accountsGetResp.Item.AvailableProducts) == 0 {
		return false, nil
	}

	for _, p := range accountsGetResp.Item.AvailableProducts {
		if p == plaid.PRODUCTS_BALANCE {
			return true, nil
		}
	}

	return false, nil
}

// * Uses a processor token to generate a Stripe-compatible bank account token.
func (p *Plaid) CreateStripeToken(ctx context.Context, accessToken, accountID string) (*string, error) {

	request := plaid.NewProcessorStripeBankAccountTokenCreateRequest(accessToken, accountID)
	stripeTokenResp, _, err := p.client.PlaidApi.ProcessorStripeBankAccountTokenCreate(ctx).ProcessorStripeBankAccountTokenCreateRequest(*request).Execute()

	if err != nil {
		return nil, err
	}
	return &stripeTokenResp.StripeBankAccountToken, nil
}

// * Fetches the public key from Plaid needed to verify webhook signatures.
func (p *Plaid) GetWebhookVerification(ctx context.Context, req *plaid.WebhookVerificationKeyGetRequest) (*plaid.JWKPublicKey, error) {

	webhookResp, _, err := p.client.PlaidApi.WebhookVerificationKeyGet(ctx).WebhookVerificationKeyGetRequest(*req).Execute()
	if err != nil {
		return nil, err
	}
	key := webhookResp.GetKey()
	return &key, err
}

func (p *Plaid) VerifyWebhook(webhookBody string, headers map[string]string) (bool, error) {
	ctx := context.Background()

	tokenString := headers["plaid-verification"]
	token, parts, err := p.parseToken(tokenString)

	if err != nil || !p.validateAlgorithm(token) {
		zap.L().Error(fmt.Sprintf("verify webhook: validate algorithm failed: %v", err))
		return false, err
	}

	keyCache := make(map[string]plaid.JWKPublicKey)

	isRefreshed, err := p.refreshKeys(ctx, token.Header["kid"].(string), keyCache)
	if err != nil {
		zap.L().Error(fmt.Sprintf("verify webhook: refresh keys failed: %v", err))
		return isRefreshed, err
	}

	if !p.validateToken(token, parts, keyCache) {
		zap.L().Error("verify webhook: validate token failed")
		return false, err
	}

	if !p.checkTimestamp(token) {
		zap.L().Error("verify webhook: check timestamp failed")
		return false, err
	}

	sha256Value := token.Claims.(jwt.MapClaims)["request_body_sha256"].(string)
	sha256Body := p.computeSHA256(webhookBody)

	return sha256Body == sha256Value, err
}

func (p *Plaid) parseToken(tokenString string) (*jwt.Token, []string, error) {
	return new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
}

func (p *Plaid) validateAlgorithm(token *jwt.Token) bool {
	return token.Method.Alg() == "ES256"
}

func (p *Plaid) refreshKeys(ctx context.Context, kid string, keyCache map[string]plaid.JWKPublicKey) (bool, error) {
	var err error
	if _, found := keyCache[kid]; !found {
		kidsToUpdate := make([]string, 0)

		for k, v := range keyCache {
			if v.ExpiredAt.Get() == nil {
				kidsToUpdate = append(kidsToUpdate, k)
			}
		}

		kidsToUpdate = append(kidsToUpdate, kid)

		for _, k := range kidsToUpdate {
			webhookRequest := *plaid.NewWebhookVerificationKeyGetRequest(k)
			webhookResponse, err := p.GetWebhookVerification(ctx, &webhookRequest)
			if err != nil {
				zap.L().Error(fmt.Sprintf("get webhook verification failed: %v", err))
				return false, err
			}
			keyCache[k] = *webhookResponse
		}
	}
	return true, err
}

func (p *Plaid) validateToken(token *jwt.Token, parts []string, keyCache map[string]plaid.JWKPublicKey) bool {
	kid := token.Header["kid"].(string)
	if _, found := keyCache[kid]; !found {
		return false
	}

	key := keyCache[kid]

	if key.ExpiredAt.Get() != nil {
		return false
	}

	publicKey := p.createPublicKey(key)
	tokenVerification := jwt.SigningMethodES256.Verify(
		parts[0]+"."+parts[1],
		parts[2],
		publicKey,
	)

	return tokenVerification == nil
}

func (p *Plaid) createPublicKey(key plaid.JWKPublicKey) *ecdsa.PublicKey {
	publicKey := new(ecdsa.PublicKey)
	publicKey.Curve = elliptic.P256()
	x, _ := base64.URLEncoding.DecodeString(key.X + "=")
	xc := new(big.Int)
	publicKey.X = xc.SetBytes(x)
	y, _ := base64.URLEncoding.DecodeString(key.Y + "=")
	yc := new(big.Int)
	publicKey.Y = yc.SetBytes(y)
	return publicKey
}

// Ensure webhook is not older than 5 minutes
func (p *Plaid) checkTimestamp(token *jwt.Token) bool {
	claims := token.Claims.(jwt.MapClaims)
	iat := claims["iat"].(float64)
	timeSinceIat := float64(time.Now().Unix()) - iat
	return timeSinceIat <= 300
}

func (p *Plaid) computeSHA256(webhookBody string) string {
	sum := sha256.Sum256([]byte(webhookBody))
	return hex.EncodeToString(sum[:])
}

func IsVerifiedStatus(s string) bool {
	return slices.Contains([]string{PaymentMethodStatusAutomaticallyVerified, PaymentMethodStatusManuallyVerified}, strings.ToLower(s))
}
