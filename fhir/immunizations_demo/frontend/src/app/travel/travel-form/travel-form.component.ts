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

import {AfterViewInit, Component, ElementRef, EventEmitter, Input, OnChanges, OnDestroy, OnInit, Output, QueryList, ViewChildren} from '@angular/core';
import {AbstractControl, FormBuilder, FormControl, FormGroup, ValidationErrors, ValidatorFn, Validators} from '@angular/forms';
import {MAT_MOMENT_DATE_FORMATS, MomentDateAdapter} from '@angular/material-moment-adapter';
import {LegacyDateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import * as moment from 'moment';
import {Observable, of, Subject} from 'rxjs';
import {map, startWith, takeUntil} from 'rxjs/operators';

import {ISO_DATE} from '../../constants';
import {PatientService} from '../../patient.service';
import {ResourceService} from '../../resource.service';
import {TravelPlan} from '../travel-plan';

interface TravelFormData {
  destination: string;
  departureDate: moment.Moment;
  returnDate: moment.Moment;
}

// Countries that the ML model has been trained on. 20 largest by population.
const VALID_COUNTRIES = [
  'Bangladesh', 'Brazil',   'China',         'Democratic Republic of the Congo',
  'Egypt',      'Ethiopia', 'Germany',       'India',
  'Indonesia',  'Iran',     'Japan',         'Mexico',
  'Nigeria',    'Pakistan', 'Philippines',   'Russia',
  'Thailand',   'Turkey',   'United States', 'Vietnam',
];

/**
 * A component for creating and editing travel forms.
 */
@Component({
  selector: 'app-travel-form',
  templateUrl: './travel-form.component.html',
  styles: [],
  providers: [
    {
      provide: LegacyDateAdapter,
      useClass: MomentDateAdapter,
      deps: [MAT_DATE_LOCALE]
    },
    {provide: MAT_DATE_FORMATS, useValue: MAT_MOMENT_DATE_FORMATS},
  ],
})
export class TravelFormComponent implements OnInit, OnChanges, AfterViewInit,
                                            OnDestroy {
  /**
   * The travel plan to edit. Leave unspecified to create a new plan.
   */
  @Input() travelPlan: TravelPlan|null = null;
  /**
   * The travel plan questionnaire this plan should be linked to.
   */
  @Input() questionnaire!: fhir.Questionnaire;

  /**
   * Emits an event when this travel item is successfully modified.
   */
  @Output() saved = new EventEmitter<void>();
  /**
   * Emits an event when the create/edit is cancelled.
   */
  @Output() cancelled = new EventEmitter<void>();

  filteredCountries: Observable<string[]> = of([]);
  form: FormGroup;
  loading = false;
  destControl: FormControl;

  @ViewChildren(TravelFormComponent, {read: ElementRef})
  private travelForm!: QueryList<ElementRef<HTMLElement>>;
  private destroyed$ = new Subject<void>();

  constructor(
      private readonly resourceService: ResourceService,
      private readonly patientService: PatientService, fb: FormBuilder) {
    this.destControl = fb.control('', [Validators.required, validCountry()]);
    this.form = fb.group({
      destination: this.destControl,
      departureDate: [null, Validators.required],
      returnDate: [null, Validators.required],
    });
    this.setFormDefaults();
  }

  ngOnInit(): void {
    this.filteredCountries = this.destControl.valueChanges.pipe(
        takeUntil(this.destroyed$),
        startWith(''),
        map((value: string|null) => {
          if (!value) {
            return VALID_COUNTRIES;
          }
          const filterValue = value.toLowerCase();
          return VALID_COUNTRIES.filter(
              country => country.toLowerCase().includes(filterValue));
        }),
    );
  }

  ngOnChanges(): void {
    if (!this.travelPlan) {
      this.setFormDefaults();
      return;
    }

    this.form.patchValue({
      ...this.travelPlan,
    });
  }

  ngAfterViewInit(): void {
    this.travelForm.changes
        .pipe(
            takeUntil(this.destroyed$),
            )
        .subscribe((form: QueryList<ElementRef<HTMLElement>>) => {
          if (form.length === 0) {
            return;
          }
          form.first.nativeElement.scrollIntoView();
        });
  }

  ngOnDestroy(): void {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  cancel(): void {
    this.setFormDefaults();
    this.cancelled.next();
  }

  async submit(): Promise<void> {
    await this.runUpdate(async () => {
      const data: TravelFormData = this.form.value;
      if (this.travelPlan) {
        // Update the fields of the travel plan.
        ({
          destination: this.travelPlan.destination,
          departureDate: this.travelPlan.departureDate,
          returnDate: this.travelPlan.returnDate
        } = data);
        await this.resourceService.saveResource(this.travelPlan.toFHIR());
      } else {
        const patient = await this.patientService.getPatient().toPromise();
        await this.resourceService.createResource(
            this.createQuestionnaireResponse(patient, data));
      }
    });
    this.setFormDefaults();
  }

  private setFormDefaults(): void {
    this.form.reset();
    this.form.patchValue({
      destination: '',
      departureDate: moment(),
      returnDate: moment().add(5, 'days'),
    });
  }

  private createQuestionnaireResponse(
      patient: fhir.Patient, data: TravelFormData): fhir.QuestionnaireResponse {
    return {
      author: {reference: `Patient/${patient.id}`},
      item: [
        {answer: [{valueString: data.destination}], linkId: '1'}, {
          item: [
            {
              answer: [{valueDate: data.departureDate.format(ISO_DATE)}],
              linkId: '2.1'
            },
            {
              answer: [{valueDate: data.returnDate.format(ISO_DATE)}],
              linkId: '2.2'
            }
          ],
          linkId: '2'
        }
      ],
      // Linking to the questionnaire allows us to easily query for all
      // QuestionnaireResponses that are travel plans.
      questionnaire: {reference: `Questionnaire/${this.questionnaire.id}`},
      resourceType: 'QuestionnaireResponse',
      // A required field for QuestionnaireResponses
      status: 'completed',
      subject: {reference: `Patient/${patient.id}`}
    };
  }

  private async runUpdate(update: () => Promise<void>): Promise<void> {
    this.loading = true;

    await update();

    this.loading = false;
    this.saved.next();
  }
}

/**
 * Checks the value of the control to determine if it's a valid country.
 */
function validCountry(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors|null => {
    if (VALID_COUNTRIES.indexOf(control.value) === -1) {
      return {
        country: control.value,
      };
    }
    return null;
  };
}
