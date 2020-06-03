# Copyright 2020 Google LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Cloud Monitoring Client."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from typing import Text
from absl import logging
from google.rpc import code_pb2


class MonitoringClient(object):
  """Creates Monitoring Client to write systematic logging custom metrics."""

  def WriteErrorMetric(self, status_code: code_pb2.Code) -> None:
    """Logs the custom error metric which can be monitoried on Stackdriver.

    This method logs the error type and using log-based metrics it can be
    monitoried.

    Args:
      status_code: code_pb2.Code. The status code of the error.
    """
    logging.info('Error type: %s', code_pb2.Code.Name(status_code))

  def WritePerformanceMetric(self, metric_name: Text, value: float) -> None:
    """Logs the performance metric value which can be monitoried on Stackdriver.

    This method logs the metric and the value passed. Using log-based metrics
    it can be monitoried.

    Args:
      metric_name: Metric name to write values to.
      value: float. The value of the point e.g. 5.4563 for representing seconds.
    """
    value_sec = round(value, 2)
    logging.info('Performance Metric: %s %.2fs ', metric_name, value_sec)
