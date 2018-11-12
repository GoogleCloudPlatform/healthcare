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
import {BehaviorSubject, Observable} from 'rxjs';

import {Immunization} from '../app/immunizations/immunization';
import {ImmunizationService} from '../app/immunizations/immunization.service';
import {createTestImmunization} from './immunization';
import {createResourceServiceSpy} from './resource-service-spy';

export class ImmunizationServiceStub extends ImmunizationService {
  readonly immunizationSubject = new BehaviorSubject<Immunization[]>([]);
  readonly immunizations$: Observable<Immunization[]>;

  constructor() {
    super(createResourceServiceSpy());
    this.immunizations$ = this.immunizationSubject.asObservable();
  }

  create(): Promise<Immunization> {
    return Promise.resolve(new Immunization(createTestImmunization()));
  }

  save(imm: Immunization): Promise<Immunization> {
    return Promise.resolve(imm);
  }

  remove(_imm: Immunization): Promise<void> {
    return Promise.resolve();
  }

  addNewDose(_imm: Immunization): Promise<void> {
    return Promise.resolve();
  }
}
