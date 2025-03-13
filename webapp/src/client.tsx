// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4 as Client4Class, ClientError} from '@mattermost/client';

import manifest from './manifest';

const Client4 = new Client4Class();

function baseRoute(): string {
    return `/plugins/${manifest.id}`;
}

function channelRoute(channelid: string): string {
    return `${baseRoute()}/channel/${channelid}`;
}


export async function getChannelTranslationStatus(channelId: string) {
    const url = `${channelRoute(channelId)}/translations`;
    const response = await fetch(url, Client4.getOptions({
        method: 'GET',
    }));

    if (response.ok) {
        return response.json();
    }

    throw new ClientError(Client4.url, {
        message: '',
        status_code: response.status,
        url,
    });
}

export async function toggleChannelTranslations(channelId: string, enabled: boolean) {
    const url = `${channelRoute(channelId)}/translations`;
    const response = await fetch(url, Client4.getOptions({
        method: 'POST',
        body: JSON.stringify({enabled}),
    }));

    if (response.ok) {
        return;
    }

    throw new ClientError(Client4.url, {
        message: '',
        status_code: response.status,
        url,
    });
}

export async function getTranslationLanguages() {
    const url = `${baseRoute()}/translation/languages`;
    const response = await fetch(url, Client4.getOptions({
        method: 'GET',
    }));

    if (response.ok) {
        return response.json()
    }

    throw new ClientError(Client4.url, {
        message: '',
        status_code: response.status,
        url,
    });
}

export async function setUserTranslationLanguage(language: string) {
    const url = `${baseRoute()}/translation/user_preference`;
    const response = await fetch(url, Client4.getOptions({
        method: 'POST',
        body: JSON.stringify({language}),
    }));

    if (response.ok) {
        return;
    }

    throw new ClientError(Client4.url, {
        message: '',
        status_code: response.status,
        url,
    });
}
