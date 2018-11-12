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
import {TestBed} from '@angular/core/testing';

import {expectAsyncToThrowError} from '../test/util';

import {GAPI_CLIENT} from './auth/gapi';
import {FHIR_STORE, FHIRStore} from './fhir-store';
import {ResourceService} from './resource.service';

describe('ResourceService', () => {
  const FHIR_CONTENT_TYPE = 'application/fhir+json;charset=utf-8';
  const FHIR_STORE_URL =
      'base/projects/p/locations/l/datasets/d/fhirStores/fs/';
  const FHIR_HEADERS = Object.freeze({
    'Content-Type': FHIR_CONTENT_TYPE,
  });

  const clientSpy = jasmine.createSpy('request');
  let service: ResourceService;
  const fhirEndpointStub: FHIRStore = {
    baseURL: 'base/',
    project: 'p',
    location: 'l',
    dataset: 'd',
    fhirStore: 'fs',
  };
  const testResource: fhir.Resource = Object.freeze({
    resourceType: 'Patient',
    id: '1',
  });

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        ResourceService, {provide: FHIR_STORE, useValue: fhirEndpointStub},
        {provide: GAPI_CLIENT, useValue: clientSpy}
      ]
    });
  });

  beforeEach(() => {
    service = TestBed.get(ResourceService);
    clientSpy.calls.reset();
  });

  it('should search for resources', async () => {
    const bundle: fhir.Bundle = {} as any;
    clientSpy.and.returnValue(buildClientResponse(bundle));

    const searchResult = await service.searchResource('Patient', {_id: '1'});

    expect(clientSpy).toHaveBeenCalledWith({
      path: `${FHIR_STORE_URL}resources/Patient/`,
      headers: FHIR_HEADERS,
      params: {
        '_id': '1',
      },
    });
    expect(searchResult).toBe(bundle);
  });

  it('should get a resource', async () => {
    clientSpy.and.returnValue(buildClientResponse(testResource));

    const getResult = await service.getResource('Patient', '1');

    expect(clientSpy).toHaveBeenCalledWith({
      path: `${FHIR_STORE_URL}resources/Patient/1`,
      headers: FHIR_HEADERS,
    });
    expect(getResult).toBe(testResource);
  });

  it('should create a resource', async () => {
    const createdResource: fhir.Resource = {} as any;
    clientSpy.and.returnValue(buildClientResponse(createdResource));

    const createResult = await service.createResource(testResource);

    expect(clientSpy).toHaveBeenCalledWith({
      path: `${FHIR_STORE_URL}resources/Patient`,
      headers: FHIR_HEADERS,
      method: 'POST',
      body: testResource,
    });
    expect(createResult).toBe(createdResource);
  });

  it('should throw an error if resouce type is missing', async () => {
    const createdResource: fhir.Resource = {} as any;
    clientSpy.and.returnValue(buildClientResponse(createdResource));

    await expectAsyncToThrowError(
        () => service.createResource({
          ...testResource,
          resourceType: undefined,
        }),
        /resource type/);

    expect(clientSpy).not.toHaveBeenCalled();
  });

  it('should save a resource', async () => {
    const savedResource: fhir.Resource = {} as any;
    clientSpy.and.returnValue(buildClientResponse(savedResource));

    const saveResult = await service.saveResource(testResource);

    expect(clientSpy).toHaveBeenCalledWith({
      path: `${FHIR_STORE_URL}resources/Patient/1`,
      headers: FHIR_HEADERS,
      method: 'PUT',
      body: testResource,
    });
    expect(saveResult).toBe(savedResource);
  });

  it('should delete a resource', async () => {
    clientSpy.and.returnValue(buildClientResponse(undefined));

    await service.deleteResource(testResource);

    expect(clientSpy).toHaveBeenCalledWith({
      path: `${FHIR_STORE_URL}resources/Patient/1`,
      headers: FHIR_HEADERS,
      method: 'DELETE',
    });
  });

  it('should delete a resource by identifier', async () => {
    clientSpy.and.returnValue(buildClientResponse(undefined));

    await service.deleteResource(testResource.resourceType!, testResource.id!);

    expect(clientSpy).toHaveBeenCalledWith({
      path: `${FHIR_STORE_URL}resources/Patient/1`,
      headers: FHIR_HEADERS,
      method: 'DELETE',
    });
  });

  it('should throw an error if resouce type is missing', async () => {
    clientSpy.and.returnValue(buildClientResponse(undefined));

    await expectAsyncToThrowError(
        () => service.deleteResource({
          ...testResource,
          id: undefined,
        }),
        /resource ID/);
    expect(clientSpy).not.toHaveBeenCalled();
  });

  it('should execute a transaction', async () => {
    const respBundle: fhir.Bundle = {} as any;
    const reqBundle: fhir.Bundle = {type: 'transaction'} as any;
    clientSpy.and.returnValue(buildClientResponse(respBundle));

    const resp = await service.executeBatch(reqBundle);

    expect(clientSpy).toHaveBeenCalledWith({
      path: FHIR_STORE_URL,
      headers: FHIR_HEADERS,
      method: 'POST',
      body: reqBundle,
    });
    expect(resp).toBe(respBundle);
  });

  function buildClientResponse<T>(data: T): gapi.client.HttpRequest<T> {
    return Promise.resolve({
      result: data,
    }) as any;
  }
});
