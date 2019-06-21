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
import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {get, isUndefined} from 'lodash';

import {ResourceService} from '../../resource.service';
import {Immunization, ImmunizationReaction, ImmunizationReason} from '../immunization';
import {ImmunizationFormComponent} from '../immunization-form/immunization-form.component';
import {ImmunizationService} from '../immunization.service';

const REACTION_OBSERVATION_PATH = 'valueCodeableConcept.coding[0].display';

@Component({
  selector: 'app-immunization-item',
  templateUrl: './immunization-item.component.html',
  styleUrls: ['./immunization-item.component.scss'],
})
export class ImmunizationItemComponent implements OnInit {
  @Input() immunization!: Immunization;
  @Output() updated = new EventEmitter<void>();

  @ViewChild(ImmunizationFormComponent) immForm!: ImmunizationFormComponent;

  // TODO: This would be better managed as a contained resource,
  // but the API doesn't support internal references IDs yet.
  reaction: fhir.Observation|null = null;
  editing = false;
  loading = false;
  header = '';
  expiringSoon = false;

  ImmunizationReason = ImmunizationReason;
  ImmunizationReaction = ImmunizationReaction;

  constructor(
      private readonly resourceService: ResourceService,
      private readonly immunizationService: ImmunizationService,
  ) {}

  ngOnInit(): void {
    this.expiringSoon = this.immunization.isExpiringSoon();
    if (this.expiringSoon) {
      this.header = `Due ${this.immunization.expirationDate.fromNow()}`;
    } else {
      this.header = `Received on ${this.immunization.date.format('ll')}`;
    }
  }

  get reactionDetails(): ImmunizationReaction|undefined {
    return get(this.reaction, REACTION_OBSERVATION_PATH);
  }

  toggleEditing(): void {
    this.editing = !this.editing;
  }

  /**
   * Saves the Immunization, retrieving the newly created reaction Observation
   * if applicable.
   */
  async save(): Promise<void> {
    return this.runUpdate(async () => {
      await this.immForm.submit();
      await this.getReaction();
    });
  }

  /**
   * Deletes this Immunization.
   */
  async delete(): Promise<void> {
    return this.runUpdate(async () => {
      await this.immunizationService.remove(this.immunization);
      if (this.reaction) {
        await this.resourceService.deleteResource(this.reaction);
      }
    });
  }

  /**
   * Marks the current Immunization as updated without creating a new dose,
   * i.e. if the more recent Immunization is already in the system but from
   * another application that didn't set our tag.
   */
  async markUpdated(): Promise<void> {
    return this.runUpdate(async () => {
      this.immunization.markUpdated();
      await this.saveImmunization();
    });
  }

  /**
   * Runs the update Immunzation transaction, which will mark the current
   * Immunization as updated and create a new dose.
   */
  async updateImm(): Promise<void> {
    return this.runUpdate(async () => {
      this.immunizationService.addNewDose(this.immunization);
    });
  }

  /**
   * Retrieves the reaction Observation associated with this Immunization, if it
   * exists.
   */
  async getReaction(): Promise<fhir.Observation|void> {
    if (this.reaction !== null) {
      return this.reaction;
    }

    const reactionUri = this.immunization.reactionID;
    if (isUndefined(reactionUri)) {
      return undefined;
    }

    const [type, id] = reactionUri.split('/');
    this.reaction =
        await this.resourceService.getResource<fhir.Observation>(type, id);
    return this.reaction;
  }

  private async saveImmunization(): Promise<void> {
    await this.immunizationService.save(this.immunization);
    this.editing = false;
  }

  private async runUpdate(update: () => Promise<void>): Promise<void> {
    this.loading = true;

    await update();

    this.loading = false;
    this.updated.next();
  }
}
