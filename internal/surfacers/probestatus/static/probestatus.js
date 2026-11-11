// Copyright 2022 The Cloudprober Authors.
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

let baseData = {
  data: {
    x: 'ts',
    xFormat: '%Y/%m/%d %H:%M',
    columns: [],
  },
  size: {
    height: 160,
  },
  // Room for the last x-axis label, which c3 otherwise cuts off.
  padding: {
    right: 50,
  },
  // The page is redrawn on every change, so there's nothing to animate.
  transition: {
    duration: 0,
  },
  axis: {
    x: {
      type: 'timeseries',
      tick: {
        format: '%Y-%m-%d %H:%M',
      },
    },
    // Pin the axis to 0-1, and pad it by a few pixels (c3's default padding
    // unit), so that lines at 0 and 1 stay clear of the chart's edges.
    y: {
      min: 0,
      max: 1,
      padding: { top: 8, bottom: 8 },
      tick: {
        values: [0, 0.25, 0.5, 0.75, 1],
      },
    },
  },
  point: {
    show: true,
    r: 2,
  },
};

function populateProbeData(probe) {
  let gd = psd[probe];

  d[probe] = structuredClone(baseData);

  // Probe names can contain characters that are special in CSS selectors,
  // e.g. '.' (https://github.com/cloudprober/cloudprober/issues/362).
  d[probe].bindto = '#' + CSS.escape('chart_' + probe);

  // Timestamps
  let x = ['ts'];

  // Construct x-axis using startTime and endTime.
  for (let s = gd.StartTime; s < gd.EndTime; s = s + gd.ResSeconds) {
    const dt = new Date(s * 1000);
    const m = dt.getMonth() + 1;
    const dtStr =
      dt.getFullYear() +
      '/' +
      m +
      '/' +
      dt.getDate() +
      ' ' +
      dt.getHours() +
      ':' +
      dt.getMinutes();
    x.push(dtStr);
  }
  d[probe].data.columns.push(x);
  if (x.length > 500) {
    d[probe].point.r = 1;
  }

  // Data colums
  for (const tgt in gd.Values) {
    let vals = gd.Values[tgt];
    let freqs = gd.Freqs[tgt];
    let c = [tgt];
    for (let i = 0; i < vals.length; i++) {
      let val = vals[i];
      if (val == -1) {
        val = null;
      }
      c.push(...Array(freqs[i]).fill(val));
    }
    d[probe].data.columns.push(c);
  }
}

function populateD() {
  for (const probe in psd) {
    populateProbeData(probe);
  }
}

// Returns null if dt is an Invalid Date, e.g. from an unparseable input.
function dateToValStr(dt) {
  if (isNaN(dt.getTime())) {
    return null;
  }
  dt.setMinutes(dt.getMinutes() - dt.getTimezoneOffset());
  return dt.toISOString().slice(0, 16);
}

// Time picker: a Grafana-style button that opens a panel of quick ranges and a
// custom duration/end time. It reads and writes the graph_duration and
// graph_endtime URL parameters, which the server uses to draw the graphs.

const quickRanges = [
  ['15m', 15 * 60],
  ['1h', 3600],
  ['6h', 6 * 3600],
  ['24h', 24 * 3600],
  ['72h', 3 * 86400],
  ['168h', 7 * 86400],
  ['720h', 30 * 86400],
];

// parseDuration parses the subset of Go durations that make sense here, e.g.
// "90m", "6h" or "1h30m". It returns seconds, or null if s doesn't parse.
function parseDuration(s) {
  s = s.trim();
  if (!/^(\d+(\.\d+)?[hms])+$/.test(s)) {
    return null;
  }
  const unitSec = { h: 3600, m: 60, s: 1 };
  let total = 0;
  for (const m of s.matchAll(/(\d+(?:\.\d+)?)([hms])/g)) {
    total += parseFloat(m[1]) * unitSec[m[2]];
  }
  return total > 0 ? Math.round(total) : null;
}

// humanizeSec returns e.g. "15 minutes", "6 hours" or "3 days".
function humanizeSec(sec) {
  const plural = (n, unit) => n + ' ' + unit + (n === 1 ? '' : 's');
  if (sec >= 2 * 86400 && sec % 86400 === 0) {
    return plural(sec / 86400, 'day');
  }
  if (sec >= 3600 && sec % 3600 === 0) {
    return plural(sec / 3600, 'hour');
  }
  if (sec >= 60 && sec % 60 === 0) {
    return plural(sec / 60, 'minute');
  }
  return plural(sec, 'second');
}

// currentRange returns the graph range from the URL: its duration in seconds,
// and its end time in seconds since epoch, or null if it ends now.
function currentRange() {
  const params = new URLSearchParams(location.search);
  let durSec = maxDurationSec;
  if (params.get('graph_duration')) {
    durSec = parseDuration(params.get('graph_duration')) || maxDurationSec;
  }
  let endSec = null;
  const e = parseInt(params.get('graph_endtime'), 10);
  if (!isNaN(e)) {
    endSec = e;
  }
  return { durSec, endSec };
}

// setRange navigates to the given range. A null duration or end time drops
// the parameter, so that the server uses its default.
function setRange(duration, endSec) {
  const params = new URLSearchParams(location.search);
  if (duration) {
    params.set('graph_duration', duration);
  } else {
    params.delete('graph_duration');
  }
  if (endSec !== null && endSec < Date.now() / 1000) {
    params.set('graph_endtime', Math.floor(endSec));
  } else {
    params.delete('graph_endtime');
  }
  window.location.search = params.toString();
}

