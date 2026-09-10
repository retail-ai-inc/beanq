<template>
  <div class="home" ref="homeEle">
    <!-- Header -->
    <div class="row justify-content-between align-items-center mb-2">
      <div class="col-auto">
        <h5 class="mb-0 text-muted">Queue Overview</h5>
      </div>
      <div class="col-auto d-flex align-items-center gap-2">
        <span class="badge rounded-pill text-bg-success status-badge" v-if="connOk">
          <span class="status-dot"></span> Live
        </span>
        <span class="badge rounded-pill text-bg-secondary status-badge" v-else>
          <span class="status-dot bg-secondary"></span> Connecting…
        </span>
        <span class="text-muted small" v-if="lastUpdate">{{ lastUpdate }}</span>
        <select
          class="form-select form-select-sm"
          style="width: auto; min-width: 7rem"
          aria-label="Duration"
          v-model="execTime"
        >
          <option value="5m">5 minute</option>
          <option value="10m">10 minute</option>
          <option value="30m">30 minute</option>
          <option value="6h">6 hour</option>
          <option value="12h">12 hour</option>
          <option value="24h">24 hour</option>
        </select>
        <button class="btn btn-sm btn-outline-primary" type="button" @click="refreshTotals" :disabled="refreshing">
          {{ refreshing ? "…" : "Refresh" }}
        </button>
      </div>
    </div>

    <!-- Primary KPIs (minimal set) -->
    <div class="row g-3 mb-3">
      <div class="col-6 col-md-4 col-xl-2" v-for="kpi in primaryKpis" :key="kpi.key">
        <div class="card kpi-card h-100 border-0 shadow-sm" :class="kpi.tone">
          <div class="card-body py-3">
            <div class="kpi-label text-muted small">{{ kpi.label }}</div>
            <div class="kpi-value" :class="kpi.valueClass">
              <router-link v-if="kpi.to" :to="kpi.to" class="text-decoration-none kpi-link">
                {{ kpi.display }}
              </router-link>
              <span v-else>{{ kpi.display }}</span>
            </div>
            <div class="kpi-sub text-muted small" v-if="kpi.sub">{{ kpi.sub }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Main chart: produce / consume -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-body p-2">
        <div class="chart-h">
          <v-chart class="chart" ref="line1" :option="queuedMessagesOption" autoresize />
        </div>
      </div>
    </div>

    <!-- Queue status -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-body">
        <div class="d-flex flex-wrap justify-content-between align-items-center mb-3 gap-2">
          <h6 class="mb-0">Queue Status</h6>
          <div class="btn-group btn-group-sm" role="group">
            <button
              v-for="tab in typeTabs"
              :key="tab.value"
              type="button"
              class="btn"
              :class="typeFilter === tab.value ? 'btn-primary' : 'btn-outline-secondary'"
              @click="typeFilter = tab.value"
            >
              {{ tab.label }}
            </button>
          </div>
        </div>
        <div class="table-responsive">
          <table class="table table-hover table-sm align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Channel / Topic</th>
                <th>Type</th>
                <th>State</th>
                <th>Backlog</th>
                <th>Partitions</th>
                <th>Idle (s)</th>
                <th>Age Risk</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="filteredQueueRows.length === 0">
                <td colspan="8" class="text-muted text-center py-4">No queue data</td>
              </tr>
              <tr v-for="(row, idx) in filteredQueueRows" :key="idx">
                <td>
                  <strong>{{ row.channel }}</strong>
                  <span class="text-muted"> / {{ row.topic }}</span>
                </td>
                <td>
                  <span class="badge type-badge" :class="'type-' + row.mood">{{ row.moodLabel }}</span>
                </td>
                <td :class="row.state === 'Run' ? 'text-success' : 'text-danger'">{{ row.state }}</td>
                <td>
                  <span :class="row.size > 1000 ? 'text-danger fw-semibold' : row.size > 100 ? 'text-warning' : ''">
                    {{ formatNum(row.size) }}
                  </span>
                </td>
                <td>{{ row.partitions || "—" }}</td>
                <td>
                  <span :class="row.idle > 300 ? 'text-danger' : row.idle > 60 ? 'text-warning' : ''">
                    {{ row.idle }}
                  </span>
                </td>
                <td>
                  <span class="badge" :class="ageRiskClass(row)">{{ ageRiskLabel(row) }}</span>
                </td>
                <td class="text-end">
                  <router-link
                    class="btn btn-sm btn-outline-primary"
                    :to="'/admin/queue/detail/' + row.channel + ':' + row.topic"
                  >
                    Detail
                  </router-link>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="mt-2 text-end" v-if="queueFlat.length > 0">
          <router-link to="/admin/queue" class="small">View all queues →</router-link>
        </div>
      </div>
    </div>

    <!-- DLQ + health -->
    <div class="row g-3 mb-3">
      <div class="col-lg-6">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body">
            <div class="d-flex justify-content-between align-items-center mb-2">
              <h6 class="mb-0">Recent Dead Letters (DLQ)</h6>
              <router-link to="/admin/log/dlq" class="small">All →</router-link>
            </div>
            <div class="table-responsive">
              <table class="table table-sm align-middle mb-0">
                <thead class="table-light">
                  <tr>
                    <th>ID</th>
                    <th>Topic</th>
                    <th>Status</th>
                    <th>Time</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="dlqRows.length === 0">
                    <td colspan="4" class="text-muted text-center py-3">No dead letters</td>
                  </tr>
                  <tr v-for="(d, i) in dlqRows" :key="i">
                    <td><code class="small">{{ shortId(d.id) }}</code></td>
                    <td class="small">{{ d.topicName || d.topic || "—" }}</td>
                    <td><span class="badge text-bg-danger">{{ d.status || "failed" }}</span></td>
                    <td class="text-muted small">{{ formatTime(d.addTime || d.createdAt || d.time) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
      <div class="col-lg-6">
        <div class="card border-0 shadow-sm h-100">
          <div class="card-body">
            <h6 class="mb-2">System Health</h6>
            <table class="table table-sm align-middle mb-0">
              <thead class="table-light">
                <tr>
                  <th>Component</th>
                  <th>Status</th>
                  <th>Detail</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(h, i) in healthRows" :key="i">
                  <td>{{ h.name }}</td>
                  <td>
                    <span class="badge" :class="h.ok ? 'text-bg-success' : 'text-bg-warning'">
                      {{ h.ok ? "Healthy" : "Attention" }}
                    </span>
                  </td>
                  <td class="text-muted small">{{ h.detail }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Advanced metrics -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-body">
        <div class="d-flex flex-wrap justify-content-between align-items-center mb-3 gap-2">
          <h6 class="mb-0">
            Advanced Metrics
            <span class="badge text-bg-success ms-2">LIVE</span>
          </h6>
          <span class="text-muted small">Consumer groups · latency · sequence · delay</span>
        </div>
        <div class="row g-3 mb-3">
          <div class="col-4"><div class="card kpi-card"><div class="card-body py-2"><div class="small text-muted">Published · current minute</div><div class="kpi-value">{{ formatNum(liveMinuteMetrics.published) }}</div></div></div></div>
          <div class="col-4"><div class="card kpi-card"><div class="card-body py-2"><div class="small text-muted">Succeeded · current minute</div><div class="kpi-value text-success">{{ formatNum(liveMinuteMetrics.success) }}</div></div></div></div>
          <div class="col-4"><div class="card kpi-card"><div class="card-body py-2"><div class="small text-muted">Failed · current minute</div><div class="kpi-value" :class="liveMinuteMetrics.failed ? 'text-danger' : ''">{{ formatNum(liveMinuteMetrics.failed) }}</div></div></div></div>
        </div>
        <div class="row g-3 mb-3">
          <div class="col-6 col-md-4 col-xl-2" v-for="kpi in advancedMetricCards" :key="kpi.key">
            <div class="card kpi-card h-100 border-0 shadow-sm metric-card" :class="kpi.tone">
              <div class="card-body py-3">
                <div class="kpi-label text-muted small">{{ kpi.label }}</div>
                <div class="kpi-value" :class="kpi.valueClass">{{ kpi.display }}</div>
                <div class="kpi-sub text-muted small" v-if="kpi.sub">{{ kpi.sub }}</div>
              </div>
            </div>
          </div>
        </div>
        <div class="row g-3">
          <div class="col-lg-6">
            <h6 class="small text-muted mb-2">Latency P50 / P95 / P99</h6>
            <div class="chart-h-sm">
              <v-chart class="chart" :option="latencyOption" autoresize />
            </div>
          </div>
          <div class="col-lg-6">
            <h6 class="small text-muted mb-2">Consumer Groups</h6>
            <div class="table-responsive">
              <table class="table table-sm align-middle mb-0">
                <thead class="table-light">
                  <tr>
                    <th>Group</th>
                    <th>Members</th>
                    <th>Lag</th>
                    <th>Pending</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="metricsLoading"><td colspan="4" class="text-muted text-center">Loading…</td></tr>
                  <tr v-else-if="consumerGroups.length === 0"><td colspan="4" class="text-muted text-center py-4">No consumer group data in the selected window</td></tr>
                  <tr v-for="(r, i) in consumerGroups" :key="i">
                    <td class="small">{{ r.group }}</td>
                    <td>{{ r.members }}</td>
                    <td :class="r.lag > 1000 ? 'text-danger fw-semibold' : ''">{{ formatNum(r.lag) }}</td>
                    <td>{{ r.pending }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
        <div class="row g-3 mt-1">
          <div class="col-lg-6">
            <h6 class="small text-muted mb-2">Hot Sequence orderKeys</h6>
            <div class="table-responsive">
              <table class="table table-sm align-middle mb-0">
                <thead class="table-light">
                  <tr>
                    <th>orderKey</th>
                    <th>Queue</th>
                    <th>List Len</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="sequenceHotkeys.length === 0"><td colspan="4" class="text-muted text-center py-4">No sequence order keys with backlog</td></tr>
                  <tr v-for="(r, i) in sequenceHotkeys" :key="i">
                    <td><code class="small">{{ r.orderKey }}</code></td>
                    <td class="small"><strong>{{ r.channel }}</strong> / {{ r.topic }}</td>
                    <td :class="r.listLen > 50 ? 'text-danger fw-semibold' : ''">{{ r.listLen }}</td>
                    <td>
                      <span class="badge" :class="r.ok ? 'text-bg-success' : 'text-bg-warning'">
                        {{ r.ok ? "OK" : "Blocked" }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div class="col-lg-6">
            <h6 class="small text-muted mb-2">Delay Promote Lag</h6>
            <div class="table-responsive">
              <table class="table table-sm align-middle mb-0">
                <thead class="table-light">
                  <tr>
                    <th>Channel / Topic</th>
                    <th>Due</th>
                    <th>Promote lag</th>
                    <th>P99</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="delayPromoteRows.length === 0"><td colspan="4" class="text-muted text-center py-4">No delay queues in the selected window</td></tr>
                  <tr v-for="(r, i) in delayPromoteRows" :key="i">
                    <td class="small"><strong>{{ r.channel }}</strong> / {{ r.topic }}</td>
                    <td>{{ r.dueWaiting }}</td>
                    <td :class="r.promoteLagMs > 5000 ? 'text-danger' : ''">{{ r.promoteLagMs }} ms</td>
                    <td>{{ r.p99LagMs }} ms</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Nodes / infrastructure (Pod CPU & memory) — collapsed -->
    <div class="card border-0 shadow-sm mb-3">
      <div class="card-header bg-white border-0 d-flex justify-content-between align-items-center py-2">
        <button
          class="btn btn-link text-decoration-none text-dark p-0 fw-semibold"
          type="button"
          @click="showNodes = !showNodes"
        >
          <span class="me-1">{{ showNodes ? "▾" : "▸" }}</span>
          Nodes / Infrastructure
        </button>
        <span class="text-muted small">
          {{ pods.length }} host(s) · CPU / Memory
        </span>
      </div>
      <div class="card-body pt-0" v-show="showNodes">
        <div v-if="pods.length === 0" class="text-muted small py-3 text-center">
          No pod heartbeat
        </div>
        <div v-for="(item, index) in pods" :key="index" class="mb-3" :class="{ 'mb-0': index === pods.length - 1 }">
          <div class="fw-semibold mb-2">{{ item.hostName }}</div>
          <table class="table table-sm mb-0">
            <thead class="table-light">
              <tr>
                <th>Cpu Total</th>
                <th>Cpu Percent</th>
                <th>Memory Total</th>
                <th>Memory Percent</th>
                <th>Memory Used</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>{{ item.cpuCount }}</td>
                <td>{{ item.cpuPercent }}(%)</td>
                <td>{{ item.memoryTotal }}(GB)</td>
                <td>{{ item.memoryPercent }}(%)</td>
                <td>{{ item.memoryUsed }}(MB)</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <LoginModal :id="loginId" ref="loginModal" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useRouter } from "vueRouter";
import LoginModal from "./components/loginModal.vue";

const [line1, useR, homeEle] = [ref(null), useRouter(), ref(null)];
const [loginId, loginModal] = [ref("staticBackdrop"), ref("loginModal")];
const [execTime, sseUrl] = [ref("5m"), ref("")];

let [queuedMessagesOption, nodeId, sse, resizeObserver, pods] = [
  ref({}),
  ref(""),
  ref(null),
  null,
  ref([]),
];

const [queues, queuesCount] = [ref([]), ref(0)];
const connOk = ref(false);
const lastUpdate = ref("");
const refreshing = ref(false);
const showNodes = ref(false);

const totals = ref({
  queue_total: 0,
  fail_count: 0,
  success_count: 0,
  db_size: 0,
  num_cpu: 0,
});
const queueFlat = ref([]);
const scheduleRows = ref([]);
const dlqRows = ref([]);
const typeFilter = ref("all");
const typeTabs = [
  { value: "all", label: "All" },
  { value: "normal", label: "Normal" },
  { value: "delay", label: "Delay" },
  { value: "sequence", label: "Sequence" },
];

const liveReady = ref(0);
const livePending = ref(0);
const liveTotal = ref(0);

const prevFailCount = ref(null);
const prevFailAt = ref(null);
const dlqRatePerMin = ref(0);

// Mock advanced metrics (not exposed by current APIs)
// Advanced metrics are not available from the current backend contract.
// Keep the section disabled until the real metrics endpoint is deployed;
// never present randomized values as production data.
const advancedMetrics = ref({ e2eLatencyP50Sec: null, e2eLatencyP95Sec: null, e2eLatencyP99Sec: null, lagTotal: null, oldestAgeSec: null, retryRate: null });
const sequenceHotkeys = ref([]);
const consumerGroups = ref([]);
const metricsLoading = ref(false);
const liveMinuteMetrics = ref({ published: 0, success: 0, failed: 0 });
const delayPromoteRows = ref([]);
const latencyOption = ref({});

const successRate = computed(() => {
  const s = Number(totals.value.success_count) || 0;
  const f = Number(totals.value.fail_count) || 0;
  const t = s + f;
  if (t === 0) return "—";
  return ((s / t) * 100).toFixed(1) + "%";
});

const totalBacklog = computed(() =>
  queueFlat.value.reduce((sum, r) => sum + (Number(r.size) || 0), 0)
);

const scheduledTotal = computed(() =>
  scheduleRows.value.reduce((sum, r) => sum + (Number(r.size) || 0), 0)
);

const maxIdle = computed(() => {
  if (!queueFlat.value.length) return 0;
  return Math.max(...queueFlat.value.map((r) => Number(r.idle) || 0));
});

/** Single primary KPI row — no duplicate success/backlog cards */
const primaryKpis = computed(() => {
  const pending = livePending.value;
  const ready = liveReady.value || totalBacklog.value;
  const consumerCount = pods.value?.length || 0;
  const fail = Number(totals.value.fail_count) || 0;
  const sched = scheduledTotal.value;
  const rate = successRate.value;

  return [
    {
      key: "pending",
      label: "Pending",
      display: formatNum(pending),
      sub: "Unacked (SSE)",
      tone: pending > 1000 ? "kpi-danger" : pending > 100 ? "kpi-warning" : "kpi-primary",
      valueClass: pending > 1000 ? "text-danger" : "",
    },
    {
      key: "backlog",
      label: "Backlog",
      display: formatNum(ready),
      sub: "Ready / XLEN",
      to: "/admin/queue",
      tone: ready > 5000 ? "kpi-danger" : ready > 500 ? "kpi-warning" : "kpi-info",
      valueClass: ready > 5000 ? "text-danger" : "",
    },
    {
      key: "consumers",
      label: "Consumers",
      display: String(consumerCount),
      sub: consumerCount === 0 ? "None online" : "Pods reporting",
      tone: consumerCount === 0 ? "kpi-danger" : "kpi-secondary",
      valueClass: consumerCount === 0 ? "text-danger" : "",
    },
    {
      key: "successRate",
      label: "Success Rate",
      display: rate,
      sub: "success / (success+fail)",
      to: "/admin/log/event?status=success",
      tone: "kpi-success",
    },
    {
      key: "failed",
      label: "Failed / DLQ",
      display: formatNum(fail),
      sub: dlqRatePerMin.value > 0 ? `~${dlqRatePerMin.value.toFixed(1)}/min` : "Mongo failed",
      to: "/admin/log/event?status=failed",
      tone: fail > 0 ? "kpi-danger" : "kpi-secondary",
      valueClass: fail > 0 ? "text-danger" : "",
    },
    {
      key: "scheduled",
      label: "Scheduled",
      display: formatNum(sched),
      sub: "Delay streams",
      to: "/admin/schedule",
      tone: "kpi-warning",
    },
  ];
});

const advancedMetricCards = computed(() => {
  const m = advancedMetrics.value;
  return [
    {
      key: "e2eP99",
      label: "E2E Latency P99",
      display: m.e2eLatencyP99Sec == null ? "—" : m.e2eLatencyP99Sec + " s",
      sub: `P50 ${m.e2eLatencyP50Sec} · P95 ${m.e2eLatencyP95Sec}`,
      tone: m.e2eLatencyP99Sec > 1 ? "kpi-danger" : m.e2eLatencyP99Sec > 0.3 ? "kpi-warning" : "kpi-info",
      valueClass: m.e2eLatencyP99Sec > 1 ? "text-danger" : "",
    },
    {
      key: "lag",
      label: "Consumer Lag",
      display: m.lagTotal == null ? "—" : formatNum(m.lagTotal),
      sub: "latest − last ACK",
      tone: m.lagTotal > 10000 ? "kpi-danger" : m.lagTotal > 1000 ? "kpi-warning" : "kpi-success",
    },
    {
      key: "oldest",
      label: "Oldest Msg Age",
      display: m.oldestAgeSec == null ? "—" : m.oldestAgeSec + " s",
      sub: "max unacked age",
      tone: m.oldestAgeSec > 300 ? "kpi-danger" : m.oldestAgeSec > 60 ? "kpi-warning" : "kpi-secondary",
    },
    {
      key: "retry",
      label: "Retry Rate",
      display: m.retryRate == null ? "—" : m.retryRate + "/min",
      sub: "re-delivery",
      tone: m.retryRate > 50 ? "kpi-danger" : "kpi-warning",
    },
  ];
});

const filteredQueueRows = computed(() => {
  let rows =
    typeFilter.value === "all"
      ? queueFlat.value
      : queueFlat.value.filter((r) => r.mood === typeFilter.value);
  return [...rows].sort((a, b) => (b.size || 0) - (a.size || 0));
});

const healthRows = computed(() => {
  const podN = pods.value?.length || 0;
  const pendingHigh = livePending.value > 1000;
  const idleHigh = maxIdle.value > 300;
  const fail = Number(totals.value.fail_count) || 0;
  console.log(connOk.value)
  return [
    {
      name: "Dashboard SSE",
      ok: connOk.value,
      detail: connOk.value ? `stream · ${execTime.value}` : "Reconnecting",
    },
    {
      name: "Consumers",
      ok: podN > 0,
      detail: podN > 0 ? `${podN} instance(s)` : "No pod heartbeat",
    },
    {
      name: "Pending",
      ok: !pendingHigh,
      detail: `Pending=${formatNum(livePending.value)}`,
    },
    {
      name: "Message Age",
      ok: !idleHigh,
      detail: `Max idle ${maxIdle.value}s`,
    },
    {
      name: "DLQ",
      ok: fail < 100,
      detail:
        fail > 0
          ? `fail=${formatNum(fail)} · ~${dlqRatePerMin.value.toFixed(1)}/min`
          : "No significant failures",
    },
  ];
});

function formatNum(n) {
  const v = Number(n);
  if (!Number.isFinite(v)) return "—";
  return v.toLocaleString();
}

function shortId(id) {
  if (!id) return "—";
  const s = String(id);
  return s.length > 12 ? s.slice(0, 8) + "…" : s;
}

function formatTime(t) {
  if (!t) return "—";
  try {
    const d = new Date(t);
    if (Number.isNaN(d.getTime())) return String(t);
    return d.toLocaleString();
  } catch {
    return String(t);
  }
}

function ageRiskClass(row) {
  const idle = Number(row.idle) || 0;
  const size = Number(row.size) || 0;
  if (size > 0 && idle > 300) return "text-bg-danger";
  if (size > 0 && idle > 60) return "text-bg-warning";
  if (size > 1000) return "text-bg-warning";
  return "text-bg-success";
}

function ageRiskLabel(row) {
  const idle = Number(row.idle) || 0;
  const size = Number(row.size) || 0;
  if (size > 0 && idle > 300) return "Stale";
  if (size > 0 && idle > 60) return "Aging";
  if (size > 1000) return "Heavy";
  return "OK";
}

function normalizeMood(raw) {
  const s = String(raw || "")
    .toLowerCase()
    .replace(/_stream$/, "")
    .replace(/-queue$/, "");
  if (s.includes("delay")) return "delay";
  if (s.includes("sequence") || s.includes("sequential")) return "sequence";
  return "normal";
}

function moodLabel(m) {
  if (m === "delay") return "Delay";
  if (m === "sequence") return "Sequence";
  return "Normal";
}

function buildProduceConsumeTrendOption(values, durationLabel) {
  const list = (Array.isArray(values) ? values : []).slice(-1000);
  const xdata = [];
  const produce = [];
  const consume = [];

  list.forEach((val) => {
    // Dashboard stream v2 uses named fields. Accept v1 arrays temporarily so
    // rolling upgrades do not blank the chart.
    const point = Array.isArray(val)
      ? { streamLength: val[0], pending: val[1], ready: val[2], timestamp: val[3] }
      : val;
    if (!point || typeof point !== "object") return;
    const pending = Number(point.pending) || 0;
    const ready = Number(point.ready) || 0;
    const total = Number(point.streamLength ?? (ready + pending)) || 0;
    produce.push(point.produced != null ? Number(point.produced) : total);
    consume.push(point.consumed != null ? Number(point.consumed) : ready);
    const ts = Number(point.timestamp);
    const parsedTs = Number.isFinite(ts) ? ts : Date.parse(String(point.timestamp || ""));
    xdata.push(Number.isFinite(parsedTs) ? new Date(parsedTs < 1e12 ? parsedTs * 1000 : parsedTs).toLocaleString() : "");
    liveReady.value = ready;
    livePending.value = pending;
    liveTotal.value = total;
  });

  const formatY = (v) => {
    const n = Number(v);
    if (!Number.isFinite(n)) return v;
    if (n >= 1e9) return (n / 1e9).toFixed(1) + "B";
    if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
    if (n >= 1e3) return (n / 1e3).toFixed(1) + "K";
    return String(Math.round(n));
  };

  return {
    color: ["#3b82f6", "#22c55e"],
    title: {
      text: "Message Produce & Consume Trend",
      left: 12,
      top: 8,
      textStyle: { fontSize: 14, fontWeight: 600, color: "#1e293b" },
      subtext: durationLabel ? `(${durationLabel})` : "",
      subtextStyle: { fontSize: 11, color: "#94a3b8" },
    },
    tooltip: {
      trigger: "axis",
      backgroundColor: "rgba(255,255,255,0.96)",
      borderColor: "#e2e8f0",
      borderWidth: 1,
      textStyle: { color: "#334155", fontSize: 12 },
      axisPointer: {
        type: "line",
        lineStyle: { color: "#cbd5e1", width: 1, type: "dashed" },
      },
      formatter(params) {
        if (!params || !params.length) return "";
        let html = `<div style="font-weight:600;margin-bottom:4px">${params[0].axisValueLabel}</div>`;
        params.forEach((p) => {
          html += `<div style="display:flex;align-items:center;gap:6px;margin:2px 0">
            <span style="width:8px;height:8px;border-radius:50%;background:${p.color};display:inline-block"></span>
            <span>${p.seriesName}</span>
            <span style="margin-left:auto;font-weight:600">${formatY(p.value)}</span>
          </div>`;
        });
        return html;
      },
    },
    legend: {
      data: [
        { name: "Produced", icon: "circle" },
        { name: "Consumed", icon: "circle" },
      ],
      right: 16,
      top: 10,
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 16,
      textStyle: { color: "#64748b", fontSize: 12 },
    },
    grid: { top: 52, left: 56, right: 20, bottom: 36, containLabel: true },
    xAxis: {
      type: "category",
      boundaryGap: false,
      name: "Time",
      nameLocation: "middle",
      nameGap: 24,
      data: xdata.length ? xdata : ["No samples"],
      axisLine: { show: true, lineStyle: { color: "#cbd5e1" } },
      axisTick: { show: true },
      axisLabel: { color: "#94a3b8", fontSize: 11, hideOverlap: true },
      splitLine: { show: false },
    },
    yAxis: {
      type: "value",
      name: "Messages",
      nameLocation: "middle",
      nameGap: 42,
      axisLine: { show: true, lineStyle: { color: "#cbd5e1" } },
      axisTick: { show: true },
      axisLabel: { color: "#94a3b8", fontSize: 11, formatter: formatY },
      splitLine: { lineStyle: { color: "#f1f5f9", type: "solid" } },
    },
    dataZoom: [{ type: "inside", start: 0, end: 100, zoomOnMouseWheel: true, moveOnMouseMove: true }],
    series: [
      {
        name: "Produced",
        type: "line",
        smooth: true,
        symbol: "circle",
        symbolSize: 5,
        sampling: "lttb",
        lineStyle: { width: 2, color: "#3b82f6" },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: "rgba(59, 130, 246, 0.28)" },
            { offset: 1, color: "rgba(59, 130, 246, 0.02)" },
          ]),
        },
        data: produce.length ? produce : [null],
      },
      {
        name: "Consumed",
        type: "line",
        smooth: true,
        symbol: "circle",
        symbolSize: 6,
        z: 3,
        sampling: "lttb",
        lineStyle: { width: 2.5, color: "#16a34a", type: "dashed" },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: "rgba(34, 197, 94, 0.22)" },
            { offset: 1, color: "rgba(34, 197, 94, 0.02)" },
          ]),
        },
        data: consume.length ? consume : [null],
      },
    ],
  };
}

