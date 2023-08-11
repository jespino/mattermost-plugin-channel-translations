package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

var ErrUsageRestriction = errors.New("usage restriction")

func (p *Plugin) checkUsageRestrictions(userID string, channel *model.Channel) error {
	if err := p.checkUsageRestrictionsForUser(userID); err != nil {
		return err
	}

	if err := p.checkUsageRestrictionsForChannel(channel); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) checkUsageRestrictionsForChannel(channel *model.Channel) error {
	if channel.Type == model.ChannelTypeDirect {
		return nil
	}

	if p.getConfiguration().EnableUseRestrictions {
		if p.getConfiguration().AllowedTeamIDs != "" && !strings.Contains(p.getConfiguration().AllowedTeamIDs, channel.TeamId) {
			return errors.Wrap(ErrUsageRestriction, "can't work on this team")
		}

		if !p.getConfiguration().AllowPrivateChannels {
			if channel.Type != model.ChannelTypeOpen {
				return errors.Wrap(ErrUsageRestriction, "can't work on private channels")
			}
		}
	}
	return nil
}

func (p *Plugin) checkUsageRestrictionsForUser(userID string) error {
	if p.getConfiguration().EnableUseRestrictions && p.getConfiguration().OnlyUsersOnTeam != "" {
		if !p.pluginAPI.User.HasPermissionToTeam(userID, p.getConfiguration().OnlyUsersOnTeam, model.PermissionViewTeam) {
			return errors.Wrap(ErrUsageRestriction, "user not on allowed team")
		}
	}

	return nil
}
