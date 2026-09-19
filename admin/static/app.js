(() => {
  "use strict";

  const state = {
    page: location.hash.slice(1) || "dashboard",
    apiKey: sessionStorage.getItem("nanomdm.apiKey") || "",
    topic: localStorage.getItem("nanomdm.topic") || "",
    version: null,
    connected: false,
    devices: JSON.parse(localStorage.getItem("nanomdm.devices") || "[]"),
    jobs: JSON.parse(localStorage.getItem("nanomdm.jobs") || "[]"),
    inventory: null,
  };

  const $ = (selector) => document.querySelector(selector);
  const escapeHTML = (value) => String(value || "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#039;" }[char]));
  const save = () => {
    localStorage.setItem("nanomdm.devices", JSON.stringify(state.devices));
    localStorage.setItem("nanomdm.jobs", JSON.stringify(state.jobs.slice(0, 50)));
  };
  const uuid = () => (crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`);
  const notify = (message, type = "") => {
    const alert = $("#alert");
    alert.textContent = message;
    alert.className = `alert ${type}`;
    alert.hidden = false;
    window.clearTimeout(notify.timer);
    notify.timer = window.setTimeout(() => { alert.hidden = true; }, 4500);
  };
  const api = async (path, options = {}) => {
    const headers = new Headers(options.headers || {});
    if (state.apiKey) headers.set("Authorization", `Basic ${btoa(`nanomdm:${state.apiKey}`)}`);
    const response = await fetch(path, { ...options, headers });
    const text = await response.text();
    let body = {};
    try { body = text ? JSON.parse(text) : {}; } catch (_) { body = { raw: text }; }
    if (!response.ok) throw new Error(body.error || response.statusText || `HTTP ${response.status}`);
    return body;
  };
  const setConnection = (connected, label) => {
    state.connected = connected;
    const pill = $("#connection-state");
    pill.textContent = label;
    pill.className = `pill ${connected ? "pill-green" : "pill-muted"}`;
  };
  const jobStatus = (job) => job.status || resultStatus(job.result);
  const resultStatus = (result) => {
    if (!result) return "Failed";
    if (result.enqueue_error) return "Failed";
    if (result.push_error) return "Partial failure";
    const statuses = Object.values(result.status || {});
    if (statuses.some((entry) => entry.command_error)) return "Failed";
    if (statuses.some((entry) => entry.push_error)) return "Partial failure";
    return "Accepted";
  };
  const shell = (title, body) => `<div class="content"><div class="intro"><div><h2>${title}</h2><p>${body}</p></div></div>`;
  const closeShell = "</div>";

  function render() {
    const pages = { dashboard: "Dashboard", devices: "Devices", commands: "Commands", enrollment: "Enrollment", settings: "Settings" };
    if (!pages[state.page]) state.page = "dashboard";
    $("#page-title").textContent = pages[state.page];
    document.querySelectorAll(".nav-link").forEach((link) => link.classList.toggle("active", link.dataset.page === state.page));
    const renderers = { dashboard: renderDashboard, devices: renderDevices, commands: renderCommands, enrollment: renderEnrollment, settings: renderSettings };
    $("#app-content").innerHTML = renderers[state.page]();
    bindPage();
  }

  function renderDashboard() {
    const accepted = state.jobs.filter((job) => job.status === "Accepted").length;
    return shell("Dashboard", "A focused view of this NanoMDM instance and its live API capabilities.") +
      `<div class="grid stats">
        <div class="card"><span class="stat-icon">⌘</span><span class="stat-label">Saved device targets</span><div class="stat-value">${state.devices.length}</div><span class="stat-foot">Managed locally in this browser</span></div>
        <div class="card"><span class="stat-icon">↗</span><span class="stat-label">Commands sent</span><div class="stat-value">${state.jobs.length}</div><span class="stat-foot">${accepted} accepted by the API</span></div>
        <div class="card"><span class="stat-icon">◉</span><span class="stat-label">Server version</span><div class="stat-value" style="font-size:22px">${escapeHTML(state.version || "—")}</div><span class="stat-foot">Reported by /version</span></div>
        <div class="card"><span class="stat-icon">✓</span><span class="stat-label">API connection</span><div class="stat-value" style="font-size:22px">${state.connected ? "Healthy" : "Pending"}</div><span class="stat-foot">Basic auth to this server</span></div>
      </div>
      <div class="grid two-col">
        <div class="card"><h3>System health</h3><p class="subhead">Live checks against exposed NanoMDM endpoints.</p>
          <div class="health-row"><span>HTTP API</span><span class="pill ${state.connected ? "pill-green" : "pill-muted"}">${state.connected ? "Reachable" : "Not checked"}</span></div>
          <div class="health-row"><span>Version endpoint</span><span class="mono">${state.version ? "/version" : "Unavailable"}</span></div>
          <div class="health-row"><span>APNs certificate</span><span class="pill ${state.topic ? "pill-amber" : "pill-muted"}">${state.topic ? "Topic configured" : "Topic required"}</span></div>
        </div>
        <div class="card"><h3>Recent commands</h3><p class="subhead">Submission history from this browser session.</p>
          ${state.jobs.length ? `<div class="table-wrap"><table class="table"><tbody>${state.jobs.slice(0, 5).map(jobRow).join("")}</tbody></table></div>` : `<div class="empty"><strong>No commands yet</strong>Choose a device target to send a real MDM command.</div>`}
        </div>
      </div>${closeShell}`;
  }

  function jobRow(job) {
    return `<tr><td><strong>${escapeHTML(job.type)}</strong><br><span class="mono">${escapeHTML(job.id.slice(0, 8))}…</span></td><td>${escapeHTML(job.device.slice(0, 18))}…</td><td><span class="pill ${jobStatus(job) === "Accepted" ? "pill-green" : "pill-red"}">${escapeHTML(jobStatus(job))}</span></td></tr>`;
  }

  function renderDevices() {
    const devices = state.inventory || state.devices;
    const notice = state.inventory ? "Live enrollment inventory from the configured storage backend. Sensitive push tokens and identity certificates are never returned." : "This storage backend does not support enrollment inventory yet. Saved targets are local browser shortcuts for the real push and command APIs.";
    return shell("Devices", notice) +
      `<div class="notice ${state.inventory ? "" : "warning"}">${state.inventory ? "Device detail fields are limited to persisted enrollment metadata; Apple inventory values arrive only after a DeviceInformation command result." : "Use a SQL storage backend (MySQL or PostgreSQL) to enable live enrollment inventory."}</div>
      <div class="card" style="margin-top:16px"><div class="toolbar"><input class="input" id="device-search" placeholder="Search enrollment IDs, serials, or type" aria-label="Search devices"><input class="input" id="new-device" placeholder="Enrollment ID shortcut" aria-label="New enrollment ID"><button class="button button-primary" id="add-device">Add shortcut</button></div>
      <div id="device-table">${deviceTable(devices)}</div></div>${closeShell}`;
  }

  function deviceTable(devices) {
    if (!devices.length) return `<div class="empty"><strong>No saved targets</strong>Paste a device or user-channel enrollment ID above. It will be used as the path target for `/v1/push` and `/v1/enqueue`.</div>`;
    return `<div class="table-wrap"><table class="table"><thead><tr><th>Enrollment ID</th><th>Serial</th><th>Type</th><th>Last seen</th><th>Status</th><th>Actions</th></tr></thead><tbody>${devices.map((device) => `<tr><td class="mono">${escapeHTML(device.id)}</td><td class="mono">${escapeHTML(device.serial_number || "—")}</td><td>${escapeHTML(device.type || device.label || "Target")}</td><td>${escapeHTML(device.last_seen_at || "—")}</td><td><span class="pill ${device.enabled === false ? "pill-red" : "pill-green"}">${device.enabled === false ? "Disabled" : "Enabled"}</span></td><td><div class="action-list"><button class="button button-secondary device-action" data-action="info" data-id="${escapeHTML(device.id)}">Info</button><button class="button button-secondary device-action" data-action="push" data-id="${escapeHTML(device.id)}">Push</button><button class="button button-secondary device-action" data-action="restart" data-id="${escapeHTML(device.id)}">Restart</button><button class="button button-danger device-action" data-action="lock" data-id="${escapeHTML(device.id)}">Lock</button><button class="button button-danger device-action" data-action="wipe" data-id="${escapeHTML(device.id)}">Wipe</button></div></td></tr>`).join("")}</tbody></table></div>`;
  }

  function renderCommands() {
    return shell("Commands", "Every action below is submitted as an Apple MDM plist through NanoMDM’s raw enqueue API.") +
      `<div class="card"><div class="toolbar"><select class="select" id="command-device" aria-label="Command target"><option value="">Select a saved target</option>${state.devices.map((d) => `<option value="${escapeHTML(d.id)}">${escapeHTML(d.label || d.id)}</option>`).join("")}</select><select class="select" id="command-type" aria-label="Command type"><option value="DeviceInformation">Device information</option><option value="ProfileList">Profile list</option><option value="RestartDevice">Restart</option><option value="ShutDownDevice">Shut down</option><option value="DeviceLock">Lock</option><option value="EraseDevice">Wipe</option></select><button class="button button-primary" id="send-command">Send command</button></div>
      <p class="hint">Commands are queued and pushed by NanoMDM. The API response confirms enqueue/push processing, but this server does not expose a command-history read endpoint.</p>
      <div class="table-wrap" style="margin-top:16px"><table class="table"><thead><tr><th>Command</th><th>Target</th><th>Submitted</th><th>Status</th></tr></thead><tbody>${state.jobs.length ? state.jobs.map((job) => `<tr><td><strong>${escapeHTML(job.type)}</strong><br><span class="mono">${escapeHTML(job.id)}</span></td><td class="mono">${escapeHTML(job.device)}</td><td>${escapeHTML(job.time)}</td><td><span class="pill ${jobStatus(job) === "Accepted" ? "pill-green" : "pill-red"}">${escapeHTML(jobStatus(job))}</span></td></tr>`).join("") : `<tr><td colspan="4" class="empty">No commands submitted from this browser.</td></tr>`}</tbody></table></div></div>${closeShell}`;
  }

  function renderEnrollment() {
    return shell("Enrollment", "Prepare operators for enrollment without implying capabilities NanoMDM does not provide.") +
      `<div class="grid two-col"><div class="card"><h3>Generate enrollment profile</h3><p class="subhead">Create a configured starting profile from the repository template.</p><div class="form-grid"><div class="field field-wide"><label for="profile-mdm-url">Public MDM URL</label><input class="input" id="profile-mdm-url" placeholder="https://mdm.example.com/mdm"><small>Must be reachable by Apple devices over HTTPS.</small></div><div class="field"><label for="profile-topic">APNs push topic</label><input class="input" id="profile-topic" value="${escapeHTML(state.topic)}" placeholder="com.apple.mgmt.External.…"></div><div class="field"><label for="profile-scep-url">SCEP URL</label><input class="input" id="profile-scep-url" placeholder="https://scep.example.com/scep"><small>SCEP is an external service.</small></div></div><div class="section-actions"><button class="button button-primary" id="download-profile">Download .mobileconfig</button></div><p class="hint">This is a configured copy of <span class="mono">docs/enroll.mobileconfig</span>. Replace the SCEP challenge and sign/distribute it through your enrollment service.</p></div>
      <div class="card"><h3>Operator checklist</h3><p class="subhead">What is available in this repository today.</p><div class="health-row"><span>Enrollment migration</span><span class="pill pill-green">Optional /migration</span></div><div class="health-row"><span>Profile hosting</span><span class="pill pill-amber">External service</span></div><div class="health-row"><span>Users and roles</span><span class="pill pill-muted">Not supported</span></div><div class="health-row"><span>Device inventory API</span><span class="pill pill-green">SQL backends</span></div></div></div>${closeShell}`;
  }

  function renderSettings() {
    return shell("Settings", "Configure this browser session for the protected NanoMDM API.") +
      `<div class="card" style="max-width:720px"><div class="form-grid"><div class="field field-wide"><label for="api-key">API key</label><input id="api-key" class="input" type="password" value="${escapeHTML(state.apiKey)}" autocomplete="off"><small>Used as the password for Basic auth with the fixed NanoMDM username. It is held in session storage only.</small></div><div class="field field-wide"><label for="push-topic">APNs topic</label><input id="push-topic" class="input" value="${escapeHTML(state.topic)}" placeholder="com.apple.mgmt.External.…"><small>Optional. Used by the health check for <span class="mono">/v1/pushcert?topic=…</span>.</small></div></div><div class="section-actions"><button class="button button-primary" id="save-settings">Save and test connection</button></div></div>
      <div class="card" style="max-width:720px;margin-top:16px"><h3>Backend limitations</h3><p class="subhead">These are intentional product boundaries, not console errors.</p><p class="hint">The current API supports push notifications, raw plist command enqueueing, push certificate management, escrow key unlock, migration, and version reporting. It does not expose device listing/details, command history, profile management, logs, or user/role management.</p></div>${closeShell}`;
  }

  function commandXML(type) {
    const id = uuid();
    const commands = {
      DeviceInformation: "<key>Queries</key><array><string>DeviceName</string><string>OSVersion</string><string>SerialNumber</string><string>ProductName</string></array>",
      ProfileList: "",
      RestartDevice: "<key>RebuildKernelCache</key><true/>",
      ShutDownDevice: "",
      DeviceLock: "<key>PIN</key><string></string>",
      EraseDevice: "<key>PIN</key><string></string><key>PreserveDataPlan</key><false/>",
    };
    return { id, plist: `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Command</key><dict><key>RequestType</key><string>${type}</string>${commands[type] || ""}</dict><key>CommandUUID</key><string>${id}</string></dict></plist>` };
  }

  async function sendCommand(device, type) {
    if (!state.apiKey) { notify("Add the API key in Settings before sending commands.", "error"); location.hash = "settings"; return; }
    if (!device) { notify("Select or add a device target first.", "error"); return; }
    const command = commandXML(type);
    try {
      const result = await api(`/v1/enqueue/${encodeURIComponent(device)}`, { method: "PUT", headers: { "Content-Type": "text/plain" }, body: command.plist });
      state.jobs.unshift({ id: command.id, type, device, time: new Date().toLocaleString(), result, status: resultStatus(result) });
      save(); notify(`${type} queued for ${device.slice(0, 18)}…`, "success"); render();
    } catch (error) { state.jobs.unshift({ id: command.id, type, device, time: new Date().toLocaleString(), result: { enqueue_error: error.message }, status: "Failed" }); save(); notify(error.message, "error"); render(); }
  }

  async function sendPush(device) {
    try { await api(`/v1/push/${encodeURIComponent(device)}`); notify("Push notification sent.", "success"); } catch (error) { notify(error.message, "error"); }
  }

  function bindPage() {
    if (state.page === "devices") {
      $("#add-device").onclick = () => { const input = $("#new-device"); const id = input.value.trim(); if (!id) return notify("Enter an enrollment ID.", "error"); if (!state.devices.some((d) => d.id === id)) state.devices.push({ id, label: id }); input.value = ""; save(); render(); };
      $("#device-search").oninput = (event) => { const query = event.target.value.toLowerCase(); const devices = state.inventory || state.devices; $("#device-table").innerHTML = deviceTable(devices.filter((d) => `${d.id} ${d.label || ""} ${d.serial_number || ""} ${d.type || ""}`.toLowerCase().includes(query))); bindDeviceActions(); };
      bindDeviceActions();
    }
    if (state.page === "commands") $("#send-command").onclick = () => sendCommand($("#command-device").value, $("#command-type").value);
    if (state.page === "enrollment") $("#download-profile").onclick = downloadProfile;
    if (state.page === "settings") $("#save-settings").onclick = async () => { state.apiKey = $("#api-key").value.trim(); state.topic = $("#push-topic").value.trim(); sessionStorage.setItem("nanomdm.apiKey", state.apiKey); localStorage.setItem("nanomdm.topic", state.topic); await checkHealth(); render(); };
  }
  function downloadProfile() {
    const mdmURL = $("#profile-mdm-url").value.trim();
    const topic = $("#profile-topic").value.trim();
    const scepURL = $("#profile-scep-url").value.trim();
    if (!mdmURL || !topic || !scepURL) return notify("MDM URL, APNs topic, and SCEP URL are required.", "error");
    const profile = `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>PayloadContent</key><array><dict><key>PayloadContent</key><dict><key>Key Type</key><string>RSA</string><key>Challenge</key><string>SCEP-CHALLENGE-HERE</string><key>Keysize</key><integer>2048</integer><key>URL</key><string>${escapeHTML(scepURL)}</string></dict><key>PayloadType</key><string>com.apple.security.scep</string><key>PayloadUUID</key><string>${uuid()}</string><key>PayloadVersion</key><integer>1</integer></dict><dict><key>AccessRights</key><integer>8191</integer><key>IdentityCertificateUUID</key><string>CB90E976-AD44-4B69-8108-8095E6260978</string><key>PayloadType</key><string>com.apple.mdm</string><key>PayloadUUID</key><string>${uuid()}</string><key>PayloadVersion</key><integer>1</integer><key>ServerURL</key><string>${escapeHTML(mdmURL)}</string><key>SignMessage</key><true/><key>Topic</key><string>${escapeHTML(topic)}</string></dict></array><key>PayloadDisplayName</key><string>NanoMDM Enrollment Profile</string><key>PayloadType</key><string>Configuration</string><key>PayloadUUID</key><string>${uuid()}</string><key>PayloadVersion</key><integer>1</integer></dict></plist>`;
    const link = document.createElement("a");
    link.href = URL.createObjectURL(new Blob([profile], { type: "application/x-apple-aspen-config" }));
    link.download = "nanomdm-enrollment.mobileconfig";
    link.click();
    URL.revokeObjectURL(link.href);
    notify("Profile downloaded. Configure its SCEP challenge before distribution.", "success");
  }
  function bindDeviceActions() {
    document.querySelectorAll(".device-action").forEach((button) => button.onclick = () => { const action = button.dataset.action; const types = { info: "DeviceInformation", restart: "RestartDevice", lock: "DeviceLock", wipe: "EraseDevice" }; if (action === "push") return sendPush(button.dataset.id); sendCommand(button.dataset.id, types[action]); });
  }
  async function checkHealth() {
    try { const body = await api("/version"); state.version = body.version || "unknown"; setConnection(true, "Connected"); await loadEnrollments(); } catch (error) { setConnection(false, "Needs API key"); state.version = null; if (state.apiKey) notify(error.message, "error"); }
  }
  async function loadEnrollments() {
    try { const items = await api("/v1/enrollments?limit=500"); state.inventory = Array.isArray(items) ? items : []; }
    catch (error) { state.inventory = null; if (error.message && !error.message.includes("Not Implemented")) notify(`Live enrollment inventory unavailable: ${error.message}`, "error"); }
  }

  window.addEventListener("hashchange", () => { state.page = location.hash.slice(1) || "dashboard"; render(); });
  $("#refresh-button").onclick = async () => { await checkHealth(); render(); };
  $("#menu-toggle").onclick = () => $(".sidebar").classList.toggle("open");
  render();
  checkHealth().then(render);
})();
