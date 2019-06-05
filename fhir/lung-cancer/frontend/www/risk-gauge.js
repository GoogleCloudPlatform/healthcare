/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/**
 * fromPolar converts polar coordinates into Cartesian coordinates.
 *
 * @param {number} radius
 * @param {number} angle Angle in degrees
 */
function fromPolar(radius, angle) {
  const rads = (angle - 90) * Math.PI / 180.0;
  return {
    x: Math.round(radius * Math.cos(rads)),
    y: Math.round(radius * Math.sin(rads)),
  };
}


class RiskGauge {
  constructor() {
    this.severity = 0;
    this.severityColors = ['', '#34A853', '#4285F4', '#FBBC04', '#EA4335'];

    this.gauge = document.querySelector('#risk_gauge path');
    this.text = document.querySelector('#risk_gauge text');
  }

  /**
   * Sets the gauge to `severity`.
   *
   * @param {number} severity
   */
  setSeverity(severity) {
    if (severity === this.severity) {
      return;
    }

    const prevSeverity = this.severity;
    this.severity = severity;
    window.requestAnimationFrame(
        ts => this.animate_(prevSeverity, severity, ts, ts));

    this.gauge.setAttribute('stroke', this.severityColors[severity]);
    this.text.innerHTML =
        ['', 'Negligible', 'Low', 'Moderate', 'High'][severity];
  }

  /**
   * Create a new animation frame for moving the dial from one severity to
   * another.
   *
   * @private
   * @param {number} startSeverity the starting severity
   * @param {number} endSeverity the ending severity
   * @param {number} start the timestamp for the start of the animation
   * @param {number} ts the current timestamp
   */
  animate_(startSeverity, endSeverity, start, ts) {
    const duration = 200 * Math.abs(startSeverity - endSeverity);
    const progress = (ts - start) / duration;

    const startAngle = [0, 0.1, 0.33, 0.66, 1][startSeverity];
    const endAngle = [0, 0.1, 0.33, 0.66, 1][endSeverity];
    const angle = startAngle + (endAngle - startAngle) * progress;

    this.gauge.setAttribute('d', this.getDialPath_(angle));

    if (progress < 1) {
      window.requestAnimationFrame(
          ts => this.animate_(startSeverity, endSeverity, start, ts));
    }
  }

  /**
   * Constructs a new path for the SVG dial that has the dial filled to `angle`.
   *
   * @private
   * @param {number} angle
   */
  getDialPath_(angle) {
    const start = 230;
    let end = 490;
    const x = 200;
    const y = 200;
    const radius = 150;

    end = start + ((end - start) * angle);
    const {
      x: sx,
      y: sy,
    } = fromPolar(radius, end);
    const {
      x: ex,
      y: ey,
    } = fromPolar(radius, start);

    const isLargeArc = end - start <= 180 ? '0' : '1';

    return `M ${x + sx} ${y + sy} A ${radius} ${radius} 0 ${isLargeArc} 0 ${
        x + ex} ${y + ey}`;
  }
}
