// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	if !p.getConfiguration().EnableTranslations {
		return post, ""
	}

	// Skip empty messages
	if post.Message == "" {
		return post, ""
	}

	// Skip if already translated
	if _, ok := post.Props["translations"]; ok {
		return post, ""
	}

	// Check if translations are enabled for this channel
	enabled, err := p.isChannelTranslationEnabled(post.ChannelId)
	if err != nil {
		return post, ""
	}
	if !enabled {
		return post, ""
	}

	newPost := post.Clone()
	newPost.Type = "custom_translation"
	return newPost, ""
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	// Skip if global translations are disabled
	if !p.getConfiguration().EnableTranslations {
		return
	}

	// Skip empty messages
	if post.Message == "" {
		return
	}

	// Skip if already translated
	if _, ok := post.Props["translations"]; ok {
		return
	}

	// Check if translations are enabled for this channel
	enabled, err := p.isChannelTranslationEnabled(post.ChannelId)
	if err != nil {
		p.pluginAPI.Log.Debug("failed to check channel translation status", "error", err)
		return
	}
	if !enabled {
		return
	}

	// Get configured languages or use default
	languages := p.getConfiguration().TranslationLanguages
	if languages == "" {
		languages = "english"
	}

	waitGroup := sync.WaitGroup{}
	mutex := sync.Mutex{}
	translations := make(map[string]interface{})
	waitlist := make(chan struct{}, 3)

	for _, language := range strings.Split(languages, ",") {
		waitlist <- struct{}{}
		waitGroup.Add(1)
		go func(lang string) {
			defer waitGroup.Done()
			maxRetry := 10

			for {
				result, err := p.translateText(post.Message, post.UserId, lang)
				if err != nil {
					p.pluginAPI.Log.Warn(fmt.Sprintf("failed to get translations: %w", err))
					maxRetry--
					if maxRetry == 0 {
						break
					}
					continue
				}

				p.pluginAPI.Log.Debug("Extracted translation raw", "translation", result, "language", lang)

				mutex.Lock()
				translations[lang] = result
				// Store translations in post props
				if post.Props == nil {
					post.Props = make(model.StringInterface)
				}
				post.Props["translations"] = translations
				_ = p.pluginAPI.Post.UpdatePost(post)
				mutex.Unlock()
				<-waitlist
				break
			}
		}(language)
	}

	waitGroup.Wait()
	close(waitlist)

	// Store translations in post props
	if post.Props == nil {
		post.Props = make(model.StringInterface)
	}
	post.Props["translations"] = translations

	// Update the post
	if err := p.pluginAPI.Post.UpdatePost(post); err != nil {
		p.pluginAPI.Log.Debug("failed to update post with translations", "error", err)
		return
	}
}
