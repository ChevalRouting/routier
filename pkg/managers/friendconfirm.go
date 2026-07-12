package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
)

func FriendReachable(f *config.Friend) bool {
	return friends.Check(context.Background(), f).Reachable
}

func PushApplyConfirm(f *config.Friend, payload PushPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	client := friendClient(f)
	if err := pushImportApply(client, f, data); err != nil {
		return err
	}

	if !FriendReachable(f) {
		return fmt.Errorf("friend %q unreachable after apply (it will roll back)", f.Name)
	}

	return friendConfirm(client, f.URL, f.Token)
}

func friendConfirm(client *http.Client, baseURL, token string) error {
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/apply/confirm", nil)
	if err != nil {
		return fmt.Errorf("create confirm request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("confirm: %w", err)
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("confirm: HTTP %d", resp.StatusCode)
	}

	return nil
}