function buildMockLatencyOption() {
  const m = advancedMetrics.value;
  return {
    color: ["#3b82f6", "#f59e0b", "#ef4444"],
    tooltip: { trigger: "axis" },
    legend: {
      data: ["P50", "P95", "P99"],
      right: 8,
      top: 0,
      textStyle: { fontSize: 11, color: "#64748b" },
    },
    grid: { top: 28, left: 8, right: 8, bottom: 4, containLabel: true },
    xAxis: {
      type: "category",
      data: ["-25m", "-20m", "-15m", "-10m", "-5m", "now"],
      axisLabel: { color: "#94a3b8", fontSize: 10 },
      axisLine: { lineStyle: { color: "#e2e8f0" } },
      axisTick: { show: false },
    },
    yAxis: {
      type: "value",
      axisLabel: { color: "#94a3b8", fontSize: 10, formatter: (v) => v + "s" },
      splitLine: { lineStyle: { color: "#f1f5f9" } },
      axisLine: { show: false },
      axisTick: { show: false },
    },
    series: [
      { name: "P50", type: "line", smooth: true, symbol: "none", data: [0.038, 0.040, 0.036, 0.044, 0.041, m.e2eLatencyP50Sec] },
      { name: "P95", type: "line", smooth: true, symbol: "none", data: [0.160, 0.175, 0.150, 0.190, 0.170, m.e2eLatencyP95Sec] },
      { name: "P99", type: "line", smooth: true, symbol: "none", data: [0.580, 0.610, 0.540, 0.700, 0.640, m.e2eLatencyP99Sec] },
    ],
  };
}


