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
import {ImmunizationReaction, ImmunizationReason} from '../app/immunizations/immunization';

export function createTestImmunization(
    vaccineCode = 'Test immunization',
    reason: ImmunizationReason = ImmunizationReason.Occupational,
    note = ''): fhir.Immunization {
  return {
    resourceType: 'Immunization',
    status: 'completed',
    notGiven: false,
    date: '1970-01-01',
    expirationDate: '1970-01-02',
    vaccineCode: {text: vaccineCode},
    patient: {reference: 'Patient/test'},
    primarySource: true,
    explanation: {
      reason: [{
        coding: [{
          code: reason,
        }],
      }],
    },
    note: [{text: note}],
  };
}

export function createTestReaction(
    id = '0-reaction',
    reaction: ImmunizationReaction =
        ImmunizationReaction.AnaphalacticShock): fhir.Observation {
  return {
    resourceType: 'Observation',
    id,
    status: 'final',
    code: {
      text: 'test',
    },
    valueCodeableConcept: {
      coding: [{display: reaction}],
    },
  };
}
