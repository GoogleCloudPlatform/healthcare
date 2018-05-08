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

package com.google.cloud.healthcare.transform;

import ca.uhn.hl7v2.HL7Exception;
import ca.uhn.hl7v2.model.Message;
import ca.uhn.hl7v2.model.Varies;
import ca.uhn.hl7v2.model.v231.datatype.CX;
import ca.uhn.hl7v2.model.v231.datatype.ST;
import ca.uhn.hl7v2.model.v231.datatype.XAD;
import ca.uhn.hl7v2.model.v231.datatype.XCN;
import ca.uhn.hl7v2.model.v231.datatype.XPN;
import ca.uhn.hl7v2.model.v231.group.ORU_R01_OBXNTE;
import ca.uhn.hl7v2.model.v231.group.ORU_R01_ORCOBRNTEOBXNTECTI;
import ca.uhn.hl7v2.model.v231.group.ORU_R01_PIDPD1NK1NTEPV1PV2;
import ca.uhn.hl7v2.model.v231.group.ORU_R01_PIDPD1NK1NTEPV1PV2ORCOBRNTEOBXNTECTI;
import ca.uhn.hl7v2.model.v231.message.ADT_A01;
import ca.uhn.hl7v2.model.v231.message.ORU_R01;
import ca.uhn.hl7v2.model.v231.segment.OBR;
import ca.uhn.hl7v2.model.v231.segment.OBX;
import ca.uhn.hl7v2.model.v231.segment.ORC;
import ca.uhn.hl7v2.model.v231.segment.PID;
import ca.uhn.hl7v2.model.v231.segment.PV1;
import ca.uhn.hl7v2.parser.CanonicalModelClassFactory;
import ca.uhn.hl7v2.parser.Parser;
import ca.uhn.hl7v2.parser.PipeParser;
import com.google.common.base.Joiner;
import com.google.common.base.Strings;
import com.google.fhir.stu3.JsonFormat;
import com.google.fhir.stu3.proto.AdministrativeGenderCode;
import com.google.fhir.stu3.proto.Bundle;
import com.google.fhir.stu3.proto.Bundle.Entry;
import com.google.fhir.stu3.proto.Bundle.Entry.Request;
import com.google.fhir.stu3.proto.BundleTypeCode;
import com.google.fhir.stu3.proto.CodeableConcept;
import com.google.fhir.stu3.proto.Coding;
import com.google.fhir.stu3.proto.ContainedResource;
import com.google.fhir.stu3.proto.DateTime;
import com.google.fhir.stu3.proto.DeviceRequest;
import com.google.fhir.stu3.proto.DiagnosticReport;
import com.google.fhir.stu3.proto.Encounter;
import com.google.fhir.stu3.proto.HTTPVerbCode;
import com.google.fhir.stu3.proto.HumanName;
import com.google.fhir.stu3.proto.Identifier;
import com.google.fhir.stu3.proto.Instant;
import com.google.fhir.stu3.proto.Observation;
import com.google.fhir.stu3.proto.Observation.Effective;
import com.google.fhir.stu3.proto.Observation.Value;
import com.google.fhir.stu3.proto.ObservationStatusCode;
import com.google.fhir.stu3.proto.Patient;
import com.google.fhir.stu3.proto.Patient.Deceased;
import com.google.fhir.stu3.proto.Patient.MultipleBirth;
import com.google.fhir.stu3.proto.Reference;
import com.google.fhir.stu3.proto.ReferenceId;
import com.google.fhir.stu3.proto.RequestStatusCode;
import com.google.fhir.stu3.proto.Uri;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Transform an HL7 message to FHIR resources. Only limited message types are supported at this
 * point. Mappings here follow the instructions on the FHIR website, e.g.
 * https://www.hl7.org/fhir/patient-mappings.html#v2.
 */
public class MessageTransformer {
  private static final Logger LOGGER =
      Logger.getLogger(MessageTransformer.class.getCanonicalName());

  private static final String CANONICAL_VERSION = "2.3.1";

  private Parser parser = new PipeParser(new CanonicalModelClassFactory(CANONICAL_VERSION));

  private boolean updateCreateEnabled;

  public MessageTransformer() {
    this(false /* updateCreateEnabled */);
  }

