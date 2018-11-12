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
// tslint:disable:no-any
import {inject, TestBed} from '@angular/core/testing';
import {ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';

import {AuthorizationGuard} from './authorization.guard';

describe('AuthorizationGuard', () => {
  const isSignedInSpy =
      jasmine.createSpyObj<gapi.auth2.IsSignedIn>('IsSignedIn', ['get']);
  const currentUserSpy = jasmine.createSpyObj<gapi.auth2.GoogleUser>(
      'GoogleUser', ['hasGrantedScopes']);
  const authSpy: gapi.auth2.GoogleAuth = {
    signIn: () => currentUserSpy,
    currentUser: {
      get() {
        return currentUserSpy;
      }
    },
    isSignedIn: isSignedInSpy,
  } as any;
  const next: ActivatedRouteSnapshot = {} as any;
  const state: RouterStateSnapshot = {} as any;

  beforeEach(() => {
    (window as any).gapi = {
      auth2: {
        getAuthInstance: () => authSpy,
      }
    };
    TestBed.configureTestingModule({providers: [AuthorizationGuard]});
  });

  it('should return true if the user is authenticated',
     inject([AuthorizationGuard], async (guard: AuthorizationGuard) => {
       isSignedInSpy.get.and.returnValue(true);
       currentUserSpy.hasGrantedScopes.and.returnValue(true);

       const authorized = await guard.canActivate(next, state);

       expect(authorized).toBeTruthy();
     }));

  it('should login if the user is not authenticated',
     inject([AuthorizationGuard], async (guard: AuthorizationGuard) => {
       isSignedInSpy.get.and.returnValue(false);
       currentUserSpy.hasGrantedScopes.and.returnValue(true);
       const signInSpy = spyOn(authSpy, 'signIn').and.callThrough();

       await guard.canActivate(next, state);

       expect(signInSpy).toHaveBeenCalled();
     }));

  it('should fail if the user has not granted the healthcare scope',
     inject([AuthorizationGuard], async (guard: AuthorizationGuard) => {
       isSignedInSpy.get.and.returnValue(true);
       currentUserSpy.hasGrantedScopes.and.returnValue(false);

       const authorized = await guard.canActivate(next, state);

       expect(authorized).toBeFalsy();
     }));
});
