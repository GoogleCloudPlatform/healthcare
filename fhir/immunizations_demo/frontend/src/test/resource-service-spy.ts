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

import {Provider} from '@angular/core';
import {isUndefined} from 'lodash';
import {empty} from 'rxjs';

import {ResourceService} from '../app/resource.service';

export function createResourceServiceSpy(): jasmine.SpyObj<ResourceService> {
  const spy = jasmine.createSpyObj<ResourceService>('ResourceService', {
    createResource: undefined,
    deleteResource: Promise.resolve(),
    executeBatch: Promise.resolve(),
    getResource: Promise.resolve(),
    saveResource: Promise.resolve(),
    searchResource: new Promise(() => {}),
  });
  // Allows spyOnProperty to be used
  Object.defineProperty(spy, 'requests$', {
    get() {
      return empty();
    }
  });
  spy.createResource.and.callFake((r: fhir.Resource) => {
    return Promise.resolve(JSON.parse(JSON.stringify(r)));
  });

  return spy;
}

export function resourceServiceSpyProvider(
    spy?: jasmine.SpyObj<ResourceService>): Provider {
  if (isUndefined(spy)) {
    spy = createResourceServiceSpy();
  }
  return {
    provide: ResourceService,
    // Use the same instance of the spy for every injection.
    useFactory: () => spy,
  };
}
