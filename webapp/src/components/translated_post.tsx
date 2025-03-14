// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import styled from 'styled-components';
import {useSelector} from 'react-redux';
import {FormattedMessage} from 'react-intl';

import {GlobalState} from '@mattermost/types/store';
import LoadingSpinner from 'src/components/widgets/loading_spinner';
import {UserProfile} from '@mattermost/types/users';

import PostText from './post_text';

const Loading = styled.div`
  opacity: 0.7;
`

interface Props {
    post: any;
}

export const TranslatedPost = (props: Props) => {
    const currentUserId = useSelector<GlobalState, string>((state) => state.entities.users.currentUserId);
    const currentUser = useSelector<GlobalState, UserProfile>((state) => state.entities.users.profiles[currentUserId]);

    let currentUserLocale = 'en'
    if (currentUser) {
      currentUserLocale = currentUser.locale || 'en';
    }

    const userPreferences = useSelector((state: GlobalState) => state.entities.preferences.myPreferences);
    const currentUserTranslationPreference = (userPreferences["pp_mattermost-channel-translatio--translation_language"] || {}).value || 'en'

    const post = props.post;
    let message = post.message
    let loading = false
    if (post.type === "custom_translation") {
        loading = true
    }
    if (post.props?.translations && post.props?.translations[currentUserTranslationPreference || currentUserLocale || '']) {
        loading = false
        message = post.props?.translations[currentUserTranslationPreference || currentUserLocale]
    }

    return (
        <>
            {loading && <Loading><LoadingSpinner/><FormattedMessage defaultMessage="Translating"/></Loading>}
            {!loading && <PostText
                message={message}
                channelID={props.post.channel_id}
                postID={props.post.id}
            />}
        </>
    );
};