  public MessageTransformer(boolean updateCreateEnabled) {
    this.updateCreateEnabled = updateCreateEnabled;
  }

  /** Converts an HL7 message to the corresponding FHIR resources as a JSON string. */
  public String transform(byte[] msg) {
    String message = new String(msg);
    try {
      Message hl7Msg = parser.parse(message);
      Bundle bundle;
      if (hl7Msg instanceof ADT_A01) {
        bundle = transform((ADT_A01) hl7Msg);
      } else if (hl7Msg instanceof ORU_R01) {
        bundle = transform((ORU_R01) hl7Msg);
      } else {
        throw new UnsupportedOperationException(
            "Only ADT_A01 and ORU_R01 are supported at this time.");
      }
      return JsonFormat.getPrinter().print(bundle);
    } catch (HL7Exception | IOException e) {
      LOGGER.log(Level.SEVERE, "Unable to parse HL7 message: " + new String(msg), e);
      throw new RuntimeException(e);
    }
  }

  private Bundle transform(ADT_A01 msg) throws IOException {
    Bundle.Builder bundleBuilder =
        Bundle.newBuilder()
            .setType(BundleTypeCode.newBuilder().setValue(BundleTypeCode.Value.TRANSACTION));

    // Used for other resources to reference patient.
    String patientUuid = UUID.randomUUID().toString();
    String patientFullUrl = String.format("urn:uuid:%s", patientUuid);

    Patient patient = transform(msg.getPID());

    // Encounter mapping starts.

    PV1 pv1 = msg.getPV1();
    // Skip if no PV1 segment is present.
    if (pv1 != null) {
      Encounter encounter = transformWithReference(pv1, patientUuid);

      bundleBuilder.addEntry(
          Entry.newBuilder()
              .setResource(ContainedResource.newBuilder().setEncounter(encounter).build())
              .setRequest(
                  Request.newBuilder()
                      .setMethod(HTTPVerbCode.newBuilder().setValue(HTTPVerbCode.Value.POST))));
    }

    List<Entry.Builder> entryBuilders = generatePatientEntries(patient);
    for (Entry.Builder entryBuilder : entryBuilders) {
      // Do not set full URL if not used.
      if (bundleBuilder.getEntryCount() > 0) {
        entryBuilder.setFullUrl(Uri.newBuilder().setValue(patientFullUrl));
      }
      bundleBuilder.addEntry(0, entryBuilder);
    }

    return bundleBuilder.build();
  }

