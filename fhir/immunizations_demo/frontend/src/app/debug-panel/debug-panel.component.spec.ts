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

import {NO_ERRORS_SCHEMA} from '@angular/core';
import {async, ComponentFixture, fakeAsync, TestBed, tick} from '@angular/core/testing';
import {MatSnackBar} from '@angular/material/snack-bar';
import {Subject} from 'rxjs';

import {environment} from '../../environments/environment';
import {resourceServiceSpyProvider} from '../../test/resource-service-spy';
import {resetSpyObj} from '../../test/util';
import {FHIR_STORE} from '../fhir-store';
import {FHIRRequest, ResourceService} from '../resource.service';

import {ClipboardService} from './clipboard.service';
import {DebugPanelComponent} from './debug-panel.component';

describe('DebugPanelComponent', () => {
  let component: DebugPanelComponent;
  let fixture: ComponentFixture<DebugPanelComponent>;
  const snackBarSpy =
      jasmine.createSpyObj<MatSnackBar>('MatSnackBar', ['open']);
  const FHIR_STORE_URL = environment.fhirEndpoint.baseURL +
      `projects/${environment.fhirEndpoint.project}/locations/${
                             environment.fhirEndpoint.location}/datasets/${
                             environment.fhirEndpoint.dataset}/fhirStores/${
                             environment.fhirEndpoint.fhirStore}/`;

  beforeEach(async(() => {
    TestBed
        .configureTestingModule({
          schemas: [NO_ERRORS_SCHEMA],
          declarations: [DebugPanelComponent],
          providers: [
            {provide: FHIR_STORE, useValue: environment.fhirEndpoint},
            {provide: MatSnackBar, useValue: snackBarSpy},
            {
              provide: ClipboardService,
              useValue: jasmine.createSpyObj('ClipboardService', ['writeText'])
            },
            resourceServiceSpyProvider(),
          ],
        })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(DebugPanelComponent);
    component = fixture.componentInstance;

    resetSpyObj(snackBarSpy);
  });

  it('should add new requests to the list', fakeAsync(() => {
       const request1: FHIRRequest = {
         method: 'GET',
         url: 'a',
         responseBody: Promise.resolve({a: 1}),
       };
       const request2: FHIRRequest = {
         method: 'POST',
         url: 'b',
         responseBody: Promise.resolve({b: 2}),
       };

       const resourceServiceSpy = TestBed.get(ResourceService);
       const requestSource = new Subject();
       spyOnProperty(resourceServiceSpy, 'requests$')
           .and.returnValue(requestSource);
       component.ngOnInit();

       let receivedRequests: FHIRRequest[] = [];
       component.requests.subscribe(requests => {
         receivedRequests = requests;
       });
       requestSource.next(request1);
       requestSource.next(request2);
       tick(1);

       expect(receivedRequests).toEqual([request2, request1]);
     }));

  it('should shorten the URL of requests for display', () => {
    expect(component.stripFHIRStoreURL(FHIR_STORE_URL + 'a'))
        .toEqual('<FHIR store>/a');
  });

  it('should format a FHIR request as a cURL command', () => {
    const request: FHIRRequest = {
      method: 'POST',
      url: 'a',
      headers: {
        Authorization: 'Bearer token',
      },
      queryParams: {b: 'query param', c: 'param2'},
      requestBody: {
        resourceType: 'Patient',
      },
      responseBody: Promise.resolve({a: 1}),
    };
    const clipboardService =
        TestBed.get(ClipboardService) as jasmine.SpyObj<ClipboardService>;

    component.copyRequestAsCURL(request);

    expect(clipboardService.writeText)
        .toHaveBeenCalledWith(
            `curl -X POST "a?b=query%20param&c=param2" -H "Authorization: Bearer token" -d '{"resourceType":"Patient"}'`);
    expect(snackBarSpy.open).toHaveBeenCalled();
  });

  it('should handle the error if text cannot be copied', () => {
    const request: FHIRRequest = {
      method: 'POST',
      url: 'a',
      responseBody: Promise.resolve({a: 1}),
    };
    const clipboardService =
        TestBed.get(ClipboardService) as jasmine.SpyObj<ClipboardService>;
    clipboardService.writeText.and.throwError('copy failed');

    component.copyRequestAsCURL(request);

    expect(snackBarSpy.open).not.toHaveBeenCalled();
  });
});
