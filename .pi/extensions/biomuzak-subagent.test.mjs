/**
 * biomuzak-subagent.test.mjs — unit tests for the pure logic in
 * biomuzak-subagent.ts (no subprocess, no model). Run: node biomuzak-subagent.test.mjs
 */
import assert from "node:assert/strict";
import path from "node:path";
import { createJiti } from "./node_modules/jiti/lib/jiti.mjs";

const jiti = createJiti(import.meta.url);
const mod = jiti(path.join(import.meta.dirname, "biomuzak-subagent.ts"));

const { extractAssistantText, JsonStreamParser, isModelResolutionError, attemptModels, isEmptyResult } = mod;

let passed = 0;
function test(name, fn) {
	fn();
	passed++;
	console.log(`ok - ${name}`);
}

test("extractAssistantText: plain string", () => {
	assert.equal(extractAssistantText("hello"), "hello");
});

test("extractAssistantText: text blocks, non-text dropped", () => {
	const out = extractAssistantText([
		{ type: "text", text: "a" },
		{ type: "thinking", text: "secret" },
		{ type: "text", text: "b" },
		null,
	]);
	assert.equal(out, "ab");
});

test("extractAssistantText: garbage input", () => {
	assert.equal(extractAssistantText(42), "");
	assert.equal(extractAssistantText(null), "");
});

test("JsonStreamParser: message_end assistant text", () => {
	const p = new JsonStreamParser();
	p.feedLine(JSON.stringify({ type: "message_end", message: { role: "assistant", content: [{ type: "text", text: "turn 1" }], stopReason: "stop" } }) + "\n");
	assert.equal(p.lastAssistantText, "turn 1");
	assert.equal(p.stopReason, "stop");
});

test("JsonStreamParser: replaced not appended; non-assistant ignored", () => {
	const p = new JsonStreamParser();
	p.feedLine(JSON.stringify({ type: "message_end", message: { role: "assistant", content: "one" } }));
	p.feedLine(JSON.stringify({ type: "message_end", message: { role: "toolResult", content: "noise" } }));
	p.feedLine(JSON.stringify({ type: "message_end", message: { role: "assistant", content: "two", errorMessage: "boom" } }));
	assert.equal(p.lastAssistantText, "two");
	assert.equal(p.errorMessage, "boom");
});

test("JsonStreamParser: torn/noise lines never throw", () => {
	const p = new JsonStreamParser();
	p.feedLine("{ not json");
	p.feedLine("");
	p.feedLine("diagnostic text on stdout");
	p.feedLine(JSON.stringify({ type: "other_event" }));
	assert.equal(p.lastAssistantText, null);
});

test("isModelResolutionError: known diagnostics", () => {
	assert.equal(isModelResolutionError("model qwen38 27B not found"), true);
	assert.equal(isModelResolutionError("Unknown model: foo"), true);
	assert.equal(isModelResolutionError("no models available"), true);
	assert.equal(isModelResolutionError("no model configured"), true);
});

test("isModelResolutionError: unrelated stderr is not a resolution error", () => {
	assert.equal(isModelResolutionError("review complete"), false);
	assert.equal(isModelResolutionError(""), false);
});

test("attemptModels: override wins", () => {
	assert.deepEqual(attemptModels(["a", "b"], "override"), ["override"]);
});

test("attemptModels: pinned order preserved", () => {
	assert.deepEqual(attemptModels(["a", "b"]), ["a", "b"]);
});

test("attemptModels: unpinned agent still gets one attempt on the session default", () => {
	assert.deepEqual(attemptModels([]), [undefined]);
});

test("isEmptyResult", () => {
	assert.equal(isEmptyResult("   \n\t "), true);
	assert.equal(isEmptyResult("CRITICAL: leak"), false);
});

test("default export registers run_agent with the four pinned agents", () => {
	const registered = [];
	mod.default({ registerTool: (t) => registered.push(t), on: () => {} });
	assert.equal(registered.length, 1);
	const tool = registered[0];
	assert.equal(tool.name, "run_agent");
	for (const name of ["Code Reviewer", "Performance Reviewer", "Risk Classifier", "Security Reviewer"]) {
		assert.ok(tool.description.includes(name), `description lists ${name}`);
	}
});

console.log(`\n${passed} tests passed`);