function flattenQueues(data) {
  const rows = [];
  if (!data || typeof data !== "object") return rows;
  const pushRow = (d, channelFallback) => {
    const mood = normalizeMood(d.moodType || d.mood || d.type);
    rows.push({
      channel: d.channel || channelFallback || "—",
      topic: d.topic || "—",
      state: d.state || "—",
      size: d.size ?? 0,
      idle: d.idle ?? 0,
      partitions: d.partitions ?? 0,
      mood,
      moodLabel: moodLabel(mood),
    });
  };
  if (Array.isArray(data)) {
    data.forEach((d) => pushRow(d));
    return rows;
  }
  Object.keys(data).forEach((channel) => {
    const list = data[channel];
    if (!Array.isArray(list)) return;
    list.forEach((d) => pushRow(d, channel));
  });
  return rows;
}

function flattenSchedule(data) {
  const rows = [];
  if (!data || typeof data !== "object") return rows;
  if (Array.isArray(data)) {
    data.forEach((d) => {
      rows.push({
        channel: d.channel || "—",
        topic: d.topic || "—",
        size: d.size ?? 0,
        state: d.state || "—",
      });
    });
    return rows;
  }
  Object.keys(data).forEach((channel) => {
    const list = data[channel];
    if (!Array.isArray(list)) return;
    list.forEach((d) => {
      rows.push({
        channel: d.channel || channel,
        topic: d.topic || "—",
        size: d.size ?? 0,
        state: d.state || "—",
      });
    });
  });
  return rows;
}

