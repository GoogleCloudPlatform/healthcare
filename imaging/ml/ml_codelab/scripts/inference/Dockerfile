# Copyright 2018 Google LLC.  All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM google/cloud-sdk

# Copy inference module code.
RUN mkdir -p /opt/inference_module/src && mkdir -p /opt/inference_module/bin
ADD / /opt/inference_module/src/

# Install dependencies.
RUN pip install --upgrade pip && pip install --upgrade virtualenv && \
    virtualenv /opt/inference_module/venv && \
    . /opt/inference_module/venv/bin/activate && \
    cd /opt/inference_module/src/ && \
    python setup.py install

# Create script to run inference module.
RUN printf '#!/bin/bash\n%s\n%s' \
      ". /opt/inference_module/venv/bin/activate && cd /opt/inference_module/src" \
      'python inference.py "$@"' > \
      /opt/inference_module/bin/inference_module && \
    chmod +x /opt/inference_module/bin/inference_module
