// Copyright 2025 The Cloudprober Authors.
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

// Helper function to format Date to datetime-local string (YYYY-MM-DDThh:mm)
function formatDateTime(date) {
    const pad = (num) => num.toString().padStart(2, "0");
    return date.getFullYear() + "-" + pad(date.getMonth() + 1) + "-" + pad(date.getDate()) + "T" + pad(date.getHours()) + ":" + pad(date.getMinutes());
}

function stringToTime(s) {
    if (!s) {
        return null;
    }
    const n = parseInt(s, 10)
    if (isNaN(n)) {
        console.error("Invalid time string: " + s)
        return null;
    }
    return new Date(n);
}

function getLocalTimeZoneAbbreviation() {
    const date = new Date();
    const timeZoneName = new Intl.DateTimeFormat("en-US", { timeZoneName: "short" })
      .formatToParts(date)
      .find((part) => part.type === "timeZoneName")
      .value;
    return timeZoneName;
}

const timeZoneAbbrElements = document.getElementsByClassName('time-zone-abbr');
for (let i = 0; i < timeZoneAbbrElements.length; i++) {
    timeZoneAbbrElements[i].textContent = getLocalTimeZoneAbbreviation();
}

const startDatetimeInput = document.getElementById('start-datetime');
const endDatetimeInput = document.getElementById('end-datetime');
const errorMessage = document.getElementById('error-message');

const urlParams = new URLSearchParams(window.location.search);

let endDateTime;
if (urlParams.has('endTime')) {
    endDateTime = stringToTime(urlParams.get('endTime'));
} else {
    endDateTime = new Date();
}
endDatetimeInput.value = formatDateTime(endDateTime);

let startDateTime;
if (urlParams.has('startTime')) {
    startDateTime = stringToTime(urlParams.get('startTime'));
} else {
    startDateTime = new Date(endDateTime.getTime() - 24 * 60 * 60 * 1000); // 24 hours before endTime
}
startDatetimeInput.value = formatDateTime(startDateTime);

function validate() {
    const startValue = startDatetimeInput.value;
    const endValue = endDatetimeInput.value;

    if (startValue && endValue) {
        const start = new Date(startValue);
        const end = new Date(endValue);

        if (end <= start) {
            errorMessage.classList.add('show');
            endDatetimeInput.setCustomValidity('End date and time must be after start');
            return false;
        } else {
            errorMessage.classList.remove('show');
            endDatetimeInput.setCustomValidity('');
            return true;
        }
    }
    return true;
}

function updateUrlParams() {
    const start = new Date(startDatetimeInput.value);
    const end = new Date(endDatetimeInput.value);

    const params = new URLSearchParams(window.location.search);
    let changed = false;
    if (params.get('endTime') != end.getTime().toString()) {
        params.set('endTime', end.getTime().toString());
        changed = true;
    }
    if (params.get('startTime') != start.getTime().toString()) {
        params.set('startTime', start.getTime().toString());
        changed = true;
    }
    if (changed) {
        window.location.search = params.toString();
    }
}

function onChange() {
    if (validate()) {
        updateUrlParams();
    }
}

function failureOnlyChange() {
    const params = new URLSearchParams(window.location.search);
    if (failureOnlyInput.checked) {
        params.set('failureOnly', 'true');
    } else {
        params.delete('failureOnly');
    }
    window.location.search = params.toString();
}

startDatetimeInput.addEventListener('change', onChange);
endDatetimeInput.addEventListener('change', onChange);

const failureOnlyInput = document.getElementById('failure-only');
failureOnlyInput.checked = urlParams.get('failureOnly') === 'true';
failureOnlyInput.addEventListener('change', failureOnlyChange);

// Initial validation
validate();