const test = require("node:test");
const assert = require("node:assert/strict");

const created = [];
global.EventSource = class EventSource {
	constructor(url, options) {
		this.url = url;
		this.options = options;
		created.push(this);
	}
};

const sseApi = require("./sse.js");

test("Init rejects missing URLs without creating a connection", () => {
	assert.throws(() => sseApi.Init(), /non-empty string/);
	assert.throws(() => sseApi.Init(""), /non-empty string/);
	assert.equal(created.length, 0);
});

test("Init preserves the supplied endpoint for native reconnection", () => {
	const source = sseApi.Init("events?page=1&pageSize=10");
	assert.equal(source.url, "/api/v1/events?page=1&pageSize=10");
	assert.deepEqual(source.options, {withCredentials: true});
	assert.equal(created.length, 1);
});
