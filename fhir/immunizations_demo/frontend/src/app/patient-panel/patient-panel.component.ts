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
import {NoopScrollStrategy, Overlay, OverlayConfig, OverlayRef} from '@angular/cdk/overlay';
import {TemplatePortal} from '@angular/cdk/portal';
import {Component, ElementRef, EventEmitter, OnDestroy, OnInit, TemplateRef, ViewChild, ViewContainerRef} from '@angular/core';
import {upperFirst} from 'lodash';
import * as moment from 'moment';
import {Subscription} from 'rxjs';
import {takeUntil} from 'rxjs/operators';
import {ISO_DATE} from '../constants';
import {PatientService} from '../patient.service';

/**
 * Data from a Patient resource parsed for display
 */
interface PatientData {
  name?: string;
  phoneNumber?: string;
  age?: number;
  gender?: string;
}

@Component({
  selector: 'app-patient-panel',
  templateUrl: './patient-panel.component.html',
  styles: []
})
export class PatientPanelComponent implements OnInit, OnDestroy {
  patient: PatientData = {};

  @ViewChild('content') private content!: TemplateRef<{}>;
  @ViewChild('trigger', {read: ElementRef})
  private trigger!: ElementRef<HTMLElement>;

  private overlayRef!: OverlayRef;
  private portal!: TemplatePortal;
  private isOpen = false;
  private close$ = new EventEmitter<void>();
  private destroyed$ = new EventEmitter<void>();
  private closeSub: Subscription|undefined;

  constructor(
      private readonly overlay: Overlay,
      private readonly viewContainerRef: ViewContainerRef,
      private readonly patientService: PatientService) {}

  ngOnInit(): void {
    this.patientService.getPatient().subscribe(p => this.setPatientData(p));

    this.close$.pipe(takeUntil(this.destroyed$)).subscribe(() => {
      if (this.overlayRef) {
        this.overlayRef.detach();
      }
      if (this.closeSub) {
        this.closeSub.unsubscribe();
        this.closeSub = undefined;
      }
      this.isOpen = false;
    });
  }

  ngOnDestroy(): void {
    this.close();
    this.destroyed$.next();
  }

  toggle(): void {
    if (this.isOpen) {
      this.close();
    } else {
      this.open();
    }
  }

  private open(): void {
    this.overlayRef = this.createOverlay();
    this.closeSub =
        this.overlayRef.backdropClick().subscribe(() => this.close());
    this.overlayRef.attach(this.portal);
    this.isOpen = true;
  }

  private close(): void {
    this.close$.next();
  }

  private createOverlay(): OverlayRef {
    if (this.overlayRef) {
      return this.overlayRef;
    }
    this.portal = new TemplatePortal(this.content, this.viewContainerRef);
    this.overlayRef = this.overlay.create(new OverlayConfig({
      positionStrategy: this.overlay.position()
                            .flexibleConnectedTo(this.trigger)
                            .withLockedPosition()
                            .setOrigin(this.trigger)
                            .withPositions([{
                              originX: 'end',
                              originY: 'bottom',
                              overlayX: 'end',
                              overlayY: 'top',
                            }])
                            .withDefaultOffsetX(0)
                            .withDefaultOffsetY(0),
      scrollStrategy: new NoopScrollStrategy(),
      hasBackdrop: true,
      backdropClass: 'cdk-overlay-transparent-backdrop',
    }));
    return this.overlayRef;
  }

  private setPatientData(p: fhir.Patient) {
    const birthDate = moment(p.birthDate, ISO_DATE);
    let age: number|undefined;
    if (birthDate.isValid()) {
      age = Math.floor(moment.duration(moment().diff(birthDate)).asYears());
    }

    const nameData = (p.name || [])[0];
    const name = nameData ? `${nameData.family}, ${nameData.given}` : '';

    const teleData = (p.telecom || [])[0];
    const phoneNumber = teleData ? teleData.value : '';

    const gender = upperFirst(p.gender);

    this.patient = {
      age,
      name,
      phoneNumber,
      gender,
    };
  }
}