function updateDlqRate(failCount) {
  const now = Date.now();
  const cur = Number(failCount) || 0;
  if (prevFailCount.value != null && prevFailAt.value != null) {
    const dtMin = Math.max((now - prevFailAt.value) / 60000, 1 / 60);
    const delta = Math.max(cur - prevFailCount.value, 0);
    dlqRatePerMin.value = delta / dtMin;
  }
  prevFailCount.value = cur;
  prevFailAt.value = now;
}

async function loadTotals() {
  try {
    const res = await dashboardApi.Total();
    totals.value = {
      queue_total: res?.queue_total ?? 0,
      fail_count: res?.fail_count ?? 0,
      success_count: res?.success_count ?? 0,
      db_size: res?.db_size ?? 0,
      num_cpu: res?.num_cpu ?? 0,
    };
    updateDlqRate(totals.value.fail_count);
  } catch (e) {
    if (e?.status === 401) loginModal.value?.error?.(new Error(e));
  }
}

async function loadQueues() {
  try {
    const res = await request.get("queues", { params: { page: 1, pageSize: 200 } });
    queueFlat.value = flattenQueues(res ?? {});
  } catch (e) {
    if (e?.status === 401) loginModal.value?.error?.(new Error(e));
  }
}

async function loadSchedules() {
  try {
    const res = await scheduleApi.GetSchedule(1, 100);
    const data = res?.data ?? res ?? {};
    scheduleRows.value = flattenSchedule(data);
  } catch (e) {
    scheduleRows.value = [];
  }
}

