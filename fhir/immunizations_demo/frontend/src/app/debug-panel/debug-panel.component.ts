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

import {Component, Inject, OnInit} from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';
import {from, Observable, of} from 'rxjs';
import {map, scan, switchAll} from 'rxjs/operators';

import {FHIR_STORE, FHIRStore} from '../fhir-store';
import {FHIRRequest, ResourceService} from '../resource.service';
import {encodeURL} from '../util';
import {ClipboardService} from './clipboard.service';

/**
 * This component allows the user to view information about FHIR API requests as
 * they occur.
 */
@Component({
  selector: 'app-debug-panel',
  templateUrl: './debug-panel.component.html',
  styleUrls: ['./debug-panel.component.scss'],
})
export class DebugPanelComponent implements OnInit {
  requests: Observable<FHIRRequest[]> = of([]);

  private readonly fhirStoreUrl: string;

  constructor(
      @Inject(FHIR_STORE) fhirStore: FHIRStore,
      private readonly snackbar: MatSnackBar,
      private readonly resourceService: ResourceService,
      private readonly clipboardService: ClipboardService,
  ) {
    this.fhirStoreUrl =
        fhirStore.baseURL +
        encodeURL`projects/${fhirStore.project}/locations/${
            fhirStore.location}/datasets/${fhirStore.dataset}/fhirStores/${
            fhirStore.fhirStore}/`;
  }

  ngOnInit(): void {
    this.requests = this.resourceService.requests$.pipe(
        scan<FHIRRequest, FHIRRequest[]>(
            (acc, req) => {
              this.formatResponseBody(req);
              acc.unshift(req);
              return acc;
            },
            []),
    );
  }

  /** Strips off the FHIR store's base URL from url. */
  stripFHIRStoreURL(url: string): string {
    return url.replace(this.fhirStoreUrl, '<FHIR store>/');
  }

  /**
   * Formats the selected request as a cURL command and copies it to the user's
   * clipboard.
   */
  copyRequestAsCURL(req: FHIRRequest): void {
    let curlCommand = `curl -X ${req.method} "${req.url}`;
    if (req.queryParams) {
      curlCommand += '?';
      const queryParams = req.queryParams;
      curlCommand += Object.keys(queryParams)
                         .map(key => encodeURL`${key}=${queryParams[key]}`)
                         .join('&');
    }
    curlCommand += '"';

    const headers = req.headers || {};
    for (const header of Object.keys(headers)) {
      curlCommand += ` -H "${header}: ${headers[header]}"`;
    }

    if (req.requestBody) {
      curlCommand += ` -d '${JSON.stringify(req.requestBody)}'`;
    }

    try {
      this.clipboardService.writeText(curlCommand);
      this.snackbar.open('Request copied to clipboard.', undefined, {
        duration: 2000,
      });
    } catch (e) {
      console.error(`unable to copy text: ${e}`);
    }
  }

  private formatResponseBody(req: FHIRRequest): void {
    const responseBodyJSON =
        from(req.responseBody)
            .pipe(map(res => JSON.stringify(res, undefined, 2)));
    req.responseBody =
        from([of('Pending'), responseBodyJSON]).pipe(switchAll());
  }
}
