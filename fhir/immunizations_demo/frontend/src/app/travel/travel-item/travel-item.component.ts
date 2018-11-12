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
import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {MAT_MOMENT_DATE_FORMATS, MomentDateAdapter} from '@angular/material-moment-adapter';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {ResourceService} from '../../resource.service';
import {TravelFormComponent} from '../travel-form/travel-form.component';
import {TravelPlan} from '../travel-plan';

@Component({
  selector: 'app-travel-item',
  templateUrl: './travel-item.component.html',
  styleUrls: ['./travel-item.component.scss'],
  providers: [
    {
      provide: DateAdapter,
      useClass: MomentDateAdapter,
      deps: [MAT_DATE_LOCALE]
    },
    {provide: MAT_DATE_FORMATS, useValue: MAT_MOMENT_DATE_FORMATS},
  ],
})
export class TravelItemComponent {
  /**
   * The travel plan to edit/display.
   */
  @Input() travelPlan!: TravelPlan;
  /**
   * The questionnaire that `travelPlan` is in response to.
   */
  @Input() questionnaire!: fhir.Questionnaire;
  /**
   * Emits an event when this travel item is successfully modified.
   */
  @Output() saved = new EventEmitter<void>();

  editing = false;
  loading = false;

  @ViewChild(TravelFormComponent)
  private travelForm: TravelFormComponent|undefined;

  constructor(private readonly resourceService: ResourceService) {}

  cancel(): void {
    this.travelForm!.cancel();
    this.editing = false;
  }

  async save(): Promise<void> {
    return this.runUpdate(async () => {
      await this.travelForm!.submit();
      this.editing = false;
    });
  }

  async delete(): Promise<void> {
    return this.runUpdate(async () => {
      await this.resourceService.deleteResource(this.travelPlan.toFHIR());
    });
  }

  private async runUpdate(update: () => Promise<void>): Promise<void> {
    this.loading = true;

    await update();

    this.loading = false;
    this.saved.next();
  }
}