async function loadDlq() {
  try {
    const res = await dlqApi.List(1, 8, "", "", "", "");
    const list = Array.isArray(res)
      ? res
      : res?.data || res?.list || res?.items || res?.records || [];
    dlqRows.value = Array.isArray(list) ? list.slice(0, 8) : [];
  } catch (e) {
    dlqRows.value = [];
  }
}

async function loadMetrics() {
  metricsLoading.value = true;
  try {
    const res = await request.get("dashboard/metrics");
    const groups = res?.consumerGroups || res?.data?.consumerGroups || [];
    const minute = res?.minute || res?.data?.minute || {};
    const metricsData = res?.data || res || {};
    advancedMetrics.value.lagTotal = metricsData.lagTotal == null ? null : Number(metricsData.lagTotal);
    const latency = metricsData.latency || {};
    advancedMetrics.value.e2eLatencyP50Sec = latency.p50 ?? null;
    advancedMetrics.value.e2eLatencyP95Sec = latency.p95 ?? null;
    advancedMetrics.value.e2eLatencyP99Sec = latency.p99 ?? null;
    const hasLatencySamples = [latency.p50, latency.p95, latency.p99].some((value) => value != null && Number.isFinite(Number(value)));
    latencyOption.value = {
      title: hasLatencySamples
        ? { show: false }
        : { text: "No latency samples in the selected window", left: "center", top: "middle", textStyle: { fontSize: 12, color: "#94a3b8", fontWeight: "normal" } },
      tooltip: { trigger: "axis" },
      legend: { data: ["P50", "P95", "P99"], top: 0 },
      grid: { top: 36, left: 48, right: 16, bottom: 28, containLabel: true },
      xAxis: { type: "category", name: "", nameLocation: "middle", nameGap: 24, data: ["last 15m"], axisLine: { show: true, lineStyle: { color: "#cbd5e1" } }, axisTick: { show: true }, axisLabel: { color: "#94a3b8" } },
      yAxis: { type: "value", name: "Latency (s)", nameLocation: "middle", nameGap: 42, min: 0, axisLine: { show: true, lineStyle: { color: "#cbd5e1" } }, axisTick: { show: true }, axisLabel: { formatter: (v) => v + " s" }, splitLine: { show: true, lineStyle: { color: "#f1f5f9" } } },
      series: [
        { name: "P50", type: "bar", data: [latency.p50 == null ? null : latency.p50] },
        { name: "P95", type: "bar", data: [latency.p95 == null ? null : latency.p95] },
        { name: "P99", type: "bar", data: [latency.p99 == null ? null : latency.p99] },
      ],
    };
    delayPromoteRows.value = (metricsData.delayQueues || []).map((row) => ({
      channel: row.channel, topic: row.topic, dueWaiting: row.dueWaiting, promoteLagMs: null, p99LagMs: null,
    }));
    sequenceHotkeys.value = metricsData.sequenceHotKeys || [];
    liveMinuteMetrics.value = {
      published: Number(minute.published) || 0,
      success: Number(minute.success) || 0,
      failed: Number(minute.failed) || 0,
    };
    consumerGroups.value = groups.map((g) => ({
      group: g.group || "—",
      members: Number(g.members) || 0,
      lag: g.lag == null ? 0 : Number(g.lag),
      pending: g.pending == null ? "—" : Number(g.pending),
    }));
  } catch (e) {
    consumerGroups.value = [];
    liveMinuteMetrics.value = { published: 0, success: 0, failed: 0 };
  } finally {
    metricsLoading.value = false;
  }
}

