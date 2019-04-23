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

import {Component, EventEmitter, Input, Output} from '@angular/core';
import {FormBuilder, FormGroup, Validators} from '@angular/forms';
import {MAT_MOMENT_DATE_FORMATS, MomentDateAdapter} from '@angular/material-moment-adapter';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {get, isUndefined, omit, set} from 'lodash';
import * as moment from 'moment';

import {ResourceService} from '../../resource.service';
import {Immunization, ImmunizationReaction, ImmunizationReason} from '../immunization';
import {ImmunizationService} from '../immunization.service';

const REACTION_OBSERVATION_PATH = 'valueCodeableConcept.coding[0].display';

interface ImmunizationFormData {
  date: moment.Moment;
  expirationDate?: moment.Moment;
  name: string;
  reaction?: ImmunizationReaction;
  reason?: ImmunizationReason;
  note?: string;
}

/**
 * A component for creating and editing immunizations.
 */
@Component({
  selector: 'app-immunization-form',
  templateUrl: './immunization-form.component.html',
  exportAs: 'immunizationForm',
  styleUrls: ['./immunization-form.component.scss'],
  providers: [
    {
      provide: DateAdapter,
      useClass: MomentDateAdapter,
      deps: [MAT_DATE_LOCALE]
    },
    {provide: MAT_DATE_FORMATS, useValue: MAT_MOMENT_DATE_FORMATS},
  ],
})
export class ImmunizationFormComponent {
  /**
   * The Immunization to edit, or null to create a new one.
   */
  @Input() immunization: Immunization|null = null;
  /**
   * The reaction Observation to edit, or null to create a new one.
   */
  @Input() reaction: fhir.Observation|null = null;

  /**
   * Emits an event when the Immunization has been saved to the server.
   */
  @Output() saved = new EventEmitter<void>();
  /**
   * Emits an event when the edits in the form are cancelled.
   */
  @Output() cancelled = new EventEmitter<void>();

  ImmunizationReason = ImmunizationReason;
  ImmunizationReaction = ImmunizationReaction;

  form: FormGroup;
  loading = false;

  constructor(
      private readonly resourceService: ResourceService,
      private readonly immunizationService: ImmunizationService,
      fb: FormBuilder,
  ) {
    this.form = fb.group({
      name: [null, Validators.required],
      date: [null, Validators.required],
      expirationDate: [],
      reaction: [],
      reason: [],
      note: [],
    });
    this.resetForm();
  }

  ngOnChanges(): void {
    if (!this.immunization) {
      this.resetForm();
      return;
    }

    this.form.patchValue({
      ...this.immunization,
      reason: this.immunization.reason,
      note: this.immunization.note,
      reaction: get(this.reaction, REACTION_OBSERVATION_PATH),
    });
  }

  /**
   * Cancels creation or discards any edits to the immunization.
   */
  cancel(): void {
    this.resetForm();
    this.cancelled.next();
  }

  /**
   * Saves the immunization.
   */
  async submit(): Promise<void> {
    this.loading = true;
    let imm: Immunization;
    const formData: ImmunizationFormData = this.form.value;
    if (this.immunization === null) {
      imm = await this.immunizationService.create(
          Immunization.fromProps(formData));
    } else {
      imm = this.immunization;
      Object.assign(imm, omit(formData, ['reaction']));
    }
    this.updateReaction(imm, formData.reaction);

    if (!this.reaction) {
      await this.resourceService.deleteResource(
          'Observation',
          `${imm.toFHIR().id}-reaction`,
      );
    } else {
      // Use a save (PUT) operation to ensure the reaction has the ID we
      // assigned.
      await this.resourceService.saveResource(this.reaction);
    }
    // The reaction is guaranteed to exist on the server now, so the link won't
    // fail.
    await this.immunizationService.save(imm);

    this.loading = false;
    this.saved.next();
  }

  /**
   * Resets the form the the default state.
   */
  private resetForm(): void {
    this.form.reset();
    this.form.patchValue({
      name: '',
      date: moment(),
      expirationDate: undefined,
      reaction: undefined,
      reason: undefined,
      note: undefined,
    });
  }

  /**
   * Patches `this.reaction` with the values from the form or deletes it if
   * none is defined.
   * @param immunization The immunization being created or edited.
   * @param value The new reaction or undefined if it should be deleted.
   */
  private updateReaction(
      immunization: Immunization, value: ImmunizationReaction|undefined): void {
    if (isUndefined(value)) {
      immunization.reactionID = undefined;
      this.reaction = null;
      return;
    }

    if (this.reaction === null) {
      this.reaction = this.createReactionObservation(immunization, value);
    } else {
      set(this.reaction, REACTION_OBSERVATION_PATH, value);
    }
    immunization.reactionID = `Observation/${this.reaction.id}`;
  }

  /**
   * Creates a new reaction from form data.
   * @param immunization The immunization this reaction is related to.
   * @param reactionType The type of reaction that occurred.
   * @returns The new reaction (FHIR Observation).
   */
  private createReactionObservation(
      immunization: Immunization,
      reactionType: ImmunizationReaction): fhir.Observation {
    return {
      resourceType: 'Observation',
      // Create an ID that is guaranteed to be unique, since the Immunization ID
      // unique.
      id: `${immunization.toFHIR().id}-reaction`,
      status: 'final',
      code: {
        text: 'reaction',
      },
      effectiveDateTime: immunization.date.toISOString(),
      valueCodeableConcept: {
        coding: [{
          display: reactionType,
        }]
      }
    };
  }
}
