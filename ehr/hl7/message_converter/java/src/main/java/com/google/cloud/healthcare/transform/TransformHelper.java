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

import ca.uhn.hl7v2.model.DataTypeException;
import ca.uhn.hl7v2.model.primitive.CommonTM;
import ca.uhn.hl7v2.model.v231.datatype.CE;
import ca.uhn.hl7v2.model.v231.datatype.CX;
import ca.uhn.hl7v2.model.v231.datatype.EI;
import ca.uhn.hl7v2.model.v231.datatype.FN;
import ca.uhn.hl7v2.model.v231.datatype.ID;
import ca.uhn.hl7v2.model.v231.datatype.IS;
import ca.uhn.hl7v2.model.v231.datatype.ST;
import ca.uhn.hl7v2.model.v231.datatype.TS;
import ca.uhn.hl7v2.model.v231.datatype.XAD;
import ca.uhn.hl7v2.model.v231.datatype.XPN;
import com.google.fhir.stu3.proto.Address;
import com.google.fhir.stu3.proto.AddressUseCode;
import com.google.fhir.stu3.proto.AdministrativeGenderCode;
import com.google.fhir.stu3.proto.Boolean;
import com.google.fhir.stu3.proto.Code;
import com.google.fhir.stu3.proto.CodeableConcept;
import com.google.fhir.stu3.proto.Coding;
import com.google.fhir.stu3.proto.DateTime;
import com.google.fhir.stu3.proto.HumanName;
import com.google.fhir.stu3.proto.Identifier;
import com.google.fhir.stu3.proto.Instant;
import com.google.fhir.stu3.proto.Instant.Precision;
import com.google.fhir.stu3.proto.ObservationStatusCode;
import com.google.fhir.stu3.proto.Patient.Deceased;
import com.google.fhir.stu3.proto.Patient.MultipleBirth;
import com.google.fhir.stu3.proto.RequestStatusCode;
import com.google.fhir.stu3.proto.String;
import com.google.fhir.stu3.proto.Uri;

/**
 * This helper follows the mappings published here:
 * https://www.hl7.org/fhir/datatypes-mappings.html.
 */
class TransformHelper {
  static Identifier cxToId(CX cx) {
    if (cx == null
        || cx.getIdentifierTypeCode() == null
        || cx.getIdentifierTypeCode().getValue() == null) {
      return null;
    }

    return Identifier.newBuilder()
        .setType(
            CodeableConcept.newBuilder()
                .setText(String.newBuilder().setValue(cx.getCx5_IdentifierTypeCode().getValue())))
        .build();
  }

  static HumanName xpnToName(XPN xpn) {
    if (xpn == null) {
      return null;
    }

    HumanName.Builder builder = HumanName.newBuilder();

    FN familyName = xpn.getXpn1_FamilyLastName();
    if (familyName != null
        && familyName.getFamilyName() != null
        && familyName.getFamilyName().getValue() != null) {
      builder.setFamily(String.newBuilder().setValue(familyName.getFamilyName().getValue()));
    }

    ST givenName = xpn.getXpn2_GivenName();
    if (givenName != null && givenName.getValue() != null) {
      builder.addGiven(String.newBuilder().setValue(givenName.getValue()));
    }

    ST middleOrInitial = xpn.getXpn3_MiddleInitialOrName();
    if (middleOrInitial != null && middleOrInitial.getValue() != null) {
      builder.addGiven(String.newBuilder().setValue(middleOrInitial.getValue()));
    }

    ST prefix = xpn.getXpn5_PrefixEgDR();
    if (prefix != null && prefix.getValue() != null) {
      builder.addPrefix(String.newBuilder().setValue(prefix.getValue()));
    }

    return builder.build();
  }

  static AdministrativeGenderCode isToSex(IS is) {
    if (is == null || is.getValue() == null) {
      return null;
    }

    AdministrativeGenderCode.Value val;
    switch (is.getValue()) {
      case "M":
        val = AdministrativeGenderCode.Value.MALE;
        break;
      case "F":
        val = AdministrativeGenderCode.Value.FEMALE;
        break;
      case "O":
        val = AdministrativeGenderCode.Value.OTHER;
        break;
      default:
        val = AdministrativeGenderCode.Value.UNKNOWN;
    }
    return AdministrativeGenderCode.newBuilder().setValue(val).build();
  }

  static Deceased idToDeceased(ID id) {
    if (id == null || id.getValue() == null) {
      return null;
    }

    switch (id.getValue()) {
      case "Y":
        return Deceased.newBuilder().setBoolean(Boolean.newBuilder().setValue(true)).build();
      case "N":
        return Deceased.newBuilder().setBoolean(Boolean.newBuilder().setValue(false)).build();
      default:
        return null;
    }
  }