async function refreshTotals() {
  refreshing.value = true;
  try {
    await Promise.all([loadTotals(), loadQueues(), loadSchedules(), loadDlq(), loadMetrics()]);
    lastUpdate.value = "Updated " + new Date().toLocaleTimeString();
  } finally {
    refreshing.value = false;
  }
}

function resize() {
  line1.value?.resize();
}

watch(
  () => execTime.value,
  (n) => {
    execTime.value = n;
    queues.value = [];
    sseConnect();
  }
);

function sseConnect() {
  if (sse.value) sse.value.close();
  // Each SSE session is a fresh snapshot for the selected duration. Do not
  // append points from a previous session to the new chart.
  queues.value = [];
  liveReady.value = 0;
  livePending.value = 0;
  liveTotal.value = 0;
  sseUrl.value = `dashboard/stream?duration=${execTime.value}`;
  sse.value = sseApi.Init(sseUrl.value);
  sse.value.onopen = () => {
    connOk.value = true;
  };
  sse.value.addEventListener("dashboard", function (res) {
    // Some browsers do not reliably fire EventSource.onopen when the
    // authenticated stream immediately emits its first event. Receiving a
    // dashboard event is definitive proof that the SSE connection is healthy.
    connOk.value = true;
    const { code, msg, data } = JSON.parse(res.data);
    if (code === "1004") {
      loginModal.value.error(new Error(msg));
      sse.value.close();
      connOk.value = false;
      return;
    }
    if (code === "1111" && msg === "DONE") {
      // The stream sends the historical snapshot in the DONE event. Keep it
      // in the same buffer as incremental samples before rendering; otherwise
      // the chart is rebuilt from an empty array even though SSE has data.
      if (Array.isArray(data)) queues.value.push(...data);
      else if (data && typeof data === "object") queues.value.push(data);
      if (queues.value.length > 1000) {
        queues.value.splice(0, queues.value.length - 1000);
      }
      queuesCount.value = queues.value.length;
      sse.value.close();
      queuedMessagesOption.value = buildProduceConsumeTrendOption(queues.value, execTime.value);
      lastUpdate.value = "Updated " + new Date().toLocaleTimeString();
      setTimeout(sseConnect, 3000);
      return;
    }
    if (data == null) {
      queuedMessagesOption.value = buildProduceConsumeTrendOption(queues.value, execTime.value);
    } else {
      if (Array.isArray(data)) queues.value.push(...data);
      else if (typeof data === "object") queues.value.push(data);
      // Keep only the most recent 1000 samples in the chart window.
      if (queues.value.length > 1000) {
        queues.value.splice(0, queues.value.length - 1000);
      }
      queuedMessagesOption.value = buildProduceConsumeTrendOption(queues.value, execTime.value);
    }
  });
  sse.value.onerror = (err) => {
    console.log(err);
    connOk.value = false;
    sse.value.close();
    setTimeout(sseConnect, 1500);
  };
}