  private Bundle transform(ORU_R01 msg) throws HL7Exception {
    Bundle.Builder bundleBuilder =
        Bundle.newBuilder()
            .setType(BundleTypeCode.newBuilder().setValue(BundleTypeCode.Value.TRANSACTION));

    // Used for other resources to reference patient.
    String patientUuid = UUID.randomUUID().toString();
    String patientFullUrl = String.format("urn:uuid:%s", patientUuid);

    // Group object.
    ORU_R01_PIDPD1NK1NTEPV1PV2ORCOBRNTEOBXNTECTI group =
        msg.getPIDPD1NK1NTEPV1PV2ORCOBRNTEOBXNTECTI();
    if (group == null) {
      return bundleBuilder.build();
    }

    ORU_R01_PIDPD1NK1NTEPV1PV2 pidGroup = group.getPIDPD1NK1NTEPV1PV2();
    if (pidGroup == null) {
      return bundleBuilder.build();
    }

    Patient patient = transform(pidGroup.getPID());

    PV1 pv1 = pidGroup.getPV1PV2().getPV1();
    if (pv1 != null) {
      Encounter encounter = transformWithReference(pv1, patientUuid);

      bundleBuilder.addEntry(
          Entry.newBuilder()
              .setResource(ContainedResource.newBuilder().setEncounter(encounter).build())
              .setRequest(
                  Request.newBuilder()
                      .setMethod(HTTPVerbCode.newBuilder().setValue(HTTPVerbCode.Value.POST))));
    }

    ORU_R01_ORCOBRNTEOBXNTECTI orcGroup = group.getORCOBRNTEOBXNTECTI();
    if (orcGroup == null) {
      return bundleBuilder.build();
    }

    ORC orc = orcGroup.getORC();
    if (orc != null) {
      DeviceRequest request = transformWithReference(orc, patientUuid);

      bundleBuilder.addEntry(
          Entry.newBuilder()
              .setResource(ContainedResource.newBuilder().setDeviceRequest(request).build())
              .setRequest(
                  Request.newBuilder()
                      .setMethod(HTTPVerbCode.newBuilder().setValue(HTTPVerbCode.Value.POST))));
    }

    OBR obr = orcGroup.getOBR();
    if (obr != null) {
      DiagnosticReport diagnosticReport = transformWithReference(obr, patientUuid);

      bundleBuilder.addEntry(
          Entry.newBuilder()
              .setResource(
                  ContainedResource.newBuilder().setDiagnosticReport(diagnosticReport).build())
              .setRequest(
                  Request.newBuilder()
                      .setMethod(HTTPVerbCode.newBuilder().setValue(HTTPVerbCode.Value.POST))));
    }

    ORU_R01_OBXNTE obxNte = orcGroup.getOBXNTE();

    OBX obx = obxNte.getOBX();
    if (obx != null) {
      Observation observation = transformWithReference(obx, patientUuid);

      bundleBuilder.addEntry(
          Entry.newBuilder()
              .setResource(ContainedResource.newBuilder().setObservation(observation).build())
              .setRequest(
                  Request.newBuilder()
                      .setMethod(HTTPVerbCode.newBuilder().setValue(HTTPVerbCode.Value.POST))));
    }

    List<Entry.Builder> entryBuilders = generatePatientEntries(patient);
    for (Entry.Builder entryBuilder : entryBuilders) {
      // Do not set full URL if not used.
      if (bundleBuilder.getEntryCount() > 0) {
        entryBuilder.setFullUrl(Uri.newBuilder().setValue(patientFullUrl));
      }
      bundleBuilder.addEntry(0, entryBuilder);
    }

    return bundleBuilder.build();
  }

  /**
   * Generates entry builders for patient. If enable_update_create is turned off, then we generate a
   * PUT as well as a POST request. The example we are following is here:
   * http://hl7.org/fhir/bundle-transaction.json.html.
   */
  private List<Entry.Builder> generatePatientEntries(Patient patient) {
    List<Entry.Builder> builders = new ArrayList<>();

    builders.add(
        Entry.newBuilder()
            .setResource(ContainedResource.newBuilder().setPatient(patient))
            .setRequest(
                Request.newBuilder()
                    .setUrl(
                        Uri.newBuilder()
                            .setValue(String.format("Patient?%s", assembleQueryParams(patient))))
                    .setMethod(HTTPVerbCode.newBuilder().setValue(HTTPVerbCode.Value.PUT))));

    if (!updateCreateEnabled) {
      builders.add(
          Entry.newBuilder()
              .setResource(ContainedResource.newBuilder().setPatient(patient))
              .setRequest(
                  Request.newBuilder()
                      .setIfNoneExist(
                          com.google.fhir.stu3.proto.String.newBuilder()
                              .setValue(assembleQueryParams(patient)))
                      .setMethod(HTTPVerbCode.newBuilder().setValue(HTTPVerbCode.Value.POST))));
    }

    return builders;
  }

