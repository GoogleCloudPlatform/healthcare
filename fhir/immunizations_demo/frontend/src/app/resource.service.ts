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
import {Inject, Injectable} from '@angular/core';
import {isString, isUndefined} from 'lodash';
import {Observable, Subject} from 'rxjs';

import {GAPI_CLIENT, GoogleApiClient} from './auth/gapi';
import {FHIR_STORE, FHIRStore} from './fhir-store';
import {encodeURL} from './util';

export interface FHIRRequest {
  method: string;
  url: string;
  headers?: {[key: string]: string};
  queryParams?: {[key: string]: string};
  requestBody?: object;
  // tslint:disable-next-line:no-any
  responseBody: Observable<any>|Promise<any>;
}

/**
 * ResourceService is a wrapper layer for FHIR requests. It handles auth and
 * constructing the appropriate FHIR resource urls. The API exposes Promises to
 * allow this class to be used in non-Angular applications with minimal
 * modification. The FHIR store used is configured in environment.ts.
 */
// TODO(b/117507489): Add error handling to API requests.
@Injectable({
  providedIn: 'root',
})
export class ResourceService {
  /** A stream of the FHIR store requests made via this service. */
  requests$: Observable<FHIRRequest>;

  private readonly FHIR_STORE_URL: string;
  private readonly requests = new Subject<FHIRRequest>();

  constructor(
      @Inject(FHIR_STORE) fhirStore: FHIRStore,
      @Inject(GAPI_CLIENT) private readonly client: GoogleApiClient) {
    this.requests$ = this.requests.asObservable();

    this.FHIR_STORE_URL =
        fhirStore.baseURL +
        encodeURL`projects/${fhirStore.project}/locations/${
            fhirStore.location}/datasets/${fhirStore.dataset}/fhirStores/${
            fhirStore.fhirStore}/`;
  }

  /**
   * Performs a search on resources of type `resourceType` against the FHIR
   * store.
   * @param resourceType The resource type to search upon.
   * @param queryParams Search parameters.
   * @returns The search result bundle.
   */
  async searchResource(
      resourceType: string,
      queryParams?: {[key: string]: string}): Promise<fhir.Bundle> {
    const resourceUrl =
        this.FHIR_STORE_URL + encodeURL`resources/${resourceType}/`;
    return this.performRequest<fhir.Bundle>({
      path: resourceUrl,
      headers: getFHIRHeaders(),
      params: queryParams,
    });
  }

  /**
   * Retrieves a resource from the FHIR store.
   * @param resourceType The type of the resource to be retrieved.
   * @param resourceId The ID of the resource to fetch.
   */
  async getResource<R extends fhir.Resource>(
      resourceType: string, resourceId: string): Promise<R> {
    const resourceUrl = this.FHIR_STORE_URL +
        encodeURL`resources/${resourceType}/${resourceId}`;
    return this.performRequest<R>({
      path: resourceUrl,
      headers: getFHIRHeaders(),
    });
  }

  /**
   * Creates a resource in the FHIR store with a server-assigned ID.
   * @param resource The resource data.
   * @returns The newly created resource from the server.
   */
  async createResource<R extends fhir.Resource>(resource: R): Promise<R> {
    const resourceType = getResourceType(resource);
    const resourceUrl =
        this.FHIR_STORE_URL + encodeURL`resources/${resourceType}`;
    return this.performRequest<R>({
      path: resourceUrl,
      method: 'POST',
      headers: getFHIRHeaders(),
      body: resource,
    });
  }

  /**
   * Saves a resource in the FHIR store, potentially creating it if supported by
   * the FHIR store.
   * @param resource The updated data for the resource.
   * @returns The server's new copy of the resource.
   */
  async saveResource<R extends fhir.Resource>(resource: R): Promise<R> {
    const resourceType = getResourceType(resource);
    const resourceID = getResourceID(resource);
    const resourceUrl = this.FHIR_STORE_URL +
        encodeURL`resources/${resourceType}/${resourceID}`;
    return this.performRequest<R>({
      path: resourceUrl,
      method: 'PUT',
      headers: getFHIRHeaders(),
      body: resource,
    });
  }

  /**
   * Marks the resource as deleted in the FHIR store.
   * @param resourceType The type of the resource to be deleted.
   * @param resourceID The ID of the resource to be deleted.
   */
  async deleteResource<R extends fhir.Resource>(
      resourceType: string, resourceID: string): Promise<void>;

  /**
   * Marks the resource as deleted in the FHIR store.
   * @param resource The resource to be deleted.
   */
  async deleteResource<R extends fhir.Resource>(resource: R): Promise<void>;

  async deleteResource<R extends fhir.Resource>(
      resource: R|string, id?: string): Promise<void> {
    let resourceType: string;
    let resourceID: string;
    if (isString(resource)) {
      resourceType = resource;
      resourceID = id!;
    } else {
      resourceType = getResourceType(resource);
      resourceID = getResourceID(resource);
    }
    this.performRequest({
      path: this.FHIR_STORE_URL +
          encodeURL`resources/${resourceType}/${resourceID}`,
      method: 'DELETE',
      headers: getFHIRHeaders(),
    });
  }

  /**
   * Executes a transaction contained in `bundle` in the FHIR store.
   * @param bundle The bundle to be executed.
   * @returns The transaction result.
   */
  async executeBatch(bundle: fhir.Bundle): Promise<fhir.Bundle> {
    return this.performRequest<fhir.Bundle>({
      path: this.FHIR_STORE_URL,
      method: 'POST',
      headers: getFHIRHeaders(),
      body: bundle,
    });
  }

  private performRequest<T>(options: gapi.client.RequestOptions): Promise<T> {
    const result = this.client<T>(options).then(resp => resp.result);
    this.requests.next({
      method: options.method || 'GET',
      url: options.path,
      headers: options.headers,
      queryParams: options.params,
      requestBody: options.body,
      responseBody: result,
    });
    return result;
  }
}

/**
 * Returns the headers necessary for a FHIR request. This is a function and not
 * a static property because gapi.client.request modifies the header object is
 * passed, so headers could be inadvertently shared between requests.
 */
function getFHIRHeaders(): {[key: string]: string} {
  // This is the only content type officially supported by Google FHIR
  // stores (other than for PATCH requests).
  return {
    'Content-Type': 'application/fhir+json;charset=utf-8',
  };
}

function getResourceID<R extends fhir.Resource>(r: R): string {
  const id = r.id;
  if (isUndefined(id)) {
    throw new Error('resource ID not specified');
  }
  return id;
}

function getResourceType<R extends fhir.Resource>(r: R): string {
  const resourceType = r.resourceType;
  if (isUndefined(resourceType)) {
    throw new Error('resource type not specified');
  }
  return resourceType;
}
