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

var baseData = {
    data: {
        x: 'ts',
        xFormat: '%Y/%m/%d %H:%M',
        columns: [],
    },
    axis: {
        x: {
            type: 'timeseries',
            tick: {
                format: '%Y-%m-%d %H:%M',
            }
        }
    },
    point: {
        show: true,
        r: 2
    }
};

function populateProbeData() {
    let gd = psd[probe];

    d[probe] = structuredClone(baseData);
    d[probe].bindto = '#chart_' + probe;

    // Timestamps
    let x = ['ts'];

    // Construct x-axis using startTime and endTime.
    for (let s = gd.StartTime; s < gd.EndTime; s = s + gd.ResSeconds) {
        dt = new Date(s * 1000);
        m = dt.getMonth() + 1;
        dtStr = dt.getFullYear() + '/' + m + '/' + dt.getDate() + ' ' + dt.getHours() + ':' + dt.getMinutes();
        x.push(dtStr);
    }
    d[probe].data.columns.push(x);
    if (x.length < 31) {
        d[probe].point.show = true;
    }

    // Data colums
    for (tgt in gd.Values) {
        let vals = gd.Values[tgt];
        let freqs = gd.Freqs[tgt];
        let c = [tgt];
        for (var i = 0; i < vals.length; i++) {
            let val = vals[i]
            if (val == -1) {
                val = null;
            }
            c.push(...Array(freqs[i]).fill(val));
        }
        d[probe].data.columns.push(c);
    }
}

function populateD() {
    for (probe in psd) {
        populateProbeData(probe);
    }
}

$(document).ready(function(){
    $("#show-hide-debug-info").click(function(){
      $("#debug-info").toggle();
    });
});