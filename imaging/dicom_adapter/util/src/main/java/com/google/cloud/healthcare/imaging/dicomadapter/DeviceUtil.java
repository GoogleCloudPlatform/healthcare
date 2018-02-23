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

import java.util.concurrent.Executors;
import org.dcm4che3.net.ApplicationEntity;
import org.dcm4che3.net.Connection;
import org.dcm4che3.net.Device;
import org.dcm4che3.net.TransferCapability;
import org.dcm4che3.net.service.DicomServiceRegistry;

/** Static utilities for creating {@link Device} used in dcm4che library. */
public class DeviceUtil {
  private DeviceUtil() {}

  private static final String ALL_ALLOWED_SOP_CLASSES = "*";
  private static final String ALL_ALLOWED_TRANSFER_SYNTAXES = "*";

  /** Creates a DICOM server listening to the port for the given services handling all syntaxes */
  static Device createServerDevice(
      String applicationEntityName, Integer dicomPort, DicomServiceRegistry serviceRegistry) {
    TransferCapability transferCapability =
        new TransferCapability(
            null /* commonName */,
            ALL_ALLOWED_SOP_CLASSES,
            TransferCapability.Role.SCP,
            ALL_ALLOWED_TRANSFER_SYNTAXES);
    return createServerDevice(
        applicationEntityName, dicomPort, serviceRegistry, transferCapability);
  }

  /** Creates a DICOM server listening to the port for the given services */
  public static Device createServerDevice(
      String applicationEntityName,
      Integer dicomPort,
      DicomServiceRegistry serviceRegistry,
      TransferCapability transferCapability) {
    // Create a DICOM device.
    Device device = new Device("dicom-to-dicomweb-adapter-server");
    Connection connection = new Connection();
    connection.setPort(dicomPort);
    device.addConnection(connection);

    // Create an application entity (a network node) listening on input port.
    ApplicationEntity applicationEntity = new ApplicationEntity(applicationEntityName);
    applicationEntity.setAssociationAcceptor(true);
    applicationEntity.addConnection(connection);
    applicationEntity.addTransferCapability(transferCapability);
    device.addApplicationEntity(applicationEntity);

    // Add the DICOM request handlers to the device.
    device.setDimseRQHandler(serviceRegistry);
    device.setScheduledExecutor(Executors.newSingleThreadScheduledExecutor());
    device.setExecutor(Executors.newCachedThreadPool());
    return device;
  }

  /** Creates a DICOM client device containing the given Application Entity */
  public static Device createClientDevice(
      ApplicationEntity applicationEntity, Connection connection) {
    Device device = new Device("dicom-to-dicomweb-adapter-client");
    device.addConnection(connection);
    device.addApplicationEntity(applicationEntity);
    device.setScheduledExecutor(Executors.newSingleThreadScheduledExecutor());
    device.setExecutor(Executors.newCachedThreadPool());
    return device;
  }
}
