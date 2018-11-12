// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import {InjectionToken} from '@angular/core';
import {environment} from 'src/environments/environment';

export const HEALTHCARE_SCOPE =
    'https://www.googleapis.com/auth/cloud-healthcare';

export const GAPI_CLIENT = new InjectionToken('GAPI_CLIENT');

/**
 * Initialize the Google API client, prompting the user to sign in if they
 * haven't already.
 */
export function initGapi(): () => Promise<void> {
  return () => {
    // Wrap in an Angular-aware Promise. The typings for the .then methods also
    // throw errors in Typescript, so we can't await them.
    return new Promise((resolve, reject) => {
      gapi.load('client:auth2', {
        callback() {
          gapi.auth2
              .init({
                client_id: environment.clientId,
                ux_mode: 'redirect',
                scope: HEALTHCARE_SCOPE,
                fetch_basic_profile: false,
                redirect_uri: window.location.origin + '/oauth-callback',
              })
              .then(() => gapi.client.init({}), reject)
              .then(resolve, reject);
        },
        onerror: reject,
      });
    });
  };
}

export interface GoogleApiClient {
  <T>(args: gapi.client.RequestOptions): gapi.client.HttpRequest<T>;
}
