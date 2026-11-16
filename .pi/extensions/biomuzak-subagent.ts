/**
 * biomuzak-subagent.ts — run a pinned agent (`.github/agents/*.agent.md`) as a
 * fresh-context subagent.
 *
 * Registers a single `run_agent` tool. Each call spawns a read-only pi
 * subprocess with a pinned model, the agent's full prompt, and only the
 * read tools — so "never modify code" is enforced by the tool set, not
 * just the prompt.
 *
 * The subprocess runs ASYNCHRONOUSLY: the TUI stays responsive while the
 * reviewer works, Esc cancels the run (SIGTERM, then SIGKILL after a grace
 * period), and the reviewer's latest assistant text is streamed into the
 * tool result as it finishes each turn. The old `execFileSync` version
 * blocked the event loop for the whole run — that freeze is what the
 * 2026-11-13 /pre-pr-review aborts were.
 *
 * Model pinning: the agent frontmatter pins models as pi model patterns
 * (primary first, fallbacks after). Each is passed to `pi --model` in
 * order; if a pattern does not resolve, the next is tried, and if none
 * does the subprocess falls back to the session default model.
 *
 * The reviewer subprocess is a full pi session, so it sees the repo's
 * AGENTS.md (symlinked to .github/copilot-instructions.md) and the
 * .github/instructions/*.instructions.md files; the agent file's body is
 * used as its system prompt via --system-prompt. A short runtime note
 * translates the Copilot tool names onto pi's.
 *
 * This is a pi-specific wiring: the same agent files are the single source
 * of truth for both harnesses (pi and Copilot).
 */
import { spawn } from "node:child_process";
import os from "node:os";
import path from "node:path";
import fs from "node:fs";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

const AGENTS_DIR = path.resolve(import.meta.dirname, "..", "..", ".github", "agents");
const READONLY_TOOLS = "read,bash,grep,find,ls";

// Hard cap on one reviewer run. Long runs stay visible (streamed) and
// cancellable, but a hung model must not live forever.
const AGENT_TIMEOUT_MS = 30 * 60 * 1000;
const SIGKILL_GRACE_MS = 5 * 1000;

// Copilot-era tool names appear in the agent prompts; translate them for
// the pi subprocess so the reviewer does not hallucinate tools.
const RUNTIME_NOTE = `

---
**Runtime (pi):** you are running as a fresh pi subprocess, not Copilot.
Tool translation: \`execute\` is \`bash\` (read-only git inspection only),
\`search\` is \`rg\`/grep via bash, \`web\` is \`curl -sL '<url>' | pandoc -f html -t gfm\`.
You have no edit/write tools — never attempt to modify files.
Read REVIEW.md (via bash \`cat REVIEW.md\` or the read tool) before reviewing.`;

// ---------------------------------------------------------------------------
// JSONL stream parsing
//
// `pi --mode json` emits one JSON record per LF-terminated line: a session
// header, then session events. `message_end` carries the authoritative
// final message; we only need the assistant turns (their text is the
// progress stream and the final answer) and their stopReason/errorMessage.
// See docs/json.md of the pi package.
// ---------------------------------------------------------------------------

/** Pull the text out of an assistant message's content (string or blocks). */
export function extractAssistantText(content: unknown): string {
	if (typeof content === "string") return content;
	if (!Array.isArray(content)) return "";
	const parts: string[] = [];
	for (const block of content) {
		if (block && typeof block === "object" && (block as { type?: string }).type === "text") {
			const text = (block as { text?: unknown }).text;
			if (typeof text === "string") parts.push(text);
		}
	}
	return parts.join("");
}

export class JsonStreamParser {
	private _lastAssistantText: string | null = null;
	private _stopReason: string | null = null;
	private _errorMessage: string | null = null;

	/** The text of the most recent assistant message_end (replaced, not appended). */
	get lastAssistantText(): string | null {
		return this._lastAssistantText;
	}

	get stopReason(): string | null {
		return this._stopReason;
	}

	get errorMessage(): string | null {
		return this._errorMessage;
	}

	/** Feed one LF-terminated record (a trailing \r is tolerated). Never throws. */
	feedLine(line: string): void {
		const trimmed = line.replace(/\r$/, "").trim();
		if (!trimmed) return;
		let event: { type?: string; message?: { role?: string; content?: unknown; stopReason?: string; errorMessage?: string } };
		try {
			event = JSON.parse(trimmed);
		} catch {
			return; // stderr noise or a torn final line — diagnostics are not records
		}
		if (event.type !== "message_end" || !event.message) return;
		const message = event.message;
		if (message.role !== "assistant") return;
		this._lastAssistantText = extractAssistantText(message.content);
		if (message.stopReason) this._stopReason = message.stopReason;
		if (message.errorMessage) this._errorMessage = message.errorMessage;
	}
}

