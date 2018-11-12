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
import {trigger} from '@angular/animations';
import {AfterViewInit, Component, ElementRef, OnDestroy, QueryList, ViewChildren} from '@angular/core';
import {flow, orderBy, partition} from 'lodash/fp';
import {Observable, Subscription} from 'rxjs';
import * as operators from 'rxjs/operators';

import {OPEN_IN} from '../../animations';
import {Immunization} from '../immunization';
import {ImmunizationFormComponent} from '../immunization-form/immunization-form.component';
import {ImmunizationService} from '../immunization.service';

@Component({
  selector: 'app-immunization-list',
  templateUrl: './immunization-list.component.html',
  styleUrls: [
    './immunization-list.component.scss',
  ],
  animations: [
    trigger('openIn', OPEN_IN),
  ]
})
export class ImmunizationListComponent implements AfterViewInit, OnDestroy {
  readonly dueImmunizations$: Observable<Immunization[]>;
  readonly upToDateImmunizations$: Observable<Immunization[]>;

  newImmunization = false;

  @ViewChildren(ImmunizationFormComponent, {read: ElementRef})
  private immunizationForm!: QueryList<ElementRef<HTMLElement>>;
  private readonly subscriptions = new Subscription();

  constructor(private readonly immunizationService: ImmunizationService) {
    const immunizations$ =
        this.immunizationService.immunizations$.pipe(operators.map(imms => {
          return flow(
              orderBy<Immunization>([i => i.date], ['desc']),
              partition<Immunization>(i => i.isExpiringSoon()))(imms);
        }));
    this.dueImmunizations$ =
        immunizations$.pipe(operators.map(([due, _]) => due));
    this.upToDateImmunizations$ =
        immunizations$.pipe(operators.map(([_, upToDate]) => upToDate));
  }

  ngAfterViewInit(): void {
    this.subscriptions.add(this.immunizationForm.changes.subscribe(
        (form: QueryList<ElementRef<HTMLElement>>) => {
          if (form.length === 0) {
            return;
          }
          form.first.nativeElement.scrollIntoView();
        }));
  }

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
  }

  add(): void {
    this.newImmunization = true;
  }

  addNewImmunization(): void {
    this.newImmunization = false;
  }
}