  private Observation transformWithReference(OBX obx, String patientUuid) throws HL7Exception {
    Observation.Builder observationBuilder = Observation.newBuilder();

    // Subject.

    // PID-3.
    observationBuilder.setSubject(
        Reference.newBuilder().setPatientId(ReferenceId.newBuilder().setValue(patientUuid)));

    // Identifier

    // OBX-3 + OBX-4, see https://www.hl7.org/fhir/observation-mappings.html#v2 for other
    // mapping options.
    Identifier.Builder idBuilder = Identifier.newBuilder();
    CodeableConcept id = TransformHelper.ceToCodeableConcept(obx.getObx3_ObservationIdentifier());
    if (id != null) {
      idBuilder.setType(id);
    }

    ST observationSubId = obx.getObx4_ObservationSubID();
    if (observationSubId != null && observationSubId.getValue() != null) {
      idBuilder.setValue(
          com.google.fhir.stu3.proto.String.newBuilder().setValue(observationSubId.getValue()));
    }

    if ((observationSubId != null && observationSubId.getValue() != null) && id != null) {
      observationBuilder.addIdentifier(idBuilder.build());
    }

    // Value.

    // OBX-5. How to use OBX-2 and OBX-6?
    Varies[] values = obx.getObx5_ObservationValue();
    if (values != null) {
      StringBuilder sb = new StringBuilder();
      for (Varies value : values) {
        if (value.getData() != null && !value.getData().isEmpty()) {
          sb.append(value.getData().toString());
          sb.append("\n");
        }
      }

      observationBuilder.setValue(
          Value.newBuilder()
              .setStringValue(
                  com.google.fhir.stu3.proto.String.newBuilder().setValue(sb.toString())));
    }

    // Effective.

    // OBX-14. Should use OBX-19 after v2.4.
    DateTime dt = TransformHelper.tsToDateTime(obx.getObx14_DateTimeOfTheObservation());
    if (dt != null) {
      observationBuilder.setEffective(Effective.newBuilder().setDateTime(dt));
    }

    // Code.

    // OBX-3.
    if (id != null) {
      observationBuilder.setCode(id);
    }

    // Status.

    // OBX-11.
    ObservationStatusCode status =
        TransformHelper.idToObservationStatusCode(obx.getObx11_ObservationResultStatus());
    if (status != null) {
      observationBuilder.setStatus(status);
    }

    // Performer.

    // OBX-16.
    XCN[] practitioners = obx.getObx16_ResponsibleObserver();
    if (practitioners != null) {
      for (XCN xcn : practitioners) {
        observationBuilder.addPerformer(
            Reference.newBuilder()
                .setPractitionerId(
                    ReferenceId.newBuilder().setValue(xcn.getIDNumber().getValue())));
      }
    }

    // Observation methods.

    // OBX-17.
    CodeableConcept method = TransformHelper.ceToCodeableConcept(obx.getObx17_ObservationMethod());
    if (method != null) {
      observationBuilder.setMethod(method);
    }

    // How to map ST to reference range?

    return observationBuilder.build();
  }

  private DiagnosticReport transformWithReference(OBR obr, String patientUuid) {
    DiagnosticReport.Builder diagnosticReportBuilder = DiagnosticReport.newBuilder();

    // Category

    // Subject.

    // PID-3.
    diagnosticReportBuilder.setSubject(
        Reference.newBuilder().setPatientId(ReferenceId.newBuilder().setValue(patientUuid)));

    // Issued time.

    // OBR-22.
    Instant instant = TransformHelper.tsToInstant(obr.getObr22_ResultsRptStatusChngDateTime());
    if (instant != null) {
      diagnosticReportBuilder.setIssued(instant);
    }

    // Code.

    // OBR-4.
    CodeableConcept code = TransformHelper.ceToCodeableConcept(obr.getObr4_UniversalServiceID());
    if (code != null) {
      diagnosticReportBuilder.setCode(code);
    }

    return diagnosticReportBuilder.build();
  }

  private DeviceRequest transformWithReference(ORC orc, String patientUuid) {
    DeviceRequest.Builder deviceRequestBuilder = DeviceRequest.newBuilder();

    // Device request identifier.

    // ORC-2.
    Identifier placerOrderNumber = TransformHelper.eiToIdentifier(orc.getPlacerOrderNumber());
    if (placerOrderNumber != null) {
      deviceRequestBuilder.addIdentifier(placerOrderNumber);
    }

    // Subject.

    // PID-3.
    deviceRequestBuilder.setSubject(
        Reference.newBuilder().setPatientId(ReferenceId.newBuilder().setValue(patientUuid)));

    // Group identifier.

    // ORC-4.
    Identifier placerGroupNumber = TransformHelper.eiToIdentifier(orc.getPlacerGroupNumber());
    if (placerGroupNumber != null) {
      deviceRequestBuilder.setGroupIdentifier(placerGroupNumber);
    }

    // Status.

    // ORC-5.
    RequestStatusCode statusCode = TransformHelper.idToRequestStatusCode(orc.getOrc5_OrderStatus());
    if (statusCode != null) {
      deviceRequestBuilder.setStatus(statusCode);
    }

    CodeableConcept reasonCode =
        TransformHelper.ceToCodeableConcept(orc.getOrc16_OrderControlCodeReason());
    if (reasonCode != null) {
      deviceRequestBuilder.addReasonCode(reasonCode);
    }

    return deviceRequestBuilder.build();
  }