  static Address xadToAddress(XAD xad) {
    if (xad == null) {
      return null;
    }

    Address.Builder builder = Address.newBuilder();

    ST streetAddress = xad.getXad1_StreetAddress();
    if (streetAddress != null && streetAddress.getValue() != null) {
      builder.addLine(String.newBuilder().setValue(streetAddress.getValue()));
    }

    ST otherDesignation = xad.getXad2_OtherDesignation();
    if (otherDesignation != null && otherDesignation.getValue() != null) {
      builder.addLine(String.newBuilder().setValue(otherDesignation.getValue()));
    }

    ST city = xad.getXad3_City();
    if (city != null && city.getValue() != null) {
      builder.setCity(String.newBuilder().setValue(city.getValue()));
    }

    IS district = xad.getXad9_CountyParishCode();
    if (district != null && district.getValue() != null) {
      builder.setDistrict(String.newBuilder().setValue(district.getValue()));
    }

    ST state = xad.getXad4_StateOrProvince();
    if (state != null && state.getValue() != null) {
      builder.setState(String.newBuilder().setValue(state.getValue()));
    }

    ID country = xad.getXad6_Country();
    if (country != null && country.getValue() != null) {
      builder.setCountry(String.newBuilder().setValue(country.getValue()));
    }

    ST postalCode = xad.getXad5_ZipOrPostalCode();
    if (postalCode != null && postalCode.getValue() != null) {
      builder.setPostalCode(String.newBuilder().setValue(postalCode.getValue()));
    }

    return builder.build();
  }

  static MultipleBirth idToMultipleBirth(ID id) {
    if (id == null || id.getValue() == null) {
      return null;
    }

    switch (id.getValue()) {
      case "Y":
        return MultipleBirth.newBuilder().setBoolean(Boolean.newBuilder().setValue(true)).build();
      case "N":
        return MultipleBirth.newBuilder().setBoolean(Boolean.newBuilder().setValue(false)).build();
      default:
        return null;
    }
  }

  static Coding isToPatientClass(IS is) {
    if (is == null || is.getValue() == null) {
      return null;
    }

    switch (is.getValue()) {
      case "I": // inpatient
        return Coding.newBuilder().setCode(Code.newBuilder().setValue("inpatient")).build();
      case "O": // outpatient
        return Coding.newBuilder().setCode(Code.newBuilder().setValue("outpatient")).build();
      case "P": // pre-admission
        return Coding.newBuilder().setCode(Code.newBuilder().setValue("preadmission")).build();
      case "E": // emergency
        return Coding.newBuilder().setCode(Code.newBuilder().setValue("emergency")).build();
      default:
        // Unrecognized value.
        return null;
    }
  }

  static Identifier eiToIdentifier(EI ei) {
    if (ei == null) {
      return null;
    }

    // Value.

    // EI-1.
    Identifier.Builder builder = Identifier.newBuilder();

    ST entityIdentifier = ei.getEi1_EntityIdentifier();
    if (entityIdentifier != null && entityIdentifier.getValue() != null) {
      builder.setValue(String.newBuilder().setValue(entityIdentifier.getValue()));
    }

    // System.

    // EI-2-4.
    StringBuilder sb = new StringBuilder();

    ID universalIdType = ei.getEi4_UniversalIDType();
    if (universalIdType != null && universalIdType.getValue() != null) {
      sb.append(universalIdType.getValue());
    }
    sb.append("-");

    ST universalId = ei.getEi3_UniversalID();
    if (universalId != null && universalId.getValue() != null) {
      sb.append(universalId.getValue());
    }
    sb.append("-");

    IS namespaceId = ei.getNamespaceID();
    if (namespaceId != null && namespaceId.getValue() != null) {
      sb.append(namespaceId.getValue());
    }

    builder.setSystem(Uri.newBuilder().setValue(sb.toString()));
    return builder.build();
  }

