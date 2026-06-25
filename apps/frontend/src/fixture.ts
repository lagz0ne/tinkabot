export function generatedUrl() {
  return URL.createObjectURL(new Blob([generatedHtml()], { type: "text/html" }));
}

function generatedHtml() {
  return `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <style>
      * { box-sizing: border-box; }
      html { background: #eef2f5; }
      body {
        margin: 0;
        color: #1d2329;
        background: #eef2f5;
        font: 14px/1.4 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        -webkit-font-smoothing: antialiased;
      }
      main { min-height: 100vh; padding: 20px; display: grid; gap: 14px; align-content: start; }
      h2 { margin: 0; font-size: 22px; }
      p { margin: 0; color: #52606c; }
      dl { display: grid; gap: 8px; margin: 0; }
      dl > div { display: grid; grid-template-columns: 120px minmax(0, 1fr); gap: 8px; border-top: 1px solid #e5e9ef; padding-top: 8px; }
      dt { color: #687482; }
      dd { margin: 0; overflow-wrap: anywhere; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
      button, input { min-height: 44px; padding: 8px 12px; border: 1px solid #b8c2cc; border-radius: 6px; background: #f8fafc; color: #1d2329; font: inherit; touch-action: manipulation; }
      button { width: max-content; }
      button:disabled { opacity: .5; }
      .chess { min-height: 100vh; padding: clamp(12px, 3vw, 24px); display: grid; grid-template-columns: minmax(270px, 360px) minmax(360px, min(72vh, 680px)); align-items: start; justify-content: center; gap: clamp(14px, 3vw, 28px); background: #eef2f5; }
      .chess-panel { align-content: start; gap: 12px; padding: 16px; border: 1px solid #d8dee5; border-radius: 8px; background: #fff; box-shadow: 0 12px 32px rgba(31, 41, 55, .10); }
      .chess-head { display: grid; grid-template-columns: minmax(0, 1fr) max-content; gap: 12px; align-items: start; border: 0; padding: 0; }
      .chess-head h2 { overflow-wrap: anywhere; }
      .chess-status { color: #1f5f3d; font-weight: 700; }
      .chess-badge { align-self: start; padding: 5px 8px; border: 1px solid #d5dde5; border-radius: 999px; background: #f8fafc; color: #52606c; font-size: 12px; font-weight: 700; text-transform: uppercase; }
      .chess-form { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin: 0 0 4px; border: 0; padding: 0; }
      .chess-form label { display: grid; gap: 5px; color: #52606c; font-size: 13px; }
      .chess-form input { width: 100%; background: #fff; }
      .chess-form button { width: 100%; }
      .chess-score { gap: 6px; }
      .chess-score > div { grid-template-columns: 74px minmax(0, 1fr); }
      .chess-proof-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
      .chess-board { display: grid; grid-template-columns: repeat(8, minmax(0, 1fr)); width: min(92vw, min(72vh, 680px)); aspect-ratio: 1; border: 2px solid #24313c; border-radius: 8px; overflow: hidden; background: #24313c; box-shadow: 0 18px 40px rgba(31, 41, 55, .20); touch-action: none; user-select: none; }
      .chess-square { position: relative; width: 100%; height: 100%; min-height: 0; padding: 0; border: 0; border-radius: 0; display: grid; place-items: center; font-size: clamp(28px, 7.4vmin, 56px); line-height: 1; -webkit-tap-highlight-color: transparent; }
      .chess-light { background: #e8d9b8; }
      .chess-dark { background: #55785f; }
      .chess-square[data-own="true"] { cursor: pointer; }
      .chess-piece { width: 82%; height: 82%; border-radius: 50%; display: grid; place-items: center; font-family: "Noto Color Emoji", "Apple Color Emoji", "Segoe UI Emoji", "Noto Sans Symbols 2", "DejaVu Sans", sans-serif; transform: translateY(-1px); }
      .chess-white-piece { background: rgba(255, 255, 255, .86); color: #f8fafc; text-shadow: 0 1px 0 #475569, 0 2px 7px rgba(15, 23, 42, .55); box-shadow: inset 0 -5px 10px rgba(148, 163, 184, .18), 0 3px 9px rgba(15, 23, 42, .22); }
      .chess-black-piece { background: rgba(15, 23, 42, .84); color: #111827; text-shadow: 0 1px 0 #f8fafc, 0 2px 7px rgba(255, 255, 255, .28); box-shadow: inset 0 4px 10px rgba(255, 255, 255, .12), 0 3px 9px rgba(15, 23, 42, .24); }
      .chess-selected { outline: 4px solid #f5c542; outline-offset: -4px; z-index: 1; }
      .chess-target::after { content: ""; position: absolute; width: 28%; height: 28%; border-radius: 50%; background: rgba(245, 197, 66, .42); }
      .chess-last { box-shadow: inset 0 0 0 4px rgba(64, 120, 87, .55); }
      .chess-pending .chess-piece { opacity: .72; transform: translateY(-1px) scale(.96); }
      .chess-moves { max-height: 180px; overflow: auto; padding-left: 22px; color: #334155; }
      .type-race { min-height: 100vh; display: grid; grid-template-columns: minmax(280px, 380px) minmax(360px, 760px); gap: clamp(14px, 3vw, 28px); padding: clamp(14px, 3vw, 28px); align-items: start; justify-content: center; background: #edf3f1; color: #1b2724; }
      .type-panel, .type-track { display: grid; gap: 14px; padding: 18px; border: 1px solid #d4ddd8; border-radius: 8px; background: #fff; box-shadow: 0 14px 30px rgba(27, 39, 36, .10); }
      .type-head { display: grid; grid-template-columns: minmax(0, 1fr) max-content; gap: 12px; align-items: start; }
      .type-head h2 { overflow-wrap: anywhere; }
      .type-badge { padding: 5px 8px; border: 1px solid #d6ded9; border-radius: 999px; background: #f7faf8; color: #52615a; font-size: 12px; font-weight: 700; text-transform: uppercase; }
      .type-status { color: #1f6b4f; font-weight: 700; }
      .type-prompt { padding: 16px; border: 1px solid #d7ded9; border-radius: 8px; background: #f7faf8; color: #26332e; font-size: 18px; line-height: 1.7; }
      .type-prompt span { border-radius: 4px; }
      .type-ok { background: #d7f4dc; }
      .type-bad { background: #ffe1d6; color: #912f20; }
      .type-input { width: 100%; min-height: 132px; resize: vertical; border: 1px solid #b8c5be; border-radius: 8px; background: #fff; color: #17201d; font: 18px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
      .type-actions { display: flex; flex-wrap: wrap; gap: 10px; }
      .type-actions button { min-width: 112px; }
      .type-score { gap: 8px; }
      .type-score > div { grid-template-columns: 92px minmax(0, 1fr); }
      .type-runners { display: grid; gap: 12px; }
      .type-runner { display: grid; gap: 7px; padding: 12px; border: 1px solid #e0e7e2; border-radius: 8px; background: #fbfcfb; }
      .type-runner-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; font-weight: 700; }
      .type-meter { height: 14px; overflow: hidden; border-radius: 999px; background: #e6ece8; }
      .type-meter > span { display: block; height: 100%; min-width: 0; border-radius: inherit; background: linear-gradient(90deg, #38a169, #2b7bbb); }
      .type-proof-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
      @media (max-width: 760px) {
        .chess { grid-template-columns: 1fr; padding: 10px; gap: 12px; }
        .chess-panel { order: 2; padding: 14px; }
        .chess-board { order: 1; width: min(96vw, 620px); margin: 0 auto; }
        .chess-form { grid-template-columns: 1fr; }
        .type-race { grid-template-columns: 1fr; padding: 10px; gap: 12px; }
        .type-panel { order: 2; }
        .type-track { order: 1; }
      }
    </style>
  </head>
  <body>
    <main id="generated">Generated content</main>
    <script>
      const read = (fn) => {
        try { return String(fn()); } catch (err) { return "denied:" + err.name; }
      };
      const state = {
        lease: null,
        demo: null,
        baseRevision: 0,
        sent: 0,
        actions: 0,
        readbacks: 0,
        denied: 0,
        complete: false,
        failed: false,
        choice: "",
        itemKey: "",
        last: "booting",
        pending: new Map(),
        board: { turn: "", cells: {}, winner: "" },
        chess: null,
        nameDraft: "",
        pendingMove: null,
        selectedSquare: "",
        typeRace: null,
        typeDraft: "",
        typeQueued: "",
        typeSending: false,
        alias: "",
        delivery: "",
        pushes: 0,
        waiters: [],
	      };
      const root = document.querySelector("#generated");

	      window.addEventListener("message", (event) => {
	        if (event.data?.type === "tinkabot.lease") {
	          state.lease = event.data.lease;
	          state.demo = event.data.demo || {};
	          if (state.demo.chess) state.nameDraft = String(state.demo.playerName || "");
	          if (state.demo.typeRace) state.alias = String(state.demo.alias || "");

          parent.postMessage({
            type: "content.probe",
            cookie: read(() => document.cookie),
            storage: read(() => localStorage.length),
          }, "*");

          if (state.lease.commands.includes("participant_action") && state.demo.typeRace) {
            renderTypeRace();
            runTypeRace().catch((err) => {
              state.last = err instanceof Error ? err.message : String(err);
              state.failed = true;
              renderTypeRace();
            });
          } else if (state.lease.commands.includes("participant_action") && state.demo.chess) {
            renderChess();
            runChess().catch((err) => {
              state.last = err instanceof Error ? err.message : String(err);
              state.failed = true;
              renderChess();
            });
          } else if (state.lease.commands.includes("participant_action") && state.demo.board) {
            renderBoard();
            runBoard().catch((err) => {
              state.last = err instanceof Error ? err.message : String(err);
              state.failed = true;
              renderBoard();
            });
          } else if (state.lease.commands.includes("participant_action")) {
            renderParticipant();
            runParticipant().catch((err) => {
              state.last = err instanceof Error ? err.message : String(err);
              state.failed = true;
              state.complete = true;
              renderParticipant();
            });
          } else if (state.lease.commands.includes("item_submit")) {
            renderVisual();
            runVisual().catch((err) => {
              state.last = err instanceof Error ? err.message : String(err);
              state.failed = true;
              state.complete = true;
              renderVisual();
            });
          } else {
            renderDefault();
            sendDefaultIntents(state.lease);
          }
          return;
        }
        if (event.data?.type === "tinkabot.state") {
          applyState(event.data);
          return;
        }
        if (event.data?.type === "tinkabot.command.result") {
          const hit = state.pending.get(event.data.commandId);
          if (!hit) return;
          state.pending.delete(event.data.commandId);
          if (event.data.error) {
            hit.reject(new Error(event.data.error));
          } else {
            hit.resolve(event.data.response);
          }
        }
      });

      window.__tinkabotDemo = {
        refresh: () => refreshBoard(),
        snapshot: () => boardSnapshot(),
        submit: (cell, options = {}) => submitBoardAction(String(cell), options),
        escape: () => sendScopeEscape(),
      };
      window.__tinkabotChess = {
        refresh: () => refreshChess(),
        snapshot: () => chessSnapshot(),
        read: (key) => send("participant_read", { key: String(key) }, "chess-read"),
        join: (name, board) => joinChess(String(name), String(board || state.demo.boardNo || "")),
        move: (from, to, promotion = "q", options = {}) => moveChess(String(from), String(to), String(promotion || "q"), options),
        resign: () => resignChess(),
      };
      window.__tinkabotTypeRace = {
        refresh: () => refreshTypeRace(),
        snapshot: () => typeRaceSnapshot(),
        read: (key) => send("participant_read", { key: String(key) }, "type-read"),
        join: (alias) => joinTypeRace(String(alias || state.alias || "")),
        progress: (typed, options = {}) => progressTypeRace(String(typed || ""), options),
        typeText: (typed, options = {}) => progressTypeRace(String(typed || ""), options),
        escape: (participantId) => sendTypeScopeEscape(String(participantId || "")),
      };

      function sendDefaultIntents(lease) {
        parent.postMessage({
          type: "content.intent",
          command: "select_artifact",
          commandId: "cmd-frame-001",
          expectedRevision: lease.artifactRevision,
          nonce: lease.nonce,
          frameId: lease.frameId,
          artifactRevision: lease.artifactRevision,
          schemaRevision: lease.schemaRevision,
          payload: { artifactKey: "preview.main" },
        }, "*");

        parent.postMessage({
          type: "content.intent",
          command: "select_artifact",
          commandId: "cmd-raw-001",
          expectedRevision: lease.artifactRevision,
          nonce: lease.nonce,
          frameId: lease.frameId,
          artifactRevision: lease.artifactRevision,
          schemaRevision: lease.schemaRevision,
          payload: { subject: "tb.internal.admin.delete" },
        }, "*");
      }

      async function runVisual() {
        const key = String(state.demo.visualKey || "");
        if (key === "") throw new Error("visual key missing");
        state.choice = String(state.demo.choice || "diagram-a");
        const response = await send("item_submit", {
          key,
          status: "resolved",
          expectedRevision: 0,
          value: {
            choice: state.choice,
            label: state.choice,
            submittedAt: new Date().toISOString(),
          },
        }, "submit-choice");
        if (response?.status !== "accepted") {
          state.denied += 1;
          state.last = "submit denied: " + (response?.reason || "unknown");
          renderVisual();
          return;
        }
        state.itemKey = response?.item?.key || key;
        state.complete = true;
        state.last = "submitted";
        renderVisual();
      }

      async function runParticipant() {
        const initial = await send("participant_read", { key: state.demo.stateKey }, "read-state");
        state.baseRevision = Number(initial?.item?.revision || 0);
        if (!state.baseRevision) throw new Error("state revision missing");
        renderParticipant();

        const count = Math.max(0, Number(state.demo.autoActions || 0));
        for (let i = 0; i < count; i++) {
          await submitAction();
          const wait = Math.max(1, Number(state.demo.intervalMs || 25));
          await sleep(wait);
        }
        state.complete = true;
        state.last = "complete";
        renderParticipant();
      }

      async function runBoard() {
        state.last = "watching";
        renderBoard();
      }

      async function runChess() {
        state.last = "watching";
        renderChess();
      }

      async function runTypeRace() {
        state.last = "watching";
        renderTypeRace();
      }

      async function refreshBoard() {
        const response = await send("participant_read", { key: state.demo.stateKey }, "read-board");
        if (response?.status !== "accepted" || !response.item) {
          state.denied += 1;
          state.last = "board read denied: " + (response?.reason || "unknown");
          renderBoard();
          throw new Error(state.last);
        }
        state.baseRevision = Number(response.item.revision || 0);
        state.board = normalizeBoard(response.item.value);
        state.readbacks += 1;
        state.last = "board rev " + state.baseRevision;
        renderBoard();
        return boardSnapshot();
      }

      async function submitBoardAction(cell, options = {}) {
        if (!state.baseRevision) await waitForState();
        const actionId = String(options.actionId || ("board-" + state.lease.participantId + "-" + cell + "-" + Date.now()));
        const action = await send("participant_action", {
          appId: options.appId || state.lease.appId,
          participantId: options.participantId || state.lease.participantId,
          actionId,
          stateKey: state.demo.stateKey,
          baseRevision: Number(options.baseRevision || state.baseRevision),
          value: { cell },
        }, "board-action");
        if (action?.status !== "accepted") {
          state.denied += 1;
          state.last = "action denied: " + (action?.reason || "unknown");
          renderBoard();
          return { action, snapshot: boardSnapshot() };
        }
        state.actions += 1;
        const actionKey = action?.item?.key;
        if (typeof actionKey !== "string" || actionKey === "") {
          state.denied += 1;
          state.last = "action response missing item";
          renderBoard();
          throw new Error(state.last);
        }
        state.itemKey = actionKey;
        state.last = actionKey;
        renderBoard();
        return { action, snapshot: boardSnapshot() };
      }

	      async function refreshChess() {
	        const response = await send("participant_read", { key: state.demo.stateKey }, "read-chess");
	        if (response?.status !== "accepted" || !response.item) {
	          state.denied += 1;
	          state.last = "chess read denied: " + (response?.reason || "unknown");
          renderChess();
          throw new Error(state.last);
        }
        state.baseRevision = Number(response.item.revision || 0);
	        state.chess = normalizeChess(response.item.value);
	        if (state.pendingMove && state.baseRevision !== state.pendingMove.baseRevision) state.pendingMove = null;
	        state.readbacks += 1;
	        state.last = "chess rev " + state.baseRevision;
	        if (!chessNameFocused()) renderChess();
	        return chessSnapshot();
	      }

      async function refreshTypeRace() {
        const response = await send("participant_read", { key: state.demo.stateKey }, "read-type");
        if (response?.status !== "accepted" || !response.item) {
          state.denied += 1;
          state.last = "type read denied: " + (response?.reason || "unknown");
          renderTypeRace();
          throw new Error(state.last);
        }
        state.baseRevision = Number(response.item.revision || 0);
        state.typeRace = normalizeTypeRace(response.item.value);
        state.readbacks += 1;
        state.last = "race rev " + state.baseRevision;
        if (!typeInputFocused()) renderTypeRace();
        return typeRaceSnapshot();
      }

      function applyState(msg) {
        const item = msg.item || {};
        if (!state.demo || item.key !== state.demo.stateKey) return;
        const rev = Number(item.revision || 0);
        if (!rev || rev < state.baseRevision) return;
        state.baseRevision = rev;
        state.delivery = String(msg.source || "trusted-shell.nats-watch.push");
        state.pushes += 1;
        if (state.demo.chess) {
          state.chess = normalizeChess(item.value);
          if (state.pendingMove && state.baseRevision !== state.pendingMove.baseRevision) state.pendingMove = null;
          state.last = "push rev " + state.baseRevision;
          resolveWaiters();
          if (!chessNameFocused()) renderChess();
          return;
        }
        if (state.demo.typeRace) {
          state.typeRace = normalizeTypeRace(item.value);
          state.last = "push rev " + state.baseRevision;
          resolveWaiters();
          if (!typeInputFocused()) renderTypeRace();
          flushTypeProgress();
          return;
        }
        if (state.demo.board) {
          state.board = normalizeBoard(item.value);
          state.last = "push rev " + state.baseRevision;
          resolveWaiters();
          renderBoard();
          return;
        }
        state.last = "push rev " + state.baseRevision;
        resolveWaiters();
        renderParticipant();
      }

      function waitForState() {
        if (state.baseRevision) return Promise.resolve();
        return new Promise((resolve, reject) => {
          const timer = setTimeout(() => {
            state.waiters = state.waiters.filter((hit) => hit.resolve !== resolve);
            reject(new Error("state watch timeout"));
          }, 7000);
          state.waiters.push({ resolve: () => {
            clearTimeout(timer);
            resolve();
          } });
        });
      }

      function resolveWaiters() {
        const waiters = state.waiters;
        state.waiters = [];
        waiters.forEach((hit) => hit.resolve());
      }

	      async function joinChess(name, boardNo) {
	        if (!state.baseRevision) await waitForState();
	        const cleanName = name.trim();
	        const cleanBoard = boardNo.trim();
	        state.nameDraft = cleanName;
	        const action = await send("participant_action", {
          appId: state.lease.appId,
          participantId: state.lease.participantId,
          actionId: "join-" + state.lease.participantId + "-" + Date.now(),
          stateKey: state.demo.stateKey,
          baseRevision: state.baseRevision,
          value: { type: "join", board: cleanBoard, name: cleanName },
        }, "chess-join");
        return afterChessAction(action);
      }

      async function moveChess(from, to, promotion = "q", options = {}) {
        if (!state.baseRevision) await waitForState();
        const baseRevision = Number(options.baseRevision || state.baseRevision);
        state.pendingMove = { from, to, baseRevision };
        state.selectedSquare = "";
        renderChess();
        try {
          const action = await send("participant_action", {
            appId: state.lease.appId,
            participantId: state.lease.participantId,
            actionId: String(options.actionId || ("move-" + state.lease.participantId + "-" + from + to + "-" + Date.now())),
            stateKey: state.demo.stateKey,
            baseRevision,
            value: { type: "move", board: state.demo.boardNo, from, to, promotion },
          }, "chess-move");
          return afterChessAction(action);
        } catch (err) {
          state.pendingMove = null;
          throw err;
        }
      }

      async function resignChess() {
        if (!state.baseRevision) await waitForState();
        const action = await send("participant_action", {
          appId: state.lease.appId,
          participantId: state.lease.participantId,
          actionId: "resign-" + state.lease.participantId + "-" + Date.now(),
          stateKey: state.demo.stateKey,
          baseRevision: state.baseRevision,
          value: { type: "resign", board: state.demo.boardNo },
        }, "chess-resign");
        return afterChessAction(action);
      }

      async function afterChessAction(action) {
        if (action?.status !== "accepted") {
          state.pendingMove = null;
          state.denied += 1;
          state.last = "action denied: " + (action?.reason || "unknown");
          renderChess();
          return { action, snapshot: chessSnapshot() };
        }
        state.actions += 1;
        const actionKey = action?.item?.key;
        if (typeof actionKey !== "string" || actionKey === "") {
          state.denied += 1;
          state.last = "action response missing item";
          renderChess();
          throw new Error(state.last);
        }
        state.itemKey = actionKey;
        state.last = actionKey;
        renderChess();
        return { action, snapshot: chessSnapshot() };
      }

      async function joinTypeRace(alias) {
        if (!state.baseRevision) await waitForState();
        const clean = alias.trim() || state.alias || anonymousAlias();
        state.alias = clean;
        const action = await send("participant_action", {
          appId: state.lease.appId,
          participantId: state.lease.participantId,
          actionId: "join-" + state.lease.participantId + "-" + Date.now(),
          stateKey: state.demo.stateKey,
          baseRevision: state.baseRevision,
          value: { type: "join", race: state.demo.raceNo, alias: clean },
        }, "type-join");
        return afterTypeAction(action);
      }

      async function progressTypeRace(typed, options = {}) {
        if (!state.baseRevision) await waitForState();
        const value = String(typed || "");
        state.typeDraft = value;
        const action = await send("participant_action", {
          appId: state.lease.appId,
          participantId: state.lease.participantId,
          actionId: String(options.actionId || ("type-" + state.lease.participantId + "-" + Date.now())),
          stateKey: state.demo.stateKey,
          baseRevision: Number(options.baseRevision || state.baseRevision),
          value: { type: "progress", race: state.demo.raceNo, typed: value },
        }, "type-progress");
        return afterTypeAction(action);
      }

      async function afterTypeAction(action) {
        if (action?.status !== "accepted") {
          state.denied += 1;
          state.last = "action denied: " + (action?.reason || "unknown");
          if (!typeInputFocused()) renderTypeRace();
          return { action, snapshot: typeRaceSnapshot() };
        }
        state.actions += 1;
        const actionKey = action?.item?.key;
        if (typeof actionKey !== "string" || actionKey === "") {
          state.denied += 1;
          state.last = "action response missing item";
          if (!typeInputFocused()) renderTypeRace();
          throw new Error(state.last);
        }
        state.itemKey = actionKey;
        state.last = actionKey;
        if (!typeInputFocused()) renderTypeRace();
        return { action, snapshot: typeRaceSnapshot() };
      }

      function queueTypeProgress() {
        state.typeQueued = state.typeDraft;
        flushTypeProgress();
      }

      function flushTypeProgress() {
        if (!state.demo?.typeRace || state.typeSending || !state.baseRevision || state.typeQueued === "") return;
        const typed = state.typeQueued;
        state.typeQueued = "";
        state.typeSending = true;
        progressTypeRace(typed)
          .catch((err) => {
            state.last = err instanceof Error ? err.message : String(err);
            state.failed = true;
            if (!typeInputFocused()) renderTypeRace();
          })
          .finally(() => {
            state.typeSending = false;
          });
      }

      function sendTypeScopeEscape(participantId) {
        const lease = state.lease;
        const target = participantId || (lease.participantId === "anon-a" ? "anon-b" : "anon-a");
        parent.postMessage({
          type: "content.intent",
          command: "participant_action",
          commandId: "type-escape-" + (++state.sent) + "-" + Date.now(),
          expectedRevision: lease.artifactRevision,
          nonce: lease.nonce,
          frameId: lease.frameId,
          artifactRevision: lease.artifactRevision,
          schemaRevision: lease.schemaRevision,
          appId: lease.appId,
          participantId: target,
          payload: {
            actionId: "escape",
            stateKey: state.demo.stateKey,
            baseRevision: state.baseRevision,
            value: { type: "progress", race: state.demo.raceNo, typed: "" },
          },
        }, "*");
      }

      function sendScopeEscape() {
        const lease = state.lease;
        parent.postMessage({
          type: "content.intent",
          command: "participant_action",
          commandId: "escape-" + (++state.sent) + "-" + Date.now(),
          expectedRevision: lease.artifactRevision,
          nonce: lease.nonce,
          frameId: lease.frameId,
          artifactRevision: lease.artifactRevision,
          schemaRevision: lease.schemaRevision,
          appId: lease.appId,
          participantId: lease.participantId === "alice" ? "bob" : "alice",
          payload: {
            actionId: "escape",
            stateKey: state.demo.stateKey,
            baseRevision: state.baseRevision,
            value: { cell: "a1" },
          },
        }, "*");
      }

      async function submitAction() {
        const seq = state.actions + 1;
        const actionId = "rt-" + seq + "-" + Date.now();
        const action = await send("participant_action", {
          appId: state.lease.appId,
          participantId: state.lease.participantId,
          actionId,
          stateKey: state.demo.stateKey,
          baseRevision: state.baseRevision,
          value: { seq, participant: state.lease.participantId, submittedAt: new Date().toISOString() },
        }, "action");
        if (action?.status !== "accepted") {
          state.denied += 1;
          state.last = "action denied: " + (action?.reason || "unknown");
          renderParticipant();
          return;
        }
        state.actions += 1;
        const actionKey = action?.item?.key;
        if (typeof actionKey !== "string" || actionKey === "") {
          state.denied += 1;
          state.last = "action response missing item";
          renderParticipant();
          throw new Error(state.last);
        }
        state.last = actionKey;
        renderParticipant();

        const readback = await send("participant_read", { key: actionKey }, "read-action");
        if (readback?.status === "accepted") state.readbacks += 1;
        else state.denied += 1;
        renderParticipant();
      }

      function send(command, payload, label) {
        const lease = state.lease;
        const commandId = label + "-" + (++state.sent) + "-" + Date.now();
        const msg = {
          type: "content.intent",
          command,
          commandId,
          expectedRevision: lease.artifactRevision,
          nonce: lease.nonce,
          frameId: lease.frameId,
          artifactRevision: lease.artifactRevision,
          schemaRevision: lease.schemaRevision,
          appId: lease.appId,
          participantId: lease.participantId,
          payload,
        };
        parent.postMessage(msg, "*");
        return new Promise((resolve, reject) => {
          state.pending.set(commandId, { resolve, reject });
          setTimeout(() => {
            if (!state.pending.has(commandId)) return;
            state.pending.delete(commandId);
            reject(new Error("command timeout: " + command));
          }, 7000);
        });
      }

      function renderDefault() {
        root.innerHTML = "<h2>Generated content</h2><p>Default artifact proof is active.</p>";
      }

      function renderParticipant() {
        const lease = state.lease || {};
        root.innerHTML =
          "<h2 data-demo=\\"title\\">Participant " + escapeHtml(lease.participantId || "") + "</h2>" +
          "<p data-demo=\\"status\\">" + escapeHtml(state.failed ? "failed" : state.complete ? "complete" : "running") + "</p>" +
          "<button data-demo=\\"submit\\">Submit</button>" +
          "<dl>" +
          row("App", lease.appId || "") +
          row("State", state.demo?.stateKey || "") +
          row("Base Rev", String(state.baseRevision || 0)) +
          row("Actions", String(state.actions), "actions") +
          row("Readbacks", String(state.readbacks), "readbacks") +
          row("Denied", String(state.denied), "denied") +
          row("Last", state.last) +
          "</dl>";
        root.querySelector("[data-demo=submit]")?.addEventListener("click", () => {
          submitAction().catch((err) => {
            state.last = err instanceof Error ? err.message : String(err);
            renderParticipant();
          });
        });
        if (state.complete) root.dataset.complete = "true";
      }

      function renderBoard() {
        const lease = state.lease || {};
        const board = normalizeBoard(state.board);
        const status = state.failed ? "failed" : board.winner ? "winner " + board.winner : board.turn ? "turn " + board.turn : "loading";
        const cells = ["a1", "a2", "a3", "b1", "b2", "b3", "c1", "c2", "c3"].map((cell) => {
          const owner = board.cells[cell] || "";
          const label = owner === "alice" ? "X" : owner === "bob" ? "O" : "";
          const disabled = owner || board.winner ? " disabled" : "";
          return "<button data-cell=\\"" + cell + "\\"" + disabled + ">" + escapeHtml(label || cell) + "</button>";
        }).join("");
        root.innerHTML =
          "<h2 data-demo=\\"title\\">Board " + escapeHtml(lease.participantId || "") + "</h2>" +
          "<p data-demo=\\"status\\">" + escapeHtml(status) + "</p>" +
          "<section data-demo=\\"board\\" style=\\"display:grid;grid-template-columns:repeat(3,64px);gap:8px\\">" + cells + "</section>" +
          "<dl>" +
          row("App", lease.appId || "") +
          row("State", state.demo?.stateKey || "") +
          row("Revision", String(state.baseRevision || 0), "revision") +
          row("Turn", board.turn || "", "turn") +
          row("Winner", board.winner || "", "winner") +
          row("Cells", boardCellsText(board), "cells") +
          row("Delivery", state.delivery, "delivery") +
          row("Actions", String(state.actions), "actions") +
          row("Reads", String(state.readbacks), "readbacks") +
          row("Denied", String(state.denied), "denied") +
          row("Last", state.last) +
          "</dl>";
        root.querySelectorAll("[data-cell]").forEach((button) => {
          button.addEventListener("click", () => {
            submitBoardAction(button.getAttribute("data-cell")).catch((err) => {
              state.last = err instanceof Error ? err.message : String(err);
              state.failed = true;
              renderBoard();
            });
          });
        });
        root.dataset.boardReady = state.baseRevision ? "true" : "false";
        if (board.winner) root.dataset.complete = "true";
      }

      function renderChess() {
        const chess = normalizeChess(state.chess);
        const color = chessColor(chess, state.lease?.participantId || "");
        const boardNo = state.demo?.boardNo || chess.board || "";
        const status = chess.result?.status
          ? (chess.result.winner ? chess.result.status + " " + chess.result.winner : chess.result.status)
          : state.pendingMove
            ? "move pending"
          : chess.status === "active"
            ? "turn " + chess.turn
            : "waiting";
        const joined = color !== "";
        root.innerHTML =
          "<main class=\\"chess\\" data-chess=\\"app\\">" +
          "<section class=\\"chess-panel\\">" +
          "<header class=\\"chess-head\\">" +
          "<div><h2 data-chess=\\"title\\">Board " + escapeHtml(shortBoard(boardNo)) + "</h2>" +
	          "<p class=\\"chess-status\\" data-chess=\\"status\\">" + escapeHtml(status) + "</p></div>" +
          "<span class=\\"chess-badge\\">" + escapeHtml(color || "viewer") + "</span>" +
          "</header>" +
		          "<div class=\\"chess-form\\">" +
		          "<label>Name<input data-chess=\\"name\\" value=\\"" + escapeHtml(state.nameDraft) + "\\" /></label>" +
		          "<label>Board<input data-chess=\\"board-no\\" value=\\"" + escapeHtml(boardNo) + "\\" readonly /></label>" +
          "<button data-chess=\\"join\\">" + escapeHtml(joined ? "Joined as " + color : "Join") + "</button>" +
          "<button data-chess=\\"resign\\">Resign</button>" +
          "</div>" +
          "<dl class=\\"chess-score\\">" +
          row("White", playerLabel(chess.players.white), "white") +
          row("Black", playerLabel(chess.players.black), "black") +
          row("You", color || "spectator", "color") +
          row("Turn", chess.turn, "turn") +
          row("Result", chess.result?.status || "", "result") +
          "</dl>" +
          "<dl class=\\"chess-proof-only\\">" +
          row("FEN", chess.fen, "fen") +
          row("Revision", String(state.baseRevision || 0), "revision") +
          row("Delivery", state.delivery, "delivery") +
          row("Last", state.last) +
          "</dl>" +
          "<ol class=\\"chess-moves\\" data-chess=\\"moves\\">" + chess.history.map((m, i) => "<li>" + escapeHtml(String(i + 1) + ". " + (m.san || m.from + "-" + m.to)) + "</li>").join("") + "</ol>" +
          "</section>" +
          "<section class=\\"chess-board\\" data-chess=\\"board\\">" + chessSquares(chess, color) + "</section>" +
          "</main>";
	        const nameInput = root.querySelector("[data-chess=name]");
	        nameInput?.addEventListener("input", () => {
	          state.nameDraft = nameInput.value;
	        });
	        root.querySelector("[data-chess=join]")?.addEventListener("click", () => {
	          const name = nameInput?.value || "";
	          const inputBoard = root.querySelector("[data-chess=board-no]")?.value || "";
	          joinChess(name, inputBoard).catch((err) => {
            state.last = err instanceof Error ? err.message : String(err);
            renderChess();
          });
        });
        root.querySelector("[data-chess=resign]")?.addEventListener("click", () => {
          resignChess().catch((err) => {
            state.last = err instanceof Error ? err.message : String(err);
            renderChess();
          });
        });
        root.querySelectorAll("[data-square]").forEach((button) => {
          button.addEventListener("click", () => selectChessSquare(button.getAttribute("data-square") || ""));
        });
        root.dataset.chessReady = state.baseRevision ? "true" : "false";
	        if (chess.result?.status) root.dataset.complete = "true";
	      }

      function renderTypeRace() {
        const race = normalizeTypeRace(state.typeRace);
        const participant = state.lease?.participantId || "";
        const player = race.players[participant] || null;
        const alias = state.alias || player?.name || anonymousAlias();
        const prompt = promptHtml(race.text, state.typeDraft || player?.typed || "");
        const status = race.result?.status
          ? "winner " + (race.players[race.result.winner]?.name || race.result.winner)
          : race.status === "active"
            ? "racing"
            : "waiting";
        const joined = Boolean(player);
        const inputDisabled = !joined || race.status !== "active" || race.result?.status ? " disabled" : "";
        root.innerHTML =
          "<main class=\\"type-race\\" data-type=\\"app\\">" +
          "<section class=\\"type-panel\\">" +
          "<header class=\\"type-head\\">" +
          "<div><h2 data-type=\\"title\\">Race " + escapeHtml(shortBoard(race.race || state.demo?.raceNo || "")) + "</h2>" +
          "<p class=\\"type-status\\" data-type=\\"status\\">" + escapeHtml(status) + "</p></div>" +
          "<span class=\\"type-badge\\">anonymous</span>" +
          "</header>" +
          "<dl class=\\"type-score\\">" +
          row("You", alias, "alias") +
          row("Runner", participant, "participant") +
          row("Progress", String(player?.percent || 0) + "%", "progress") +
          row("Mistakes", String(player?.mistakes || 0), "mistakes") +
          row("Winner", race.result?.winner || "", "winner") +
          row("Delivery", state.delivery, "delivery") +
          "</dl>" +
          "<div class=\\"type-actions\\">" +
          "<button data-type=\\"join\\">" + escapeHtml(joined ? "Joined" : "Join") + "</button>" +
          "<button data-type=\\"clear\\">Clear</button>" +
          "</div>" +
          "<dl class=\\"type-proof-only\\">" +
          row("Revision", String(state.baseRevision || 0), "revision") +
          row("Actions", String(state.actions), "actions") +
          row("Reads", String(state.readbacks), "readbacks") +
          row("Denied", String(state.denied), "denied") +
          row("Last", state.last) +
          "</dl>" +
          "</section>" +
          "<section class=\\"type-track\\">" +
          "<p class=\\"type-prompt\\" data-type=\\"prompt\\">" + prompt + "</p>" +
          "<textarea class=\\"type-input\\" data-type=\\"input\\" spellcheck=\\"false\\" autocomplete=\\"off\\"" + inputDisabled + ">" + escapeHtml(state.typeDraft || player?.typed || "") + "</textarea>" +
          "<section class=\\"type-runners\\" data-type=\\"runners\\">" + typeRunnerRows(race) + "</section>" +
          "</section>" +
          "</main>";
        const input = root.querySelector("[data-type=input]");
        input?.addEventListener("input", () => {
          state.typeDraft = input.value;
          queueTypeProgress();
        });
        root.querySelector("[data-type=join]")?.addEventListener("click", () => {
          joinTypeRace(alias).catch((err) => {
            state.last = err instanceof Error ? err.message : String(err);
            renderTypeRace();
          });
        });
        root.querySelector("[data-type=clear]")?.addEventListener("click", () => {
          state.typeDraft = "";
          renderTypeRace();
        });
        root.dataset.typeReady = state.baseRevision ? "true" : "false";
        if (race.result?.status) root.dataset.complete = "true";
      }

	      function chessNameFocused() {
	        return document.activeElement?.getAttribute("data-chess") === "name";
	      }

      function typeInputFocused() {
        return document.activeElement?.getAttribute("data-type") === "input";
      }

      function renderVisual() {
        const status = state.failed ? "failed" : state.complete ? "complete" : "running";
        root.innerHTML =
          "<h2 data-demo=\\"title\\">Diagram decision</h2>" +
          "<p data-demo=\\"status\\">" + escapeHtml(status) + "</p>" +
          "<button data-demo=\\"submit\\">Submit</button>" +
          "<dl>" +
          row("Result", state.demo?.visualKey || "", "item") +
          row("Selected", state.choice || state.demo?.choice || "", "selected") +
          row("Denied", String(state.denied), "denied") +
          row("Last", state.last) +
          "</dl>";
        root.querySelector("[data-demo=submit]")?.addEventListener("click", () => {
          runVisual().catch((err) => {
            state.last = err instanceof Error ? err.message : String(err);
            state.failed = true;
            state.complete = true;
            renderVisual();
          });
        });
        if (state.complete) root.dataset.complete = "true";
        if (state.itemKey !== "") root.dataset.itemKey = state.itemKey;
      }

      function row(label, value, proofName) {
        const proof = proofName ? " data-demo=\\\"" + escapeHtml(proofName) + "\\\"" : "";
        return "<div><dt>" + escapeHtml(label) + "</dt><dd" + proof + ">" + escapeHtml(value) + "</dd></div>";
      }

      function normalizeBoard(value) {
        const src = value && typeof value === "object" ? value : {};
        const cells = src.cells && typeof src.cells === "object" ? src.cells : {};
        return {
          turn: typeof src.turn === "string" ? src.turn : "",
          cells,
          winner: typeof src.winner === "string" ? src.winner : "",
        };
      }

      function normalizeChess(value) {
        const src = value && typeof value === "object" ? value : {};
        const players = src.players && typeof src.players === "object" ? src.players : {};
        return {
          kind: "tinkabot.chessGame.v1",
          board: typeof src.board === "string" ? src.board : String(state.demo?.boardNo || ""),
          status: typeof src.status === "string" ? src.status : "waiting",
          fen: typeof src.fen === "string" ? src.fen : "start",
          turn: typeof src.turn === "string" ? src.turn : "white",
          players: {
            white: normalizePlayer(players.white),
            black: normalizePlayer(players.black),
          },
          result: src.result && typeof src.result === "object" ? src.result : null,
          history: Array.isArray(src.history) ? src.history : [],
        };
      }

      function normalizeTypeRace(value) {
        const src = value && typeof value === "object" ? value : {};
        const players = src.players && typeof src.players === "object" ? src.players : {};
        const out = {};
        for (const id of Object.keys(players)) {
          out[id] = normalizeTypePlayer(players[id]);
        }
        return {
          kind: "tinkabot.typeRace.v1",
          race: typeof src.race === "string" ? src.race : String(state.demo?.raceNo || ""),
          status: typeof src.status === "string" ? src.status : "waiting",
          text: typeof src.text === "string" ? src.text : "",
          players: out,
          result: src.result && typeof src.result === "object" ? src.result : null,
          events: Array.isArray(src.events) ? src.events : [],
        };
      }

      function normalizeTypePlayer(value) {
        const src = value && typeof value === "object" ? value : {};
        return {
          participantId: typeof src.participantId === "string" ? src.participantId : "",
          name: typeof src.name === "string" ? src.name : "",
          typed: typeof src.typed === "string" ? src.typed : "",
          progress: Number(src.progress || 0),
          percent: Number(src.percent || 0),
          mistakes: Number(src.mistakes || 0),
          finishedAt: typeof src.finishedAt === "string" ? src.finishedAt : "",
        };
      }

      function normalizePlayer(value) {
        if (!value || typeof value !== "object") return null;
        return {
          participantId: typeof value.participantId === "string" ? value.participantId : "",
          name: typeof value.name === "string" ? value.name : "",
        };
      }

      function promptHtml(text, typed) {
        const src = String(text || "");
        const got = String(typed || "");
        return Array.from(src).map((ch, i) => {
          const mark = got.length <= i ? "" : got[i] === ch ? " type-ok" : " type-bad";
          return "<span class=\\"" + mark.trim() + "\\">" + escapeHtml(ch) + "</span>";
        }).join("");
      }

      function typeRunnerRows(race) {
        const players = Object.values(race.players).sort((a, b) => {
          if (b.progress !== a.progress) return b.progress - a.progress;
          return a.name.localeCompare(b.name);
        });
        if (players.length === 0) {
          return "<p data-type=\\"empty\\">Waiting for anonymous runners.</p>";
        }
        return players.map((p) =>
          "<article class=\\"type-runner\\" data-runner=\\"" + escapeHtml(p.participantId) + "\\">" +
          "<div class=\\"type-runner-head\\"><span>" + escapeHtml(p.name || p.participantId) + "</span><span>" + escapeHtml(String(p.percent || 0)) + "%</span></div>" +
          "<div class=\\"type-meter\\"><span style=\\"width:" + escapeHtml(String(Math.max(0, Math.min(100, p.percent || 0)))) + "%\\"></span></div>" +
          "<small>" + escapeHtml(String(p.progress || 0)) + " chars, " + escapeHtml(String(p.mistakes || 0)) + " mistakes</small>" +
          "</article>"
        ).join("");
      }

      function chessSquares(chess, color) {
        const pieces = fenPieces(chess.fen);
        const files = ["a", "b", "c", "d", "e", "f", "g", "h"];
        const ranks = color === "black" ? [1, 2, 3, 4, 5, 6, 7, 8] : [8, 7, 6, 5, 4, 3, 2, 1];
        const viewFiles = color === "black" ? [...files].reverse() : files;
        return ranks.flatMap((rank) => viewFiles.map((file) => {
          const square = file + rank;
          const dark = (files.indexOf(file) + rank) % 2 === 1;
          const piece = pieces[square] || "";
          const own = piece && piece.color === color ? "true" : "false";
          const selected = state.selectedSquare === square ? " chess-selected" : "";
          const last = isLastSquare(chess, square) ? " chess-last" : "";
          const body = piece ? "<span class=\\"chess-piece chess-" + piece.color + "-piece\\">" + escapeHtml(piece.symbol) + "</span>" : "";
          return "<button class=\\"chess-square " + (dark ? "chess-dark" : "chess-light") + selected + last + "\\" data-square=\\"" + square + "\\" data-own=\\"" + own + "\\">" + body + "</button>";
        })).join("");
      }

      function fenPieces(fen) {
        const symbols = {
          p: "♟", r: "♜", n: "♞", b: "♝", q: "♛", k: "♚",
          P: "♙", R: "♖", N: "♘", B: "♗", Q: "♕", K: "♔",
        };
        const out = {};
        const rows = String(fen || "").split(" ")[0].split("/");
        for (let y = 0; y < rows.length; y++) {
          let file = 0;
          for (const ch of rows[y]) {
            const n = Number(ch);
            if (Number.isInteger(n) && n > 0) {
              file += n;
              continue;
            }
            const square = "abcdefgh"[file] + String(8 - y);
            out[square] = { symbol: symbols[ch] || "", color: ch === ch.toUpperCase() ? "white" : "black" };
            file += 1;
          }
        }
        return out;
      }

      function selectChessSquare(square) {
        const chess = normalizeChess(state.chess);
        const color = chessColor(chess, state.lease?.participantId || "");
        if (!color || chess.turn !== color || chess.result?.status) return;
        if (!state.selectedSquare) {
          state.selectedSquare = square;
          renderChess();
          return;
        }
        const from = state.selectedSquare;
        state.selectedSquare = "";
        if (from === square) {
          renderChess();
          return;
        }
        moveChess(from, square).catch((err) => {
          state.last = err instanceof Error ? err.message : String(err);
          renderChess();
        });
      }

      function chessColor(chess, participantId) {
        if (chess.players.white?.participantId === participantId) return "white";
        if (chess.players.black?.participantId === participantId) return "black";
        return "";
      }

      function playerLabel(player) {
        return player ? player.name + " (" + player.participantId + ")" : "open";
      }

      function shortBoard(boardNo) {
        const value = String(boardNo || "");
        return value.length > 18 ? value.slice(0, 10) + "..." + value.slice(-5) : value;
      }

      function isLastSquare(chess, square) {
        const last = chess.history[chess.history.length - 1];
        return last && (last.from === square || last.to === square);
      }

      function chessSnapshot() {
        const chess = normalizeChess(state.chess);
        return {
          participant: state.lease?.participantId || "",
          revision: state.baseRevision,
          board: chess.board,
          status: chess.status,
          turn: chess.turn,
          color: chessColor(chess, state.lease?.participantId || ""),
          players: chess.players,
          fen: chess.fen,
          result: chess.result,
          history: chess.history,
          actions: state.actions,
          readbacks: state.readbacks,
          pushes: state.pushes,
          delivery: state.delivery,
          denied: state.denied,
          last: state.last,
        };
      }

      function typeRaceSnapshot() {
        const race = normalizeTypeRace(state.typeRace);
        const participant = state.lease?.participantId || "";
        return {
          participant,
          revision: state.baseRevision,
          race: race.race,
          status: race.status,
          text: race.text,
          player: race.players[participant] || null,
          players: race.players,
          result: race.result,
          actions: state.actions,
          readbacks: state.readbacks,
          pushes: state.pushes,
          delivery: state.delivery,
          denied: state.denied,
          last: state.last,
        };
      }

      function anonymousAlias() {
        const id = String(state.lease?.participantId || "anon");
        return "Anonymous " + id.slice(-5).toUpperCase();
      }

      function boardCellsText(board) {
        return ["a1", "a2", "a3", "b1", "b2", "b3", "c1", "c2", "c3"]
          .map((cell) => cell + ":" + (board.cells[cell] || ""))
          .join(",");
      }

      function boardSnapshot() {
        return {
          participant: state.lease?.participantId || "",
          revision: state.baseRevision,
          turn: state.board.turn || "",
          winner: state.board.winner || "",
          cells: { ...state.board.cells },
          actions: state.actions,
          readbacks: state.readbacks,
          pushes: state.pushes,
          delivery: state.delivery,
          denied: state.denied,
          last: state.last,
        };
      }

      function escapeHtml(value) {
        return String(value).replace(/[&<>"']/g, (ch) => ({
          "&": "&amp;",
          "<": "&lt;",
          ">": "&gt;",
          '"': "&quot;",
          "'": "&#39;",
        }[ch]));
      }

      function sleep(ms) {
        return new Promise((resolve) => setTimeout(resolve, ms));
      }

      parent.postMessage({ type: "content.ready" }, "*");
    </script>
  </body>
</html>`;
}