// ---------------------------------------------------------------------------
// Agent discovery (unchanged from the execFileSync version)
// ---------------------------------------------------------------------------

type Frontmatter = Record<string, string | string[]>;

function parseAgentFile(file: string): { name: string; description: string; prompt: string; models: string[]; reasoningEffort?: string } {
	const raw = fs.readFileSync(file, "utf8");
	const m = raw.match(/^---\n([\s\S]*?)\n---\n?([\s\S]*)$/);
	if (!m) throw new Error(`no frontmatter in ${file}`);

	const front: Frontmatter = {};
	for (const line of m[1].split("\n")) {
		const kv = line.match(/^(\w[\w-]*):\s*(.*)$/);
		if (!kv) continue;
		let value = kv[2].trim();
		if (value.startsWith("[") && value.endsWith("]")) {
			front[kv[1]] = value
				.slice(1, -1)
				.split(",")
				.map((s) => s.trim().replace(/^["']|["']$/g, ""))
				.filter(Boolean);
		} else {
			front[kv[1]] = value.replace(/^["']|["']$/g, "");
		}
	}

	const name = (front["name"] as string) || path.basename(file, ".agent.md");
	const models = Array.isArray(front["model"]) ? (front["model"] as string[]) : front["model"] ? [front["model"] as string] : [];
	return {
		name,
		description: (front["description"] as string) || "",
		prompt: m[2].trim(),
		models,
		reasoningEffort: front["reasoning-effort"] as string | undefined,
	};
}

function discoverAgents(): Map<string, ReturnType<typeof parseAgentFile>> {
	const map = new Map<string, ReturnType<typeof parseAgentFile>>();
	if (!fs.existsSync(AGENTS_DIR)) return map;
	for (const entry of fs.readdirSync(AGENTS_DIR)) {
		if (!entry.endsWith(".agent.md")) continue;
		try {
			const agent = parseAgentFile(path.join(AGENTS_DIR, entry));
			map.set(agent.name, agent);
		} catch (err) {
			console.error(`biomuzak-subagent: skipping ${entry}: ${String(err)}`);
		}
	}
	return map;
}

/**
 * Model patterns that do not resolve land on stderr, not a non-zero exit: pi
 * substitutes the session default model and exits 0, emitting a diagnostic.
 * Treat such a diagnostic as a failed attempt so the next pinned model is tried
 * — otherwise the pinned fallback is unreachable and the run is attributed to a
 * model that never ran.
 */
export function isModelResolutionError(stderr: string): boolean {
	return (
		/\bmodel\b.*\bnot (found|resolve)/i.test(stderr) ||
		/\bunknown model\b/i.test(stderr) ||
		/\bno models? available\b/i.test(stderr) ||
		/\bno model\b/i.test(stderr)
	);
}

/**
 * The model attempts one run makes, in order. An agent with no `model:` pin (the
 * Web Researcher) still gets exactly one attempt, on the session default
 * (`undefined`), rather than zero — a zero-length list made every unpinned agent
 * unrunnable.
 */
export function attemptModels(pinned: string[], override?: string): (string | undefined)[] {
	if (override) {
		return [override];
	}
	return pinned.length > 0 ? pinned : [undefined];
}

/** A run that produced no assistant text is not a completed review. */
export function isEmptyResult(text: string): boolean {
	return text.trim() === "";
}

// ---------------------------------------------------------------------------
// The async pi subprocess
// ---------------------------------------------------------------------------

interface AgentRunOutcome {
	/** The final assistant text (or the last one before the run died). */
	text: string;
	stderr: string;
	exitCode: number | null;
	terminationSignal: NodeJS.Signals | null;
	stopReason: string | null;
	errorMessage: string | null;
	spawnError: Error | null;
	wasAborted: boolean;
	timedOut: boolean;
}

function runPiAgent(args: string[], task: string, cwd: string, signal: AbortSignal | undefined, onProgress: (text: string) => void): Promise<AgentRunOutcome> {
	return new Promise((resolve) => {
		const child = spawn("pi", [...args, task], { cwd, stdio: ["ignore", "pipe", "pipe"] });
		const parser = new JsonStreamParser();
		let stderr = "";
		let stdoutBuffer = "";
		let spawnError: Error | null = null;
		let terminated = false;
		let timedOut = false;
		let killTimer: NodeJS.Timeout | null = null;

		const kill = () => {
			if (terminated) return;
			terminated = true;
			try {
				child.kill("SIGTERM");
			} catch {}
			// SIGKILL backstop: a reviewer stuck in an uninterruptible
			// provider call must not outlive the user's Esc.
			killTimer = setTimeout(() => {
				try {
					child.kill("SIGKILL");
				} catch {}
			}, SIGKILL_GRACE_MS);
		};

		const onAbort = () => kill();
		signal?.addEventListener("abort", onAbort, { once: true });
		if (signal?.aborted) kill();
		const timeoutTimer = setTimeout(() => {
			timedOut = true;
			kill();
		}, AGENT_TIMEOUT_MS);

		const cleanup = () => {
			signal?.removeEventListener("abort", onAbort);
			clearTimeout(timeoutTimer);
			if (killTimer) clearTimeout(killTimer);
		};

		child.on("error", (err) => {
			spawnError = err;
		});

		child.stdout?.on("data", (chunk: Buffer) => {
			stdoutBuffer += chunk.toString("utf8");
			let newline: number;
			while ((newline = stdoutBuffer.indexOf("\n")) >= 0) {
				const line = stdoutBuffer.slice(0, newline);
				stdoutBuffer = stdoutBuffer.slice(newline + 1);
				const before = parser.lastAssistantText;
				parser.feedLine(line);
				// Stream the reviewer's latest turn into the tool result so
				// a long review shows progress instead of looking hung.
				if (parser.lastAssistantText !== before) onProgress(parser.lastAssistantText!);
			}
		});

		child.stderr?.on("data", (chunk: Buffer) => {
			stderr += chunk.toString("utf8");
		});

		child.on("close", (exitCode, termSignal) => {
			cleanup();
			// A partial final record (no trailing LF) still counts.
			parser.feedLine(stdoutBuffer);
			resolve({
				text: parser.lastAssistantText ?? "",
				stderr,
				exitCode,
				terminationSignal: termSignal,
				stopReason: parser.stopReason,
				errorMessage: parser.errorMessage,
				spawnError,
				wasAborted: signal?.aborted === true,
				timedOut,
			});
		});
	});
}

function buildArgs(agent: ReturnType<typeof parseAgentFile>, systemPromptFile: string, model: string | undefined): string[] {
	const args = [
		"--mode",
		"json",
		"-p",
		"--no-session",
		"--tools",
		READONLY_TOOLS,
		"--system-prompt",
		systemPromptFile,
	];
	if (agent.reasoningEffort) {
		args.push("--thinking", agent.reasoningEffort);
	}
	if (model) {
		args.push("--model", model);
	}
	return args;
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

export default function (pi: ExtensionAPI) {
	const agents = discoverAgents();

	pi.registerTool({
		name: "run_agent",
		label: "Run Agent",
		description:
			`Run one of the repo's pinned agents from .github/agents/ ` +
			`(${[...agents.keys()].join(", ") || "none found"}) ` +
			`in a fresh, isolated, read-only pi context. ` +
			`Use for the review gate: pass the base ref and diff range in \`task\`; the agent reads REVIEW.md itself. ` +
			`The run streams progress, stays cancellable (Esc) and never blocks the TUI.`,
		parameters: {
			type: "object",
			required: ["agent", "task"],
			properties: {
				agent: { type: "string", description: `Agent name exactly as in its .github/agents/*.agent.md frontmatter, e.g. '${[...agents.keys()][0] ?? "Code Reviewer"}'` },
				task: { type: "string", description: "The task: what to review/audit/classify/research, with base ref and diff range (or URLs for the Web Researcher)" },
				model: { type: "string", description: "Override the pinned model with a pi model pattern, e.g. 'opencode-go/deepseek-v4.1-flash'" },
			},
		},

		async execute(_toolCallId, params: { agent: string; task: string; model?: string }, signal, onUpdate) {
			const agent = agents.get(params.agent);
			if (!agent) {
				throw new Error(`unknown agent '${params.agent}'. Available: ${[...agents.keys()].join(", ")}`);
			}

			const models = attemptModels(agent.models, params.model);
			const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "biomuzak-agent-"));
			try {
			return await runAttempts(agent, models, tmpDir, params.task, signal, onUpdate);
			} finally {
				fs.rmSync(tmpDir, { recursive: true, force: true });
			}
		},
	});
}

// runAttempts is the model-attempt loop, split out so `execute` can wrap it in a
// single try/finally that reclaims the temp directory on every exit path.
async function runAttempts(
	agent: ReturnType<typeof parseAgentFile>,
	models: (string | undefined)[],
	tmpDir: string,
	task: string,
	signal: AbortSignal | undefined,
	onUpdate: Parameters<Parameters<ExtensionAPI["registerTool"]>[0]["execute"]>[3],
) {
			const systemPromptFile = path.join(tmpDir, "system-prompt.md");
			const taskFile = path.join(tmpDir, "task.md");
			fs.writeFileSync(systemPromptFile, agent.prompt + RUNTIME_NOTE, "utf8");
			fs.writeFileSync(taskFile, task, "utf8");

			let lastOutcome: AgentRunOutcome | null = null;
			for (let i = 0; i < models.length; i++) {
				const model = models[i];
				const outcome = await runPiAgent(buildArgs(agent, systemPromptFile, model), `Task file: ${taskFile}. Read it and complete the task.`, process.cwd(), signal, (text) => {
					onUpdate?.({
						content: [{ type: "text", text }],
						details: { agent: agent.name, model, status: "running" },
					});
				});
				lastOutcome = outcome;

				// User cancelled: surface it and stop — never retry another model.
				if (outcome.wasAborted) {
					throw new Error(`agent '${agent.name}' aborted by the user${outcome.text ? ` — last output:\n${outcome.text}` : ""}`);
				}
				if (outcome.timedOut) {
					throw new Error(`agent '${agent.name}' (model ${model}) timed out after ${AGENT_TIMEOUT_MS / 60000} minutes${outcome.text ? ` — last output:\n${outcome.text}` : ""}`);
				}
				if (outcome.spawnError) {
					throw new Error(`failed to start the reviewer subprocess: ${outcome.spawnError.message}`);
				}
				if (outcome.terminationSignal) {
					throw new Error(`agent '${agent.name}' (model ${model}) was terminated by ${outcome.terminationSignal}`);
				}
				if (outcome.stopReason === "aborted") {
					throw new Error(`agent '${agent.name}' aborted mid-run${outcome.errorMessage ? `: ${outcome.errorMessage}` : ""}${outcome.text ? `\nLast output:\n${outcome.text}` : ""}`);
				}
				// A model that does not resolve must not be reported as a success: pi
				// falls back to the session default and exits 0. Try the next pinned
				// model; if this was the last, the run is a resolution failure.
				if (isModelResolutionError(outcome.stderr) || isModelResolutionError(outcome.text)) {
					if (i < models.length - 1) continue;
					const detail = outcome.stderr.trim().slice(-2000) || outcome.text.trim().slice(-2000);
					throw new Error(`agent '${agent.name}': no pinned model resolved (tried: ${models.map((m) => m ?? "<session default>").join(", ")}) — ${detail}`);
				}
				if (outcome.exitCode !== 0) {
					const detail = outcome.stderr.trim() || `exit code ${outcome.exitCode}`;
					throw new Error(`agent '${agent.name}' (model ${model ?? "session default"}) failed: ${detail.slice(-2000)}${outcome.text ? `\nLast output:\n${outcome.text}` : ""}`);
				}
				if (outcome.stopReason === "error") {
					throw new Error(`agent '${agent.name}' (model ${model ?? "session default"}) ended with an error${outcome.errorMessage ? `: ${outcome.errorMessage}` : ""}${outcome.text ? `\nLast output:\n${outcome.text}` : ""}`);
				}
				if (isEmptyResult(outcome.text)) {
					throw new Error(`agent '${agent.name}' (model ${model ?? "session default"}) produced no answer (stopReason ${outcome.stopReason ?? "none"}) — a marker must not be written for an empty review`);
				}

				const header = `[${agent.name} — model: ${model ?? "session default"}]`;
				return {
					content: [{ type: "text" as const, text: `${header}\n\n${outcome.text.trim()}` }],
					details: { agent: agent.name, model: model ?? "session default" },
				};
			}

			const detail = lastOutcome ? lastOutcome.stderr.trim().slice(-2000) || `exit code ${lastOutcome.exitCode}` : "no model was tried";
			throw new Error(`agent '${agent.name}': no pinned model resolved — ${detail}`);
}