// durationParam returns the graph_duration value for a range of sec seconds.
function durationParam(sec) {
  return sec % 3600 === 0 ? sec / 3600 + 'h' : sec % 60 === 0 ? sec / 60 + 'm' : sec + 's';
}

function rangeLabel(r) {
  if (r.endSec === null) {
    return 'Last ' + humanizeSec(r.durSec);
  }
  const end = new Date(r.endSec * 1000).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
  return humanizeSec(r.durSec) + ' ending ' + end;
}

function setupTimePicker() {
  const r = currentRange();
  const params = new URLSearchParams(location.search);
  const toggle = document.getElementById('tp-toggle');
  const panel = document.getElementById('tp-panel');

  toggle.textContent = rangeLabel(r) + ' ▾';

  const openPanel = (open) => {
    panel.hidden = !open;
    toggle.setAttribute('aria-expanded', open);
  };
  toggle.addEventListener('click', () => openPanel(panel.hidden));
  document.addEventListener('click', (e) => {
    if (!e.target.closest('.time-picker')) {
      openPanel(false);
    }
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      openPanel(false);
    }
  });

  // Quick ranges, up to how far back the status page keeps data. Like
  // Grafana's, they always end now.
  const list = document.getElementById('tp-quick-list');
  for (const [dur, sec] of quickRanges) {
    if (sec > maxDurationSec) {
      continue;
    }
    const li = document.createElement('li');
    const a = document.createElement('a');
    a.href = '#';
    a.textContent = 'Last ' + humanizeSec(sec);
    if (sec === r.durSec && r.endSec === null) {
      a.className = 'tp-active';
    }
    a.addEventListener('click', (e) => {
      e.preventDefault();
      setRange(dur, null);
    });
    li.appendChild(a);
    list.appendChild(li);
  }

  // Custom range.
  const durInput = document.getElementById('tp-duration');
  const endInput = document.getElementById('tp-endtime');
  durInput.value = params.get('graph_duration') || '';
  if (r.endSec !== null) {
    endInput.value = dateToValStr(new Date(r.endSec * 1000)) || '';
  }
  const min = dateToValStr(new Date(startTime));
  if (min) {
    endInput.min = min;
  }
  endInput.max = dateToValStr(new Date());

  document.getElementById('tp-custom').addEventListener('submit', (e) => {
    e.preventDefault();
    const errDiv = document.getElementById('tp-error');
    const dur = durInput.value.trim();
    if (dur !== '' && parseDuration(dur) === null) {
      errDiv.textContent = 'Invalid duration: use h, m and s, e.g. 90m or 1h30m.';
      return;
    }
    let endSec = null;
    if (endInput.value !== '') {
      const d = new Date(endInput.value);
      if (isNaN(d.getTime())) {
        errDiv.textContent = 'Invalid end time.';
        return;
      }
      endSec = d.getTime() / 1000;
    }
    setRange(dur || null, endSec);
  });

  // Shift the range back or forward by its own duration. Forward is only
  // useful when the range doesn't already end now, and back only while there
  // is data before the range's start.
  const nowSec = Date.now() / 1000;
  const endSec = r.endSec === null ? nowSec : r.endSec;
  const startSec = new Date(startTime).getTime() / 1000;
  const back = document.getElementById('tp-back');
  const fwd = document.getElementById('tp-fwd');
  const keepDur = params.get('graph_duration') || durationParam(r.durSec);

  back.disabled = endSec - r.durSec <= startSec;
  back.addEventListener('click', () => setRange(keepDur, endSec - r.durSec));
  fwd.disabled = r.endSec === null;
  fwd.addEventListener('click', () => setRange(keepDur, endSec + r.durSec));
}

// selectedProbes returns the selected probes in a map (object) to allow for
// quick lookups and easy modification.
function selectedProbes() {
  let params = new URLSearchParams(window.location.search);
  let currentProbes = params.getAll('probe');
  let probeMap = {};
  for (const element of currentProbes) {
    probeMap[element] = true;
  }
  return probeMap;
}

// Update probe query parameter to the provided list.
function updateProbeParams(probes) {
  let params = new URLSearchParams(window.location.search);
  params.delete('probe');
  for (const probe of probes) {
    params.append('probe', probe);
  }
  window.location.search = params.toString();
}

function handleProbeSelect(e) {
  let probe = e.target.dataset.probe;
  if (probe === undefined) {
    updateProbeParams([]);
    return;
  }

  let probeMap = selectedProbes();
  if (e.target.checked) {
    probeMap[probe] = true;
  } else {
    delete probeMap[probe];
  }
  updateProbeParams(Object.keys(probeMap));
}

// probeCheckbox returns a label wrapping a checkbox. Probe names come from the
// config and can contain any character, so they are set as text and data, and
// never parsed as HTML.
function probeCheckbox(text, probe, checked) {
  let input = $('<input type=checkbox class=probe-checkbox>').prop('checked', checked);
  if (probe !== undefined) {
    input.attr('data-probe', probe);
  }
  return $('<label>').append(input, document.createTextNode(text));
}

function generateProbeSelectors() {
  let probeMap = selectedProbes();

  let selector = $('#probe-selector');
  selector.append('Probes: ');
  selector.append(probeCheckbox('All', undefined, Object.keys(probeMap).length === 0));
  for (const p of allProbes) {
    selector.append(' | ', probeCheckbox(p, p, !!probeMap[p]));
  }

  $('.probe-checkbox').change(handleProbeSelect);
}

function showDebugInfo() {
  let params = new URLSearchParams(window.location.search);
  if (params.get('debug')) {
    $('#debug-info').show();
  }
}

$(document).ready(function () {
  setupTimePicker();
  showDebugInfo();
  generateProbeSelectors();
});
