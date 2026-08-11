package leigod

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"leigod-auto-pause/internal/config"
)

const (
	apiBase = "https://webapi.leigod.com"
	signKey = "5C5A639C20665313622F51E93E3F2783"
)

type Client struct {
	http  *http.Client
	store *config.Store
}

type apiResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type accountData struct {
	PauseStatusID int `json:"pause_status_id"`
}

func NewClient(store *config.Store) *Client {
	return &Client{http: &http.Client{Timeout: 12 * time.Second}, store: store}
}

func PasswordMD5(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}

func (c *Client) Pause(ctx context.Context) (string, error) {
	token, err := c.validToken(ctx)
	if err != nil {
		return "", err
	}
	resp, err := c.post(ctx, "/api/user/pause", url.Values{"account_token": {token}, "lang": {"zh_CN"}, "os_type": {"4"}})
	if err != nil {
		return "", err
	}
	if resp.Code == 400006 {
		token, err = c.login(ctx)
		if err != nil {
			return "", err
		}
		resp, err = c.post(ctx, "/api/user/pause", url.Values{"account_token": {token}, "lang": {"zh_CN"}, "os_type": {"4"}})
		if err != nil {
			return "", err
		}
	}
	if resp.Code != 0 {
		return "", apiError(resp)
	}
	if resp.Msg == "" {
		return "暂停成功", nil
	}
	return resp.Msg, nil
}

func (c *Client) AccountStatus(ctx context.Context) (paused bool, message string, err error) {
	token, username, password, credentialErr := c.store.Credentials()
	if credentialErr != nil {
		return false, "", credentialErr
	}
	if token == "" && (username == "" || password == "") {
		return false, "未配置雷神凭据", nil
	}
	if token == "" {
		token, err = c.login(ctx)
		if err != nil {
			return false, "", err
		}
	}
	info, err := c.info(ctx, token)
	if err != nil && strings.Contains(err.Error(), "400006") && username != "" && password != "" {
		token, err = c.login(ctx)
		if err == nil {
			info, err = c.info(ctx, token)
		}
	}
	if err != nil {
		return false, "", err
	}
	if info.PauseStatusID == 1 {
		return true, "时长已暂停", nil
	}
	return false, "时长使用中", nil
}

func (c *Client) validToken(ctx context.Context) (string, error) {
	token, username, password, err := c.store.Credentials()
	if err != nil {
		return "", err
	}
	if token != "" {
		return token, nil
	}
	if username == "" || password == "" {
		return "", errors.New("请先填写账号和密码，或直接填写 account_token")
	}
	return c.login(ctx)
}

func (c *Client) info(ctx context.Context, token string) (accountData, error) {
	resp, err := c.post(ctx, "/api/user/info", url.Values{"account_token": {token}, "lang": {"zh_CN"}, "os_type": {"4"}})
	if err != nil {
		return accountData{}, err
	}
	if resp.Code != 0 {
		return accountData{}, apiError(resp)
	}
	var data accountData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return data, fmt.Errorf("无法解析账号状态: %w", err)
	}
	return data, nil
}

func (c *Client) login(ctx context.Context) (string, error) {
	_, username, passwordMD5, err := c.store.Credentials()
	if err != nil {
		return "", err
	}
	if username == "" || passwordMD5 == "" {
		return "", errors.New("Token 已失效，请填写账号和密码以自动刷新")
	}
	values := url.Values{
		"account_token": {"null"}, "country_code": {"86"}, "lang": {"zh_CN"},
		"mobile_num": {username}, "os_type": {"4"}, "password": {passwordMD5},
		"region_code": {"1"}, "src_channel": {"guanwang"}, "username": {username},
		"ts": {strconv.FormatInt(time.Now().Unix(), 10)},
	}
	values.Set("sign", sign(values))
	resp, err := c.post(ctx, "/api/auth/login/v1", values)
	if err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", apiError(resp)
	}
	var data struct {
		LoginInfo struct {
			AccountToken string `json:"account_token"`
		} `json:"login_info"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", err
	}
	if data.LoginInfo.AccountToken == "" {
		return "", errors.New("登录成功但未返回 Token")
	}
	if err := c.store.SetToken(data.LoginInfo.AccountToken); err != nil {
		return "", err
	}
	return data.LoginInfo.AccountToken, nil
}

func sign(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(values.Get(key)))
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + "&key=" + signKey))
	return hex.EncodeToString(sum[:])
}

func (c *Client) post(ctx context.Context, path string, values url.Values) (apiResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+path, strings.NewReader(values.Encode()))
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.legod.com/")
	req.Header.Set("User-Agent", "LeiGodAutoPause/1.0 Windows")
	response, err := c.http.Do(req)
	if err != nil {
		return apiResponse{}, fmt.Errorf("连接雷神服务失败: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return apiResponse{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return apiResponse{}, fmt.Errorf("雷神服务返回 HTTP %d", response.StatusCode)
	}
	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return result, errors.New("雷神服务返回了无法识别的数据")
	}
	return result, nil
}

func apiError(resp apiResponse) error {
	message := resp.Msg
	if message == "" {
		message = "未知错误"
	}
	return fmt.Errorf("雷神接口错误 %d: %s", resp.Code, message)
}
