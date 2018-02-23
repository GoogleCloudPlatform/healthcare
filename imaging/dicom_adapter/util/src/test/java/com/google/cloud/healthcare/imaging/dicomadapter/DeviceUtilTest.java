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

package com.google.cloud.healthcare.imaging.dicomadapter;

import static com.google.common.truth.Truth.assertThat;

import java.util.Collection;
import java.util.List;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Connection;
import org.dcm4che3.net.Device;
import org.dcm4che3.net.TransferCapability;
import org.dcm4che3.net.service.DicomServiceRegistry;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public final class DeviceUtilTest {

  @Test
  public void testDeviceUtil() throws Exception {
    DicomServiceRegistry registry = new DicomServiceRegistry();
    TransferCapability transferCapability =
        new TransferCapability(
            null /* commonName */, /* all SOP classes */
            "*",
            TransferCapability.Role.SCP, /* all transfer syntaxes */
            "*");
    Device device = DeviceUtil.createServerDevice("server", 11111, registry, transferCapability);
    assertThat(device.getDeviceName()).isEqualTo("dicom-to-dicomweb-adapter-server");

    List<Connection> connections = device.listConnections();
    assertThat(connections).hasSize(1);
    assertThat(connections.get(0).getPort()).isEqualTo(11111);

    Collection<ApplicationEntity> applicationEntities = device.getApplicationEntities();
    assertThat(applicationEntities).hasSize(1);
    ApplicationEntity applicationEntity = applicationEntities.iterator().next();
    assertThat(applicationEntity.getAETitle()).isEqualTo("server");

    Collection<TransferCapability> transferCapabilities =
        applicationEntity.getTransferCapabilities();
    assertThat(transferCapabilities).hasSize(1);
    TransferCapability gotTransferCapability = transferCapabilities.iterator().next();
    assertThat(gotTransferCapability.getSopClass()).isEqualTo("*");
    assertThat(gotTransferCapability.getRole()).isEqualTo(TransferCapability.Role.SCP);
  }
}
