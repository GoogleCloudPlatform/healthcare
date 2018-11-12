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
import {Injectable} from '@angular/core';
import {ActivatedRouteSnapshot, CanActivate, RouterStateSnapshot} from '@angular/router';

import {HEALTHCARE_SCOPE} from './gapi';

/**
 * A routing guard to be used to ensure the user is logged in and has authorized
 * the application to access the Healthcare API. This guard should be present on
 * all routes that require API access.
 */
@Injectable({providedIn: 'root'})
export class AuthorizationGuard implements CanActivate {
  // This value cannot be injected because the GoogleAuth type is lazy loaded.
  // If used in a decorator this would throw an error when the file is
  // evaluated because gapi.auth2 is undefined until runtime.
  private readonly auth = gapi.auth2.getAuthInstance();

  async canActivate(_next: ActivatedRouteSnapshot, _state: RouterStateSnapshot):
      Promise<boolean> {
    let user: gapi.auth2.GoogleUser;
    if (!this.auth.isSignedIn.get()) {
      user = await this.auth.signIn();
    } else {
      user = this.auth.currentUser.get();
    }
    return user.hasGrantedScopes(HEALTHCARE_SCOPE);
  }
}
