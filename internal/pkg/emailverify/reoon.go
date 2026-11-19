package emailverify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Reoon verifies addresses through Reoon Email Verifier's single-address API
// (power mode) using the workspace's own API key.
type Reoon struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

const (
	reoonAPI     = "https://emailverifier.reoon.com"
	reoonTimeout = 60 * time.Second
)

// NewReoon constructs the client. baseURL is overridable for tests.
func NewReoon(apiKey, baseURL string) *Reoon {
	if baseURL == "" {
		baseURL = reoonAPI
	}
	return &Reoon{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: reoonTimeout, CheckRedirect: refuseInsecureRedirect},
	}
}

type reoonVerifyResponse struct {
	Email            string `json:"email"`
	Status           string `json:"status"`
	OverallScore     int    `json:"overall_score"`
	IsSafeToSend     bool   `json:"is_safe_to_send"`
	IsValidSyntax    bool   `json:"is_valid_syntax"`
	IsDisposable     bool   `json:"is_disposable"`
	IsRoleAccount    bool   `json:"is_role_account"`
	CanConnectSMTP   bool   `json:"can_connect_smtp"`
	HasInboxFull     bool   `json:"has_inbox_full"`
	IsCatchAll       bool   `json:"is_catch_all"`
	IsDeliverable    bool   `json:"is_deliverable"`
	IsDisabled       bool   `json:"is_disabled"`
	IsSpamtrap       bool   `json:"is_spamtrap"`
	MXAcceptsMail    bool   `json:"mx_accepts_mail"`
	VerificationMode string `json:"verification_mode"`
	Reason           string `json:"reason"`
	Error            string `json:"error"`
	Message          string `json:"message"`
}

type reoonBalanceResponse struct {
	APIStatus               string `json:"api_status"`
	RemainingDailyCredits   int    `json:"remaining_daily_credits"`
	RemainingInstantCredits int    `json:"remaining_instant_credits"`
	Status                  string `json:"status"`
	Reason                  string `json:"reason"`
	Error                   string `json:"error"`
	Message                 string `json:"message"`
}

// Check verifies one address with power mode.
func (r *Reoon) Check(ctx context.Context, email string) (Result, error) {
	now := time.Now().UTC()
	res := Result{Email: strings.ToLower(strings.TrimSpace(email)), CheckedAt: now, Status: StatusUnknown, Provider: ProviderReoon}

	q := url.Values{}
	q.Set("email", res.Email)
	q.Set("key", r.apiKey)
	q.Set("mode", "power")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/api/v1/verify?"+q.Encode(), nil)
	if err != nil {
		err = reoonWithoutURL(err)
		res.Reason = "reoon request failed: " + err.Error()
		return res, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		err = reoonWithoutURL(err)
		res.Reason = "reoon unreachable: " + err.Error()
		return res, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		res.Reason = "reoon rejected the API key"
		return res, ErrProviderKey
	}
	if resp.StatusCode == http.StatusPaymentRequired {
		res.Reason = "reoon: no credits left"
		return res, ErrProviderCredits
	}
	if resp.StatusCode != http.StatusOK {
		res.Reason = fmt.Sprintf("reoon answered HTTP %d", resp.StatusCode)
		return res, fmt.Errorf("reoon: http %d", resp.StatusCode)
	}
	var out reoonVerifyResponse
	if err := json.Unmarshal(body, &out); err != nil {
		res.Reason = "reoon answered with an unreadable body"
		return res, err
	}
	if msg := firstNonEmpty(out.Error, out.Message, out.Reason); msg != "" && out.Status == "" {
		return res, r.accountError(&res, msg)
	}

	v, ok := NormalizeExternal(ProviderReoon, out.Status)
	if !ok {
		res.Reason = "reoon result not recognised: " + out.Status
		return res, nil
	}
	res.Status, res.SubStatus = v.Status, v.SubStatus
	res.IsCatchAll = out.IsCatchAll || v.SubStatus == SubStatusCatchAll
	res.HasMX = out.MXAcceptsMail || out.CanConnectSMTP
	if out.OverallScore > 0 {
		res.Confidence = out.OverallScore
	}
	reason := out.Status
	if out.IsRoleAccount && res.SubStatus == SubStatusNone && res.Status == StatusRisky {
		res.SubStatus = SubStatusRole
	}
	res.Reason = "reoon: " + reason
	return res, nil
}

// ObservesBalance is true: check-account-balance reports remaining credits.
func (r *Reoon) ObservesBalance() bool { return true }

// Account validates the key and returns instant plus daily credits remaining.
func (r *Reoon) Account(ctx context.Context) (*int, error) {
	q := url.Values{}
	q.Set("key", r.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/api/v1/check-account-balance/?"+q.Encode(), nil)
	if err != nil {
		return nil, reoonWithoutURL(err)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, reoonWithoutURL(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrProviderKey
	}
	if resp.StatusCode == http.StatusPaymentRequired {
		return nil, ErrProviderCredits
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reoon: http %d", resp.StatusCode)
	}
	var out reoonBalanceResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if msg := firstNonEmpty(out.Error, out.Message, out.Reason); msg != "" && strings.EqualFold(out.Status, "error") {
		return nil, r.accountError(nil, msg)
	}
	if out.APIStatus != "" && !strings.EqualFold(out.APIStatus, "active") {
		return nil, ErrProviderKey
	}
	n := out.RemainingInstantCredits + out.RemainingDailyCredits
	return &n, nil
}

func (r *Reoon) accountError(res *Result, msg string) error {
	lower := strings.ToLower(msg)
	var err error
	switch {
	case strings.Contains(lower, "api key") || strings.Contains(lower, "apikey") || strings.Contains(lower, "unauthori") || strings.Contains(lower, "invalid key"):
		err = ErrProviderKey
	case strings.Contains(lower, "credit"):
		err = ErrProviderCredits
	default:
		err = fmt.Errorf("reoon: %s", msg)
	}
	if res != nil {
		res.Reason = "reoon: " + msg
	}
	return err
}

func reoonWithoutURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("reoon: %s: %w", ue.Op, ue.Err)
	}
	return err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
