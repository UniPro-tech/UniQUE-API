package discord

import (
	"fmt"
	"net/http"

	"github.com/UniPro-tech/UniQUE-API/internal/config"
)

func AddRoleToUser(externalUserID string, roleID string, config *config.Config) error {
	// Discord APIを使用してロールを追加
	botToken := config.DiscordConfig.BotToken
	guildId := config.DiscordConfig.Guild.ID

	if botToken == "" || guildId == "" {
		return fmt.Errorf("discord bot token or guild id not configured")
	}

	discordAPIEndpoint := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", guildId, externalUserID, roleID)
	httpClient := &http.Client{}

	req, err := http.NewRequest("PUT", discordAPIEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+botToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add role, status code: %d", resp.StatusCode)
	}
	return nil
}
