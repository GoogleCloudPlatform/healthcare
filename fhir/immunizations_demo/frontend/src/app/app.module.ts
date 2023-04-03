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


import {HttpClientModule} from '@angular/common/http';
import {APP_INITIALIZER, NgModule} from '@angular/core';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {MatMomentDateModule} from '@angular/material-moment-adapter';
import {MatLegacyAutocompleteModule} from '@angular/material/autocomplete';
import {MatLegacyButtonModule} from '@angular/material/button';
import {MatLegacyCardModule} from '@angular/material/card';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatExpansionModule} from '@angular/material/expansion';
import {MatLegacyFormFieldModule} from '@angular/material/form-field';
import {MatGridListModule} from '@angular/material/grid-list';
import {MatIconModule} from '@angular/material/icon';
import {MatLegacyInputModule} from '@angular/material/input';
import {MatLegacyListModule} from '@angular/material/list';
import {MatLegacyProgressBarModule} from '@angular/material/progress-bar';
import {MatLegacySelectModule} from '@angular/material/select';
import {MatSidenavModule} from '@angular/material/sidenav';
import {MatLegacySnackBarModule} from '@angular/material/snack-bar';
import {MatToolbarModule} from '@angular/material/toolbar';
import {MatTooltipModule} from '@angular/material/tooltip';
import {BrowserModule} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';

import {environment} from '../environments/environment';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {GAPI_CLIENT, initGapi} from './auth/gapi';
import {DebugPanelComponent} from './debug-panel/debug-panel.component';
import {FHIR_STORE} from './fhir-store';
import {ImmunizationFormComponent} from './immunizations/immunization-form/immunization-form.component';
import {ImmunizationItemComponent} from './immunizations/immunization-item/immunization-item.component';
import {ImmunizationListComponent} from './immunizations/immunization-list/immunization-list.component';
import {PatientPanelComponent} from './patient-panel/patient-panel.component';
import {ShortDatePipe} from './short-date.pipe';
import {ConditionListComponent} from './travel/condition-list/condition-list.component';
import {TravelFormComponent} from './travel/travel-form/travel-form.component';
import {TravelItemComponent} from './travel/travel-item/travel-item.component';
import {TravelListComponent} from './travel/travel-list/travel-list.component';
import {LOCATION, SESSION_STORAGE} from './window-injection-tokens';

@NgModule({
  declarations: [
    AppComponent,
    TravelListComponent,
    TravelItemComponent,
    TravelFormComponent,
    ShortDatePipe,
    ConditionListComponent,
    PatientPanelComponent,
    ImmunizationFormComponent,
    ImmunizationListComponent,
    ImmunizationItemComponent,
    DebugPanelComponent,
  ],
  imports: [
    BrowserModule,       BrowserAnimationsModule, MatToolbarModule,
    MatLegacyListModule,       MatLegacyCardModule,           MatExpansionModule,
    MatLegacyFormFieldModule,  MatLegacyInputModule,          MatLegacyButtonModule,
    MatLegacySelectModule,     MatSidenavModule,        MatIconModule,
    MatDatepickerModule, MatLegacyProgressBarModule,    MatTooltipModule,
    MatLegacySnackBarModule,   MatGridListModule,       MatLegacyAutocompleteModule,
    MatMomentDateModule, ReactiveFormsModule,     FormsModule,
    HttpClientModule,    AppRoutingModule,
  ],
  bootstrap: [AppComponent],
  providers: [
    {provide: APP_INITIALIZER, useFactory: initGapi, multi: true},
    // Use providers so that these global services can be easily replaced in
    // tests.
    {provide: FHIR_STORE, useValue: environment.fhirEndpoint},
    {provide: SESSION_STORAGE, useValue: window.sessionStorage},
    {provide: LOCATION, useValue: window.location},
    {provide: GAPI_CLIENT, useFactory: () => gapi.client.request},
  ],
})
export class AppModule {
}
