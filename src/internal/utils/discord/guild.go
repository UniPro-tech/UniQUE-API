package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/UniPro-tech/UniQUE-API/internal/config"
	"github.com/UniPro-tech/UniQUE-API/internal/query"
	"gorm.io/gorm"
)

func AddToGuild(externalUserID string, db *gorm.DB, config *config.Config) error {
	// ExternalIdentityを取得
	q := query.Use(db)
	externalIdentity, err := q.ExternalIdentity.Where(q.ExternalIdentity.Provider.Eq("discord"), q.ExternalIdentity.ExternalUserID.Eq(externalUserID)).First()
	if err != nil {
		return err
	}

	// Discord APIを使用してギルドに追加
	accessToken := externalIdentity.AccessToken
	botToken := config.DiscordConfig.BotToken
	guildId := config.DiscordConfig.Guild.ID

	discordAPIEndpoint := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", guildId, externalUserID)
	httpClient := &http.Client{}

	// body: { "access_token": "<user access token>" }
	bodyObj := map[string]string{"access_token": accessToken}
	bodyBytes, err := json.Marshal(bodyObj)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", discordAPIEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to add to guild, status code: %d", resp.StatusCode)
	}
	return nil
}
