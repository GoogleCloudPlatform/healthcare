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

import {get, isUndefined, set} from 'lodash';
import * as moment from 'moment';

import {ISO_DATE} from '../constants';

const IMMUNIZATION_PATHS = Object.freeze({
  reason: 'explanation.reason[0]',
  reasonCode: 'explanation.reason[0].coding[0].code',
  note: 'note[0].text',
  reactionReference: 'reaction[0].detail.reference',
  vaccineCode: 'vaccineCode.text',
  expirationTag: 'meta.tag',
});

const SYSTEMS = Object.freeze({
  SNOMED: 'http://snomed.info/sct',
  FHIR_TAG: 'http://hl7.org/fhir/tag',
});

/**
 * The SNOMED codes for why an immunization was received.
 */
export enum ImmunizationReason {
  Occupational = '429060002',
  Travel = '281657000',
}

/**
 * Custom codes for what a Patient's reaction to an immunization was.
 */
export enum ImmunizationReaction {
  Fever = 'Fever',
  Seizure = 'Seizure',
  Rash = 'Rash',
  Swelling = 'Swelling',
  AnaphalacticShock = 'Anaphalactic shock',
  Other = 'Other',
}

export interface ImmunizationProperties {
  date: moment.Moment;
  name: string;
  expirationDate?: moment.Moment;
  reason?: ImmunizationReason;
  reactionID?: string;
  note?: string;
}

/**
 * A wrapper around a FHIR Immunization that adds functionality and makes
 * nested properties easier to access.
 */
export class Immunization implements ImmunizationProperties {
  expirationDate: moment.Moment;
  date: moment.Moment;
  name: string;
  reason?: ImmunizationReason;
  reactionID?: string;
  note?: string;

  static fromProps(props: ImmunizationProperties): Immunization {
    const imm = new Immunization({
      resourceType: 'Immunization',
      status: 'completed',
      notGiven: false,
      patient: {},
      vaccineCode: {},
      primarySource: true,
    });
    Object.assign(imm, {
      name: props.name,
      date: props.date,
      reason: props.reason,
      note: props.note,
      reactionID: props.reactionID,
    });
    imm.expirationDate =
        props.expirationDate ? props.expirationDate : moment.invalid();
    return imm;
  }

  constructor(private fhirImmunization: fhir.Immunization) {
    this.expirationDate = fhirImmunization.expirationDate ?
        moment(fhirImmunization.expirationDate, ISO_DATE) :
        moment.invalid();
    this.name = get(fhirImmunization, IMMUNIZATION_PATHS.vaccineCode);
    this.date = moment(fhirImmunization.date, moment.ISO_8601);
    this.reason = get(this.fhirImmunization, IMMUNIZATION_PATHS.reasonCode);
    this.reactionID =
        get(this.fhirImmunization, IMMUNIZATION_PATHS.reactionReference);
    this.note = get(this.fhirImmunization, IMMUNIZATION_PATHS.note, '');
  }

  toFHIR(): fhir.Immunization {
    if (isUndefined(this.reactionID)) {
      delete this.fhirImmunization.reaction;
    } else {
      set(this.fhirImmunization, IMMUNIZATION_PATHS.reactionReference,
          this.reactionID);
      this.fhirImmunization.reaction![0].reported = true;
    }

    let reason: fhir.CodeableConcept|undefined = undefined;
    if (this.reason !== undefined) {
      reason = {
        coding: [{
          system: SYSTEMS.SNOMED,
          code: this.reason,
        }]
      };
    }
    set(this.fhirImmunization, IMMUNIZATION_PATHS.reason, reason);
    this.fhirImmunization.expirationDate = this.expirationDate.isValid() ?
        this.expirationDate.format(ISO_DATE) :
        undefined;
    set(this.fhirImmunization, IMMUNIZATION_PATHS.vaccineCode, this.name);
    this.fhirImmunization.date = this.date.toISOString();
    set(this.fhirImmunization, IMMUNIZATION_PATHS.note, this.note);

    return this.fhirImmunization;
  }

  /**
   * Returns true if this Immunization expires in the next 6 months and hasn't
   * been annotated as having a newer Immunization.
   */
  isExpiringSoon(): boolean {
    return this.expirationDate.isValid() &&
        !(get(this.fhirImmunization, IMMUNIZATION_PATHS.expirationTag, []) as
          fhir.Coding[])
             .some(t => t.code === 'updated') &&
        moment(this.expirationDate).subtract(6, 'months') < moment();
  }

  /**
   * Updates a immunization by creating a new dose with the same expiry period
   * and tagging this immunization as no longer expired.
   * @returns New immunization reflecting another administration of this
   *     vaccine.
   */
  createNewDose(): fhir.Immunization {
    this.markUpdated();

    // Assume the same expiration time period
    let expirationDate: string|undefined;
    if (this.expirationDate.isValid()) {
      expirationDate =
          moment().add(this.expirationDate.diff(this.date)).format(ISO_DATE);
    }
    // Copy over any fields that will be the same between the two Immunizations.
    return {
      resourceType: 'Immunization',
      date: new Date().toISOString(),
      expirationDate,
      patient: this.fhirImmunization.patient,
      notGiven: false,
      status: 'completed',
      vaccineCode: this.fhirImmunization.vaccineCode,
      primarySource: true,
    };
  }

  /**
   * Set a tag on the Immunization to indicate that it no longer needs to be
   * renewed because another Immunization superceeds it.
   */
  markUpdated(): void {
    set(this.fhirImmunization, IMMUNIZATION_PATHS.expirationTag, [
      {
        system: SYSTEMS.FHIR_TAG,
        code: 'updated',
      },
    ]);
  }
}