let [ssePod] = [ref(null)];
function getPods() {
  if (ssePod.value) ssePod.value.close();
  ssePod.value = sseApi.Init(`dashboard/pods/stream`);
  ssePod.value.onopen = () => {
    console.log("pods connect success");
  };
  ssePod.value.addEventListener("pods", function (res) {
    const { code, msg, data } = JSON.parse(res.data);
    if (code === "1004") {
      loginModal.value.error(new Error(msg));
      sse.value?.close();
      return;
    }
    pods.value = data
      .map((item) => {
        try {
          return JSON.parse(item);
        } catch (e) {
          return undefined;
        }
      })
      .filter(
        (item) =>
          item !== undefined && item !== null && item !== "" && Object.keys(item).length !== 0
      );
  });
  ssePod.value.onerror = (err) => {
    console.log(err);
    ssePod.value.close();
    setTimeout(getPods, 1500);
  };
}

onMounted(() => {
  queuedMessagesOption.value = buildProduceConsumeTrendOption([], execTime.value);
  latencyOption.value = {};

  const observerFun = () => {
    resize();
    return false;
  };
  resizeObserver = new ResizeObserver(() => {
    Base.Debounce(observerFun(), 5000);
  });
  const parentEle = homeEle.value.parentElement;
  resizeObserver.observe(parentEle);
  getPods();
  sseConnect();
  refreshTotals();
});