  // Used v2.2 0085 table.
  static ObservationStatusCode idToObservationStatusCode(ID id) {
    if (id == null || id.getValue() == null) {
      return null;
    }

    switch (id.getValue()) {
      case "C":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.CORRECTED).build();
      case "D":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.UNKNOWN).build();
      case "F":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.FINAL).build();
      case "I":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.UNKNOWN).build();
      case "P":
        return ObservationStatusCode.newBuilder()
            .setValue(ObservationStatusCode.Value.PRELIMINARY)
            .build();
      case "R":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.UNKNOWN).build();
      case "S":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.UNKNOWN).build();
      case "U":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.UNKNOWN).build();
      case "X":
        return ObservationStatusCode.newBuilder().setValue(ObservationStatusCode.Value.UNKNOWN).build();
      default:
        return ObservationStatusCode.newBuilder()
            .setValue(ObservationStatusCode.Value.UNRECOGNIZED)
            .build();
    }
  }

  static RequestStatusCode idToRequestStatusCode(ID id) {
    if (id == null || id.getValue() == null) {
      return null;
    }

    switch (id.getValue()) {
      case "A":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.ACTIVE).build();
      case "CA":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.CANCELLED).build();
      case "CM":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.COMPLETED).build();
      case "DC":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.CANCELLED).build();
      case "ER":
        return RequestStatusCode.newBuilder()
            .setValue(RequestStatusCode.Value.ENTERED_IN_ERROR)
            .build();
      case "HD":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.SUSPENDED).build();
      case "IP":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.ACTIVE).build();
      case "RP":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.UNKNOWN).build();
      case "SC":
        return RequestStatusCode.newBuilder().setValue(RequestStatusCode.Value.ACTIVE).build();
      default:
        return RequestStatusCode.newBuilder()
            .setValue(RequestStatusCode.Value.UNRECOGNIZED)
            .build();
    }
  }

  static CodeableConcept ceToCodeableConcept(CE... ces) {
    if (ces == null) {
      return null;
    }

    CodeableConcept.Builder ccBuilder = CodeableConcept.newBuilder();
    for (CE ce : ces) {
      Coding.Builder builder = Coding.newBuilder();

      ST nameOfCodingSystem = ce.getCe3_NameOfCodingSystem();
      boolean nameOfCodingSystemSet =
          nameOfCodingSystem != null && nameOfCodingSystem.getValue() != null;
      if (nameOfCodingSystemSet) {
        builder.setSystem(Uri.newBuilder().setValue(nameOfCodingSystem.getValue()));
      }

      ST identifier = ce.getCe1_Identifier();
      boolean identifierSet = identifier != null && identifier.getValue() != null;
      if (identifierSet) {
        builder.setCode(Code.newBuilder().setValue(identifier.getValue()));
      }

      ST text = ce.getCe2_Text();
      boolean textSet = text != null && text.getValue() != null;
      if (textSet) {
        builder.setDisplay(String.newBuilder().setValue(text.getValue()));
      }

      if (nameOfCodingSystemSet || identifierSet || textSet) {
        ccBuilder.addCoding(builder);
      }
    }

    if (ccBuilder.getCodingCount() == 0) {
      return null;
    }

    return ccBuilder.build();
  }

  static Instant tsToInstant(TS ts) {
    if (ts == null) {
      return null;
    }

    if (ts.getTs1_TimeOfAnEvent() == null) {
      return null;
    }

    Instant.Builder builder = Instant.newBuilder();
    try {
      builder.setValueUs(ts.getTs1_TimeOfAnEvent().getValueAsDate().getTime() * 1000);
      builder.setTimezone(formatOffset(ts.getTs1_TimeOfAnEvent().getGMTOffset()));
      builder.setPrecision(Precision.SECOND);
    } catch (DataTypeException e) {
      return null;
    }

    return builder.build();
  }

  static DateTime tsToDateTime(TS ts) {
    if (ts == null) {
      return null;
    }

    if (ts.getTs1_TimeOfAnEvent() == null) {
      return null;
    }

    DateTime.Builder builder = DateTime.newBuilder();
    try {
      builder.setValueUs(ts.getTs1_TimeOfAnEvent().getValueAsDate().getTime() * 1000);
      builder.setTimezone(formatOffset(ts.getTs1_TimeOfAnEvent().getGMTOffset()));
      builder.setPrecision(DateTime.Precision.SECOND);
    } catch (DataTypeException e) {
      return null;
    }

    return builder.build();
  }

  // Formats time zone offset, e.g. from 800 to "+08:00"
  private static java.lang.String formatOffset(int offset) {
    if (offset == CommonTM.GMT_OFFSET_NOT_SET_VALUE) {
      return "+00:00";
    }

    StringBuilder sb = new StringBuilder().append(offset < 0 ? "-" : "+");
    offset = Math.abs(offset);

    int hour = offset / 100;
    return sb.append(hour < 10 ? "0" : "")
        .append(hour)
        .append(":")
        .append((offset / 10) % 10)
        .append(offset % 10)
        .toString();
  }
}
