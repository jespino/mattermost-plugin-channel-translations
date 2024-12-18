package main

import (
	"encoding/json"
	"net/http"
	"slices"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/mattermost/mattermost-plugin-ai/server/ai"
	"github.com/mattermost/mattermost-plugin-ai/server/enterprise"
	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) channelAuthorizationRequired(c *gin.Context) {
	channelID := c.Param("channelid")
	userID := c.GetHeader("Mattermost-User-Id")

	channel, err := p.pluginAPI.Channel.Get(channelID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Set(ContextChannelKey, channel)

	if !p.pluginAPI.User.HasPermissionToChannel(userID, channel.Id, model.PermissionReadChannel) {
		c.AbortWithError(http.StatusForbidden, errors.New("user doesn't have permission to read channel"))
		return
	}

	bot := c.MustGet(ContextBotKey).(*Bot)
	if err := p.checkUsageRestrictions(userID, bot, channel); err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}
}

func (p *Plugin) handleToggleTranslations(c *gin.Context) {
	channelID := c.Param("channelid")

	var data struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err := p.setChannelTranslationEnabled(channelID, data.Enabled); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Return the new status in the response
	c.JSON(http.StatusOK, map[string]bool{
		"enabled": data.Enabled,
	})
}

func (p *Plugin) handleGetTranslationStatus(c *gin.Context) {
	channelID := c.Param("channelid")
	
	enabled, err := p.isChannelTranslationEnabled(channelID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, map[string]bool{
		"enabled": enabled,
	})
}

func (p *Plugin) handleSince(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	channel := c.MustGet(ContextChannelKey).(*model.Channel)
	bot := c.MustGet(ContextBotKey).(*Bot)

	if !p.licenseChecker.IsBasicsLicensed() {
		c.AbortWithError(http.StatusForbidden, enterprise.ErrNotLicensed)
		return
	}

	data := struct {
		Since        int64  `json:"since"`
		PresetPrompt string `json:"preset_prompt"`
		Prompt       string `json:"prompt"`
	}{}
	err := json.NewDecoder(c.Request.Body).Decode(&data)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	defer c.Request.Body.Close()

	user, err := p.pluginAPI.User.Get(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	posts, err := p.pluginAPI.Post.GetPostsSince(channel.Id, data.Since)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	threadData, err := p.getMetadataForPosts(posts)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Remove deleted posts
	threadData.Posts = slices.DeleteFunc(threadData.Posts, func(post *model.Post) bool {
		return post.DeleteAt != 0
	})

	formattedThread := formatThread(threadData)

	context := p.MakeConversationContext(bot, user, channel, nil)
	context.PromptParameters = map[string]string{
		"Posts": formattedThread,
	}

	promptPreset := ""
	switch data.PresetPrompt {
	case "summarize":
		promptPreset = ai.PromptSummarizeChannelSince
	case "action_items":
		promptPreset = ai.PromptFindActionItemsSince
	case "open_questions":
		promptPreset = ai.PromptFindOpenQuestionsSince
	}

	if promptPreset == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("invalid preset prompt"))
		return
	}

	p.track(evUnreadMessages, map[string]any{
		"channel_id":     channel.Id,
		"user_actual_id": user.Id,
		"since":          data.Since,
		"type":           promptPreset,
	})

	prompt, err := p.prompts.ChatCompletion(promptPreset, context, p.getDefaultToolsStore(bot, context.IsDMWithBot()))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	resultStream, err := p.getLLM(bot.cfg).ChatCompletion(prompt)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	post := &model.Post{}
	post.AddProp(NoRegen, "true")
	if err := p.streamResultToNewDM(bot.mmBot.UserId, resultStream, user.Id, post); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	promptTitle := ""
	switch data.PresetPrompt {
	case "summarize":
		promptTitle = "Summarize Unreads"
	case "action_items":
		promptTitle = "Find Action Items"
	case "open_questions":
		promptTitle = "Find Open Questions"
	}

	p.saveTitleAsync(post.Id, promptTitle)

	result := struct {
		PostID    string `json:"postid"`
		ChannelID string `json:"channelid"`
	}{
		PostID:    post.Id,
		ChannelID: post.ChannelId,
	}
	c.Render(http.StatusOK, render.JSON{Data: result})
}
