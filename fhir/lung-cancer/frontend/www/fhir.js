/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
const PROJECT_ID = 'TODO-project';
const DATASET_ID = 'TODO-dataset';
const FHIR_STORE_ID = 'TODO-fhir-store-id';
const FHIR_STORE_URL = `https://healthcare.googleapis.com/v1beta1/projects/${
    PROJECT_ID}/locations/us-central1/datasets/${DATASET_ID}/fhirStores/${
    FHIR_STORE_ID}/fhir`;

let patientId;
let weightObsId;
let smokingObsId;

const gauge = new RiskGauge();

/** Gets the ID from a FHIR path. */
function parseIdFromName(loc) {
  const parts = loc.split('/');
  return parts[parts.length - 1];
}

/**
 * Creates the Patient resource.
 *
 * @param {number} age the patient's age in years.
 * @param {string} [id] the patient's ID if this is an update.
 */
function createPatient(age, id) {
  const now = new Date();
  const patient = {
    fullUrl: 'urn:uuid:1',
    request: {
      url: 'Patient',
      method: 'POST',
    },
    resource: {
      resourceType: 'Patient',
      birthDate: now.toISOString()
                     .replace(now.getFullYear(), now.getFullYear() - age)
                     .replace(/T.*/, ''),
    },
  };
  if (id !== undefined) {
    patient.request = {
      url: `Patient/${id}`,
      method: 'PUT',
    };
    patient.resource.id = id;
  }
  return patient;
}

/**
 * Creates the weight Observation resource.
 *
 * @param {number} weight the patient's weight in kg.
 * @param {string} [obsId] the observation's ID if this is an update.
 * @param {string} [patientId] the patient's ID if this is an update.
 */
function createWeightObservation(weight, obsId, patientId) {
  const now = new Date();
  const obs = {
    fullUrl: 'urn:uuid:2',
    request: {
      url: 'Observation',
      method: 'POST',
    },
    resource: {
      resourceType: 'Observation',
      category: [{
        coding: [{
          system: 'http://hl7.org/fhir/observation-category',
          code: 'vital-signs',
          display: 'vital-signs',
        }],
      }],
      code: {
        coding: [{
          system: 'http://loinc.org',
          code: '29463-7',
          display: 'Body Weight',
        }],
        text: 'Body Weight',
      },
      subject: {
        reference: 'urn:uuid:1',
      },
      effectiveDateTime: now.toISOString(),
      issued: now.toISOString(),
      valueQuantity: {
        value: weight,
        unit: 'kg',
        system: 'http://unitsofmeasure.org',
        code: 'kg',
      },
    },
  };
  if (obsId !== undefined) {
    obs.request = {
      url: `Observation/${obsId}`,
      method: 'PUT',
    };
    obs.resource.id = obsId;
    obs.resource.subject.reference = `Patient/${patientId}`;
  }
  return obs;
}

/**
 * Creates the smoking Observation resource.
 *
 * @param {boolean} smoker true if the patient is a current/former smoker.
 * @param {string} [obsId] the observation's ID if this is an update.
 * @param {string} [patientId] the patient's ID if this is an update.
 */
function createSmokingObservation(smoker, obsId, patientId) {
  const now = new Date();

  const smokingValue = smoker ? {
    coding: [{
      system: 'http://snomed.info/sct',
      code: '8517006',
      display: 'Former smoker',
    }],
    text: 'Former smoker',
  } :
                                {
                                  coding: [{
                                    system: 'http://snomed.info/sct',
                                    code: '266919005',
                                    display: 'Never smoker',
                                  }],
                                  text: 'Never smoker',
                                };

  const obs = {
    fullUrl: 'urn:uuid:3',
    request: {
      url: 'Observation',
      method: 'POST',
    },
    resource: {
      resourceType: 'Observation',
      category: [{
        coding: [{
          system: 'http://hl7.org/fhir/observation-category',
          code: 'survey',
          display: 'survey',
        }],
      }],
      code: {
        coding: [{
          system: 'http://loinc.org',
          code: '72166-2',
          display: 'Tobacco smoking status NHIS',
        }],
        text: 'Tobacco smoking status NHIS',
      },
      subject: {
        reference: 'urn:uuid:1',
      },
      effectiveDateTime: now.toISOString(),
      issued: now.toISOString(),
      valueCodeableConcept: smokingValue,
    },
  };
  if (obsId !== undefined) {
    obs.request = {
      url: `Observation/${obsId}`,
      method: 'PUT',
    };
    obs.resource.id = obsId;
    obs.resource.subject.reference = `Patient/${patientId}`;
  }
  return obs;
}