  private Encounter transformWithReference(PV1 pv1, String patientUuid) {
    Encounter.Builder encounterBuilder = Encounter.newBuilder();

    // Encounter identifier.

    // PV1-19.
    Identifier encounterId = TransformHelper.cxToId(pv1.getPv119_VisitNumber());
    if (encounterId != null) {
      encounterBuilder.addIdentifier(encounterId);
    }

    // Patient class.

    // PV1-2.
    Coding patientClass = TransformHelper.isToPatientClass(pv1.getPv12_PatientClass());
    if (patientClass != null) {
      encounterBuilder.setClass_(patientClass);
    }

    // Subject.

    // PID-3.
    encounterBuilder.setSubject(
        Reference.newBuilder().setPatientId(ReferenceId.newBuilder().setValue(patientUuid)));

    return encounterBuilder.build();
  }

  private Patient transform(PID pid) {
    // Patient mapping starts.

    Patient.Builder patientBuilder = Patient.newBuilder();

    // Patient identifiers.

    // PID-3.
    for (CX id : pid.getPid3_PatientIdentifierList()) {
      Identifier identifier = TransformHelper.cxToId(id);
      if (identifier != null) {
        patientBuilder.addIdentifier(identifier);
      }
    }

    // Patient name.

    // PID-5.
    for (XPN name : pid.getPid5_PatientName()) {
      patientBuilder.addName(TransformHelper.xpnToName(name));
    }

    // PID-9.
    for (XPN alias : pid.getPid9_PatientAlias()) {
      patientBuilder.addName(TransformHelper.xpnToName(alias));
    }

    // Patient gender.

    // PID-8.
    AdministrativeGenderCode sex = TransformHelper.isToSex(pid.getPid8_Sex());
    if (sex != null) {
      patientBuilder.setGender(sex);
    }

    // Deceased.

    // PID-30.
    Deceased deceased = TransformHelper.idToDeceased(pid.getPid30_PatientDeathIndicator());
    if (deceased != null) {
      patientBuilder.setDeceased(deceased);
    }

    // Address.

    // PID-11.
    for (XAD address : pid.getPid11_PatientAddress()) {
      patientBuilder.addAddress(TransformHelper.xadToAddress(address));
    }

    // Marital status.

    // PID-16.
    CodeableConcept martialStatus =
        TransformHelper.ceToCodeableConcept(pid.getPid16_MaritalStatus());
    if (martialStatus != null) {
      patientBuilder.setMaritalStatus(martialStatus);
    }

    // Multiple birth.

    // PID-24.
    MultipleBirth multipleBirthIndicator =
        TransformHelper.idToMultipleBirth(pid.getPid24_MultipleBirthIndicator());
    if (multipleBirthIndicator != null) {
      patientBuilder.setMultipleBirth(multipleBirthIndicator);
    }

    return patientBuilder.build();
  }

  // Assembles the query params for a patient.
  private static String assembleQueryParams(Patient patient) {
    List<String> params = new ArrayList<>();
    if (patient.getIdentifierCount() > 0) {
      com.google.fhir.stu3.proto.String identifier = patient.getIdentifier(0).getValue();
      if (identifier != null && !Strings.isNullOrEmpty(identifier.getValue())) {
        params.add(String.format("identifier:exact=%s", identifier.getValue()));
      }
    }

    if (patient.getNameCount() > 0) {
      HumanName patientName = patient.getName(0);
      if (patientName.getFamily() != null) {
        String family = patientName.getFamily().getValue().trim();
        if (!Strings.isNullOrEmpty(family)) {
          params.add(String.format("family:exact=%s", family));
        }
      }

      if (patientName.getGivenList() != null) {
        for (com.google.fhir.stu3.proto.String givenName : patientName.getGivenList()) {
          if (givenName.getValue() != null) {
            String given = givenName.getValue().trim();
            if (!Strings.isNullOrEmpty(given)) {
              params.add(String.format("given:exact=%s", given));
            }
          }
        }
      }
    }

    if (params.isEmpty()) {
      throw new RuntimeException("No identifier or name for patient.");
    }

    return Joiner.on('&').join(params);
  }
}
