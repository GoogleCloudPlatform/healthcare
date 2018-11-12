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
import {get} from 'lodash';
import {filter, flow, map} from 'lodash/fp';
import {BehaviorSubject, from, Observable} from 'rxjs';
import * as operators from 'rxjs/operators';

import {ResourceService} from '../resource.service';

import {Immunization} from './immunization';

@Injectable({
  providedIn: 'root',
})
export class ImmunizationService {
  readonly immunizations$: Observable<Immunization[]>;

  private readonly immunizations = new BehaviorSubject<Immunization[]>([]);

  constructor(private readonly resourceService: ResourceService) {
    this.immunizations$ = this.immunizations.asObservable();
    from(this.resourceService.searchResource('Immunization'))
        .pipe(
            operators.map(
                (bundle: fhir.Bundle):
                    Immunization[] => {
                      const entries = bundle.entry || [];
                      // Convert FHIR Immunizations into a domain object and
                      // sort by date received.
                      return flow(
                          filter((e: fhir.BundleEntry) => Boolean(e.resource)),
                          map(e => new Immunization(
                                  e.resource as fhir.Immunization)),
                          )(entries);
                    }),
            )
        .subscribe(imms => {
          this.immunizations.next(imms);
        });
  }

  async create(imm: Immunization): Promise<Immunization> {
    const resource = await this.resourceService.createResource(imm.toFHIR());
    const newImm = new Immunization(resource);
    this.immunizations.next([...this.immunizations.getValue(), newImm]);
    return newImm;
  }

  async save(imm: Immunization): Promise<Immunization> {
    const resource = await this.resourceService.saveResource(imm.toFHIR());
    const newImm = new Immunization(resource);
    this.immunizations.next([...this.immunizations.getValue()]);
    return newImm;
  }

  async remove(imm: Immunization): Promise<void> {
    const imms = this.immunizations.getValue();
    const idx = imms.indexOf(imm);
    if (idx === -1) {
      throw new Error(
          `immunization was not found in the current list of immunizations`);
    }

    const newImms = imms.filter(i => i !== imm);
    await this.resourceService.deleteResource(imm.toFHIR());
    this.immunizations.next(newImms);
  }

  async addNewDose(imm: Immunization): Promise<void> {
    const newImmunization = imm.createNewDose();
    const resource = imm.toFHIR();
    const bundle = {
      resourceType: 'Bundle',
      type: 'transaction',
      entry: [
        {
          request: {
            method: 'PUT',
            url: `Immunization/${resource.id}`,
          },
          resource,
        },
        {
          request: {
            method: 'POST',
            url: 'Immunization',
          },
          resource: newImmunization,
        },
      ],
    };
    const bundleResp = await this.resourceService.executeBatch(bundle);
    const newImmResource = get(bundleResp, 'entry[1].resource');

    const newImms =
        [...this.immunizations.getValue(), new Immunization(newImmResource)];
    this.immunizations.next(newImms);
  }
}