/**
 * Poll the FHIR store for new RiskAssessments for `patient`.
 *
 * @param {string} patientId the current patient.
 * @param {number} lastUpdated the last time the RiskAssessments were updated.
 */
function getRiskAssessments(patientId, lastUpdated) {
  gapi.client
      .request({
        path: FHIR_STORE_URL + '/RiskAssessment',
        params: {
          subject: `Patient/${patientId}`,
          _lastUpdated: 'gt' + new Date(lastUpdated).toISOString(),
        },
      })
      .then(resp => {
        const data = JSON.parse(resp.body);
        if (!data.entry) {
          setTimeout(() => {
            getRiskAssessments(patientId, lastUpdated);
          }, 1000);
          return;
        }
        const riskAssessment = data.entry[0].resource;
        const risk =
            riskAssessment.prediction[0].qualitativeRisk.coding[0].code;
        const severity = {
          'negligible': 1,
          'low': 2,
          'moderate': 3,
          'high': 4,
        }[risk];

        document.querySelector('#progress_bar').classList.add('hidden');
        document.querySelector('#risk_card').classList.remove('hidden');
        gauge.setSeverity(severity);
      }, err => console.error(err));
}

/**
 * Create the patient bundle out of the form data.
 */
function createBundle() {
  const pForm = document.querySelector('#patient_form');

  const bundle = {
    resourceType: 'Bundle',
    type: 'transaction',
    entry: [
      createPatient(Number(pForm.age.value), patientId),
      createWeightObservation(
          Number(pForm.weight.value), weightObsId, patientId),
      createSmokingObservation(pForm.smoker.checked, smokingObsId, patientId),
    ],
  };

  const progressBar = document.querySelector('#progress_bar');
  progressBar.classList.remove('hidden');
  gapi.client
      .request({
        path: FHIR_STORE_URL,
        method: 'POST',
        headers: {
          'Content-Type': 'application/fhir+json;charset=utf-8',
        },
        body: bundle,
      })
      .then(
          resp => {
            const data = JSON.parse(resp.body);
            patientId = parseIdFromName(data.entry[0].response.location);
            weightObsId = parseIdFromName(data.entry[1].response.location);
            smokingObsId = parseIdFromName(data.entry[2].response.location);
            updatePatientBtn.classList.remove('disabled');

            getRiskAssessments(patientId, resp.headers['date']);
          },
          () => {
            progressBar.classList.remove('hidden');
          });
}

gapi.load('client:auth2', {
  callback() {
    gapi.auth2
        .init({
          client_id:
              '15408748777-u6qcc2694bmt531qbtp200j9f4e713do.apps.googleusercontent.com',
          ux_mode: 'redirect',
          scope: 'https://www.googleapis.com/auth/cloud-healthcare',
          fetch_basic_profile: false,
          redirect_uri: window.location.origin,
        })
        .then(() => {
          gapi.client.init({});
          const auth = gapi.auth2.getAuthInstance();
          if (!auth.isSignedIn.get()) {
            auth.signIn();
          }
        });
  },
});

document.querySelector('#create-patient-btn').addEventListener('click', e => {
  e.preventDefault();
  createBundle();
});

const updatePatientBtn = document.querySelector('#update-patient-btn');
updatePatientBtn.addEventListener('click', e => {
  e.preventDefault();
  createBundle();
});

document.querySelector('#patient_form').addEventListener('submit', e => {
  e.preventDefault();
  createBundle();
});