onUnmounted(() => {
  if (sse.value) sse.value.close();
  if (ssePod.value) ssePod.value.close();
  if (resizeObserver) resizeObserver.disconnect();
});
</script>

<style scoped>
.home {
  transition: opacity 0.5s ease;
  opacity: 1;
}

.chart-h {
  height: 300px;
  background: #fff;
  border-radius: 0.75rem;
}

.chart-h-sm {
  height: 220px;
}

.chart {
  background-color: #ffffff;
  box-sizing: border-box;
  height: 100%;
  width: 100%;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  display: inline-block;
}

.kpi-card {
  border-radius: 0.75rem;
  background: #fff;
}

.kpi-label {
  margin-bottom: 0.25rem;
}

.kpi-value {
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.2;
}

.kpi-link {
  color: inherit;
}

.kpi-link:hover {
  text-decoration: underline !important;
}

.kpi-primary {
  border-left: 4px solid #0d6efd !important;
}
.kpi-success {
  border-left: 4px solid #198754 !important;
}
.kpi-danger {
  border-left: 4px solid #dc3545 !important;
}
.kpi-info {
  border-left: 4px solid #0dcaf0 !important;
}
.kpi-warning {
  border-left: 4px solid #ffc107 !important;
}
.kpi-secondary {
  border-left: 4px solid #6c757d !important;
}

.type-badge {
  font-weight: 600;
  font-size: 0.7rem;
}
.type-normal {
  background: rgba(13, 110, 253, 0.15);
  color: #0d6efd;
}
.type-delay {
  background: rgba(255, 193, 7, 0.2);
  color: #997404;
}
.type-sequence {
  background: rgba(111, 66, 193, 0.15);
  color: #6f42c1;
}

.metric-card {
  border-style: dashed !important;
  opacity: 0.95;
}

.card-header .btn-link:hover {
  color: #0d6efd !important;
}
</style>
