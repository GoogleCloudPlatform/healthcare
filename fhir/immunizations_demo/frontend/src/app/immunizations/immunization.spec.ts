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
import * as moment from 'moment';
import {createTestImmunization} from 'src/test/immunization';
import {ISO_DATE} from '../constants';
import {Immunization, ImmunizationReason} from './immunization';

describe('Immunization', () => {
  let fhirImmunization: fhir.Immunization;
  const VACCINE_CODE = 'Test vaccine code';
  const REASON = ImmunizationReason.Travel;
  const NOTE = 'Immunization note';
  const REACTION_ID = 'reaction';
  const DATE = moment('1970-01-01', ISO_DATE);
  const EXPIRATION_DATE = moment('1970-01-02', ISO_DATE);

  beforeEach(() => {
    fhirImmunization = createTestImmunization(VACCINE_CODE, REASON, NOTE);
    fhirImmunization.reaction = [{
      detail: {
        reference: REACTION_ID,
      },
    }];
  });

  it('should parse the immunization details', () => {
    const imm = new Immunization(fhirImmunization);

    expect(imm.expirationDate.isSame(EXPIRATION_DATE))
        .toBe(true, 'expiration is equal');
    expect(imm.date.isSame(DATE)).toBe(true, 'date is equal');
    expect(imm.name).toEqual(VACCINE_CODE);
    expect(imm.reason).toEqual(REASON);
    expect(imm.reactionID).toEqual(REACTION_ID);
    expect(imm.note).toEqual(NOTE);
  });

  it('should have an invalid expiration if expiration date is undefined',
     () => {
       const imm = new Immunization({
         ...fhirImmunization,
         expirationDate: undefined,
       });

       expect(imm.expirationDate.isValid())
           .toBe(false, 'expiration date is valid');
     });

  it('should have an invalid expiration if missing from parts', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
    });

    expect(imm.expirationDate.isValid())
        .toBe(false, 'expiration date is valid');
  });

  it('should set the reaction on the FHIR resource', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
    });

    expect(imm.reactionID).toBeUndefined();

    imm.reactionID = REACTION_ID;
    const newFhirImmunization = imm.toFHIR();

    expect(newFhirImmunization.reaction).toEqual([{
      detail: {
        reference: REACTION_ID,
      },
      reported: true,
    }]);
  });

  it('should set the note on the FHIR resource', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
    });

    expect(imm.note).toBeUndefined();

    imm.note = NOTE;
    const newFhirImmunization = imm.toFHIR();

    expect(newFhirImmunization.note).toEqual([{
      text: NOTE,
    }]);
  });

  it('should set the reason on the FHIR resource', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
    });

    expect(imm.reason).toBeUndefined();

    imm.reason = REASON;
    const newFhirImmunization = imm.toFHIR();

    expect(newFhirImmunization.explanation).toEqual({
      reason: [{
        coding: [{
          system: 'http://snomed.info/sct',
          code: REASON,
        }],
      }],
    });
  });

  it('should set the vaccineCode on the FHIR resource', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: '',
    });

    expect(imm.name).toBe('');

    imm.name = VACCINE_CODE;
    const newFhirImmunization = imm.toFHIR();

    expect(newFhirImmunization.vaccineCode).toEqual({
      text: VACCINE_CODE,
    });
  });

  it('should set the expiration on the FHIR resource', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
      expirationDate: EXPIRATION_DATE,
    });

    let newFhirImmunization = imm.toFHIR();
    expect(newFhirImmunization.expirationDate)
        .toEqual(EXPIRATION_DATE.format(ISO_DATE));

    imm.expirationDate = moment.invalid();
    newFhirImmunization = imm.toFHIR();

    expect(newFhirImmunization.expirationDate).toBeUndefined();
  });

  it('should identify if the immunization is expiring soon', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
      expirationDate: EXPIRATION_DATE,
    });

    // expiration in the past is expired
    expect(imm.isExpiringSoon()).toBe(true, 'is expiring soon');

    // no expiration date, can't expire
    imm.expirationDate = moment.invalid();
    expect(imm.isExpiringSoon()).toBe(false, 'is expiring soon');

    // expiry is too far in the future
    imm.expirationDate = moment().add(1, 'year');
    expect(imm.isExpiringSoon()).toBe(false, 'is expiring soon');

    // marked as updated, won't show up as expired anymore
    imm.expirationDate = EXPIRATION_DATE;
    imm.markUpdated();
    expect(imm.isExpiringSoon()).toBe(false, 'is expiring soon');
  });

  it('should create a new dose with the same expiration period', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
      expirationDate: EXPIRATION_DATE,
    });

    const newImm = imm.createNewDose();

    expect(imm.toFHIR().meta).toEqual({
      tag: [{
        system: 'http://hl7.org/fhir/tag',
        code: 'updated',
      }],
    });

    const date = moment(newImm.date, moment.ISO_8601).startOf('day');
    expect(date.isSame(moment().startOf('day')));

    const expirationDate = moment(newImm.expirationDate, ISO_DATE);
    expect(expirationDate.isSame(date.add(1, 'day')))
        .toBe(true, 'expiration is 1 day after immunization');

    expect(newImm.vaccineCode.text).toEqual(imm.name);
    expect(newImm.patient).toEqual(imm.toFHIR().patient);
  });

  it('should create a new dose with no expiry', () => {
    const imm = Immunization.fromProps({
      date: DATE,
      name: VACCINE_CODE,
    });

    const newImm = imm.createNewDose();

    expect(newImm.expirationDate).toBeUndefined();
  });
});
