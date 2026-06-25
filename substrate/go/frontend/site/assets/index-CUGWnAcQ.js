var Yi=(r,e)=>()=>(e||r((e={exports:{}}).exports,e),e.exports);var Do=Yi((zo,at)=>{(function(){const e=document.createElement("link").relList;if(e&&e.supports&&e.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))s(i);new MutationObserver(i=>{for(const n of i)if(n.type==="childList")for(const c of n.addedNodes)c.tagName==="LINK"&&c.rel==="modulepreload"&&s(c)}).observe(document,{childList:!0,subtree:!0});function t(i){const n={};return i.integrity&&(n.integrity=i.integrity),i.referrerPolicy&&(n.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?n.credentials="include":i.crossOrigin==="anonymous"?n.credentials="omit":n.credentials="same-origin",n}function s(i){if(i.ep)return;i.ep=!0;const n=t(i);fetch(i.href,n)}})();class Xi extends Error{constructor(e,t,s={}){super(t),this.kind=e,this.details=s}kind;details;layer="FrontendIsolation"}const Zi="allow-scripts",Qi=new Set(["allow","allowresponses","bearer","cred","credential","credentials","deny","headers","jwt","nats","nkey","permission","permissions","publish","reply","replysubject","secret","seed","subject","subjects","subscribe","password","token","tokens"]);function en(r="generated artifact"){return{title:r,sandbox:tn(Zi),referrerPolicy:"no-referrer"}}function tn(r){const e=new Set(r.split(/\s+/).filter(Boolean));if(e.size!==1||!e.has("allow-scripts"))throw fe("FrameSandboxDenied","Generated content requires script-only sandbox",{value:r});return[...e].sort().join(" ")}function sn(r){return{...r,nonce:r.nonce??on()}}function rn(r,e){return r.sessions.includes(e)}function nn(r,e,t,s){if(e!==t)throw fe("FrameLeaseDenied","Message source does not match leased frame",{frameId:r.frameId});Ke(s);const i=an(s);if(i.nonce!==r.nonce)throw fe("FrameLeaseDenied","Message nonce does not match frame lease",{frameId:r.frameId});if(i.frameId!==r.frameId)throw fe("FrameLeaseDenied","Message frame id does not match lease",{frameId:r.frameId,actual:i.frameId});if(i.artifactRevision!==r.artifactRevision)throw fe("FrameLeaseDenied","Message artifact revision is stale",{expected:r.artifactRevision,actual:i.artifactRevision});if(i.expectedRevision!==r.artifactRevision)throw fe("FrameLeaseDenied","Message expected revision is stale",{expected:r.artifactRevision,actual:i.expectedRevision});if(i.schemaRevision!==r.schemaRevision)throw fe("FrameLeaseDenied","Message schema revision is stale",{expected:r.schemaRevision,actual:i.schemaRevision});if(!r.commands.includes(i.command))throw fe("FrameCapabilityDenied","Command is not allowed for frame lease",{command:i.command});if(i.sessionId!==void 0&&!rn(r,i.sessionId))throw fe("FrameScopeEscape","Session is not in frame lease observation scope",{sessionId:i.sessionId});if(r.appId!==void 0&&i.appId!==r.appId)throw fe("FrameScopeEscape","App is not in frame lease scope",{expected:r.appId,actual:i.appId});if(r.participantId!==void 0&&i.participantId!==r.participantId)throw fe("FrameScopeEscape","Participant is not in frame lease scope",{expected:r.participantId,actual:i.participantId});return{kind:"browser.command_intent",type:"content.intent",command:i.command,commandId:i.commandId,expectedRevision:i.expectedRevision,payload:i.payload,context:{sessionId:r.sessionId,capabilityId:r.capabilityId,artifactId:r.artifactId,artifactRevision:r.artifactRevision,frameId:r.frameId,appId:r.appId,participantId:r.participantId,chain:r.chain}}}function Ke(r,e=[],t=new WeakSet){if(ni(r)&&!t.has(r)){if(t.add(r),Array.isArray(r)){r.forEach((s,i)=>Ke(s,[...e,String(i)],t));return}if(r instanceof Map){let s=0;for(const[i,n]of r){const c=typeof i=="string"?i:String(s);Rr(c,[...e,c]),typeof i!="string"&&Ke(i,[...e,`$key${s}`],t),Ke(n,[...e,c],t),s+=1}return}if(r instanceof Set){let s=0;for(const i of r)Ke(i,[...e,String(s)],t),s+=1;return}for(const[s,i]of Object.entries(r))Rr(s,[...e,s]),Ke(i,[...e,s],t)}}function an(r){if(!cn(r))throw fe("FrameMessageInvalid","Message must be an object");if(r.type!=="content.intent")throw fe("FrameMessageInvalid","Message type is not supported",{type:r.type});for(const e of["command","commandId","expectedRevision","nonce","frameId","artifactRevision","schemaRevision"])if(typeof r[e]!="string"||r[e].length===0)throw fe("FrameMessageInvalid","Message field is required",{field:e});return r}function fe(r,e,t={}){return new Xi(r,e,t)}function on(){if(typeof crypto.randomUUID=="function")return crypto.randomUUID();const r=new Uint8Array(16);crypto.getRandomValues(r),r[6]=r[6]&15|64,r[8]=r[8]&63|128;const e=[...r].map(t=>t.toString(16).padStart(2,"0")).join("");return`${e.slice(0,8)}-${e.slice(8,12)}-${e.slice(12,16)}-${e.slice(16,20)}-${e.slice(20)}`}function Rr(r,e){const t=r.toLowerCase().replace(/[-_]/g,"");if([...Qi].some(s=>t.includes(s)))throw fe("FrameCapabilityDenied","Generated content cannot send raw authority",{path:e.join(".")})}function ni(r){return typeof r=="object"&&r!==null}function cn(r){return ni(r)}function un(){return URL.createObjectURL(new Blob([hn()],{type:"text/html"}))}function hn(){return`<!doctype html>
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
        const proof = proofName ? " data-demo=\\"" + escapeHtml(proofName) + "\\"" : "";
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
    <\/script>
  </body>
</html>`}const Ie=new Uint8Array(0),et=new TextEncoder,ke=new TextDecoder;function ln(...r){let e=0;for(let i=0;i<r.length;i++)e+=r[i].length;const t=new Uint8Array(e);let s=0;for(let i=0;i<r.length;i++)t.set(r[i],s),s+=r[i].length;return t}function Tt(...r){const e=[];for(let t=0;t<r.length;t++)e.push(et.encode(r[t]));return e.length===0?Ie:e.length===1?e[0]:ln(...e)}function jr(r){return!r||r.length===0?"":ke.decode(r)}const Or="0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ",Cr=36,dn=0xcfd41b9100000,Nr=33,fn=333,Mr=22;function pn(r){for(let e=0;e<r.length;e++)r[e]=Math.floor(Math.random()*255)}function mn(r){globalThis?.crypto?.getRandomValues?globalThis.crypto.getRandomValues(r):pn(r)}class bn{buf;seq;inc;inited;constructor(){this.buf=new Uint8Array(Mr),this.inited=!1}init(){this.inited=!0,this.setPre(),this.initSeqAndInc(),this.fillSeq()}initSeqAndInc(){this.seq=Math.floor(Math.random()*dn),this.inc=Math.floor(Math.random()*(fn-Nr)+Nr)}setPre(){const e=new Uint8Array(12);mn(e);for(let t=0;t<12;t++){const s=e[t]%36;this.buf[t]=Or.charCodeAt(s)}}fillSeq(){let e=this.seq;for(let t=Mr-1;t>=12;t--)this.buf[t]=Or.charCodeAt(e%Cr),e=Math.floor(e/Cr)}next(){return this.inited||this.init(),this.seq+=this.inc,this.seq>0xcfd41b9100000&&(this.setPre(),this.initSeqAndInc()),this.fillSeq(),String.fromCharCode.apply(String,this.buf)}reset(){this.init()}}const Ze=new bn;var De;(function(r){r.Disconnect="disconnect",r.Reconnect="reconnect",r.Update="update",r.LDM="ldm",r.Error="error"})(De||(De={}));var gt;(function(r){r.Reconnecting="reconnecting",r.PingTimer="pingTimer",r.StaleConnection="staleConnection",r.ClientInitiatedReconnect="client initiated reconnect"})(gt||(gt={}));var E;(function(r){r.ApiError="BAD API",r.BadAuthentication="BAD_AUTHENTICATION",r.BadCreds="BAD_CREDS",r.BadHeader="BAD_HEADER",r.BadJson="BAD_JSON",r.BadPayload="BAD_PAYLOAD",r.BadSubject="BAD_SUBJECT",r.Cancelled="CANCELLED",r.ConnectionClosed="CONNECTION_CLOSED",r.ConnectionDraining="CONNECTION_DRAINING",r.ConnectionRefused="CONNECTION_REFUSED",r.ConnectionTimeout="CONNECTION_TIMEOUT",r.Disconnect="DISCONNECT",r.InvalidOption="INVALID_OPTION",r.InvalidPayload="INVALID_PAYLOAD",r.MaxPayloadExceeded="MAX_PAYLOAD_EXCEEDED",r.NoResponders="503",r.NotFunction="NOT_FUNC",r.RequestError="REQUEST_ERROR",r.ServerOptionNotAvailable="SERVER_OPT_NA",r.SubClosed="SUB_CLOSED",r.SubDraining="SUB_DRAINING",r.Timeout="TIMEOUT",r.Tls="TLS",r.Unknown="UNKNOWN_ERROR",r.WssRequired="WSS_REQUIRED",r.JetStreamInvalidAck="JESTREAM_INVALID_ACK",r.JetStream404NoMessages="404",r.JetStream408RequestTimeout="408",r.JetStream409MaxAckPendingExceeded="409",r.JetStream409="409",r.JetStreamNotEnabled="503",r.JetStreamIdleHeartBeat="IDLE_HEARTBEAT",r.AuthorizationViolation="AUTHORIZATION_VIOLATION",r.AuthenticationExpired="AUTHENTICATION_EXPIRED",r.ProtocolError="NATS_PROTOCOL_ERR",r.PermissionsViolation="PERMISSIONS_VIOLATION",r.AuthenticationTimeout="AUTHENTICATION_TIMEOUT",r.AccountExpired="ACCOUNT_EXPIRED"})(E||(E={}));function gn(r){return typeof r.code=="string"}class ai{messages;constructor(){this.messages=new Map,this.messages.set(E.InvalidPayload,"Invalid payload type - payloads can be 'binary', 'string', or 'json'"),this.messages.set(E.BadJson,"Bad JSON"),this.messages.set(E.WssRequired,"TLS is required, therefore a secure websocket connection is also required")}static getMessage(e){return yn.getMessage(e)}getMessage(e){return this.messages.get(e)||e}}const yn=new ai;class j extends Error{name;message;code;permissionContext;chainedError;api_error;constructor(e,t,s){super(e),this.name="NatsError",this.message=e,this.code=t,this.chainedError=s}static errorForCode(e,t){const s=ai.getMessage(e);return new j(s,e,t)}isAuthError(){return this.code===E.AuthenticationExpired||this.code===E.AuthorizationViolation||this.code===E.AccountExpired}isAuthTimeout(){return this.code===E.AuthenticationTimeout}isPermissionError(){return this.code===E.PermissionsViolation}isProtocolError(){return this.code===E.ProtocolError}isJetStreamError(){return this.api_error!==void 0}jsError(){return this.api_error?this.api_error:null}}var xe;(function(r){r[r.Exact=0]="Exact",r[r.CanonicalMIME=1]="CanonicalMIME",r[r.IgnoreCase=2]="IgnoreCase"})(xe||(xe={}));var Te;(function(r){r.Timer="timer",r.Count="count",r.JitterTimer="jitterTimer",r.SentinelMsg="sentinelMsg"})(Te||(Te={}));var $t;(function(r){r.STATS="io.nats.micro.v1.stats_response",r.INFO="io.nats.micro.v1.info_response",r.PING="io.nats.micro.v1.ping_response"})($t||($t={}));const ds="Nats-Service-Error",fs="Nats-Service-Error-Code";class ps extends Error{code;constructor(e,t){super(t),this.code=e}static isServiceError(e){return ps.toServiceError(e)!==null}static toServiceError(e){const t=e?.headers?.get(fs)||"";if(t!==""){const s=parseInt(t)||400,i=e?.headers?.get(ds)||"";return new ps(s,i.length?i:t)}return null}}function He(r=""){if(r=r||"_INBOX",typeof r!="string")throw new Error("prefix must be a string");return r.split(".").forEach(e=>{if(e==="*"||e===">")throw new Error(`inbox prefixes cannot have wildcards '${r}'`)}),`${r}.${Ze.next()}`}const Gs="127.0.0.1";var Ve;(function(r){r.PING="PING",r.STATS="STATS",r.INFO="INFO"})(Ve||(Ve={}));function _s(r,...e){for(let t=0;t<e.length;t++){const s=e[t];Object.keys(s).forEach(function(i){r[i]=s[i]})}return r}function is(r){return ke.decode(r).replace(/\n/g,"␊").replace(/\r/g,"␍")}function vt(r,e=!0){const t=e?j.errorForCode(E.Timeout):null;let s,i;const n=new Promise((c,a)=>{s={cancel:()=>{i&&clearTimeout(i)}},i=setTimeout(()=>{a(t===null?j.errorForCode(E.Timeout):t)},r)});return Object.assign(n,s)}function Et(r=0){let e;const t=new Promise(s=>{const i=setTimeout(()=>{s()},r);e={cancel:()=>{i&&clearTimeout(i)}}});return Object.assign(t,e)}function W(){let r={};const e=new Promise((t,s)=>{r={resolve:t,reject:s}});return Object.assign(e,r)}function oi(r){for(let e=r.length-1;e>0;e--){const t=Math.floor(Math.random()*(e+1));[r[e],r[t]]=[r[t],r[e]]}return r}function wn(r){return r===0?0:Math.floor(r/2+Math.random()*r)}function cr(r=[0,250,250,500,500,3e3,5e3]){Array.isArray(r)||(r=[0,250,250,500,500,3e3,5e3]);const e=r.length-1;return{backoff(t){return wn(t>e?r[e]:r[t])}}}function V(r){return r*1e6}function ur(r){return Math.floor(r/1e6)}function Tr(r){let s=!0;const i=new Array(r.length);for(let n=0;n<r.length;n++){let c=r.charCodeAt(n);if(c===58||c<33||c>126)throw new j(`'${r[n]}' is not a valid character for a header key`,E.BadHeader);s&&97<=c&&c<=122?c-=32:!s&&65<=c&&c<=90&&(c+=32),i[n]=c,s=c==45}return String.fromCharCode(...i)}function ze(r=0,e=""){if(r===0&&e!==""||r>0&&e==="")throw new Error("setting status requires both code and description");return new We(r,e)}const Fs="NATS/1.0";class We{_code;headers;_description;constructor(e=0,t=""){this._code=e,this._description=t,this.headers=new Map}[Symbol.iterator](){return this.headers.entries()}size(){return this.headers.size}equals(e){if(e&&this.headers.size===e.headers.size&&this._code===e._code){for(const[t,s]of this.headers){const i=e.values(t);if(s.length!==i.length)return!1;const n=[...s].sort(),c=[...i].sort();for(let a=0;a<n.length;a++)if(n[a]!==c[a])return!1}return!0}return!1}static decode(e){const t=new We,i=ke.decode(e).split(`\r
`),n=i[0];if(n!==Fs){let c=n.replace(Fs,"").trim();if(c.length>0){t._code=parseInt(c,10),isNaN(t._code)&&(t._code=0);const a=t._code.toString();c=c.replace(a,""),t._description=c.trim()}}return i.length>=1&&i.slice(1).map(c=>{if(c){const a=c.indexOf(":");if(a>-1){const d=c.slice(0,a),m=c.slice(a+1).trim();t.append(d,m)}}}),t}toString(){if(this.headers.size===0&&this._code===0)return"";let e=Fs;this._code>0&&this._description!==""&&(e+=` ${this._code} ${this._description}`);for(const[t,s]of this.headers)for(let i=0;i<s.length;i++)e=`${e}\r
${t}: ${s[i]}`;return`${e}\r
\r
`}encode(){return et.encode(this.toString())}static validHeaderValue(e){if(/[\r\n]/.test(e))throw new j("invalid header value - \\r and \\n are not allowed.",E.BadHeader);return e.trim()}keys(){const e=[];for(const t of this.headers.keys())e.push(t);return e}findKeys(e,t=xe.Exact){const s=this.keys();switch(t){case xe.Exact:return s.filter(i=>i===e);case xe.CanonicalMIME:return e=Tr(e),s.filter(i=>i===e);default:{const i=e.toLowerCase();return s.filter(n=>i===n.toLowerCase())}}}get(e,t=xe.Exact){const s=this.findKeys(e,t);if(s.length){const i=this.headers.get(s[0]);if(i)return Array.isArray(i)?i[0]:i}return""}last(e,t=xe.Exact){const s=this.findKeys(e,t);if(s.length){const i=this.headers.get(s[0]);if(i)return Array.isArray(i)?i[i.length-1]:i}return""}has(e,t=xe.Exact){return this.findKeys(e,t).length>0}set(e,t,s=xe.Exact){this.delete(e,s),this.append(e,t,s)}append(e,t,s=xe.Exact){const i=Tr(e);s===xe.CanonicalMIME&&(e=i);const n=this.findKeys(e,s);e=n.length>0?n[0]:e;const c=We.validHeaderValue(t);let a=this.headers.get(e);a||(a=[],this.headers.set(e,a)),a.push(c)}values(e,t=xe.Exact){const s=[];return this.findKeys(e,t).forEach(n=>{const c=this.headers.get(n);c&&s.push(...c)}),s}delete(e,t=xe.Exact){this.findKeys(e,t).forEach(i=>{this.headers.delete(i)})}get hasError(){return this._code>=300}get status(){return`${this._code} ${this._description}`.trim()}toRecord(){const e={};return this.keys().forEach(t=>{e[t]=this.values(t)}),e}get code(){return this._code}get description(){return this._description}static fromRecord(e){const t=new We;for(const s in e)t.headers.set(s,e[s]);return t}}function $r(){return{encode(r){return et.encode(r)},decode(r){return ke.decode(r)}}}function $e(r){return{encode(e){try{return e===void 0&&(e=null),et.encode(JSON.stringify(e))}catch(t){throw j.errorForCode(E.BadJson,t)}},decode(e){try{return JSON.parse(ke.decode(e),r)}catch(t){throw j.errorForCode(E.BadJson,t)}}}}function ci(r){return r&&r.data.length===0&&r.headers?.code===503?j.errorForCode(E.NoResponders):null}class ui{_headers;_msg;_rdata;_reply;_subject;publisher;static jc;constructor(e,t,s){this._msg=e,this._rdata=t,this.publisher=s}get subject(){return this._subject?this._subject:(this._subject=ke.decode(this._msg.subject),this._subject)}get reply(){return this._reply?this._reply:(this._reply=ke.decode(this._msg.reply),this._reply)}get sid(){return this._msg.sid}get headers(){if(this._msg.hdr>-1&&!this._headers){const e=this._rdata.subarray(0,this._msg.hdr);this._headers=We.decode(e)}return this._headers}get data(){return this._rdata?this._msg.hdr>-1?this._rdata.subarray(this._msg.hdr):this._rdata:new Uint8Array(0)}respond(e=Ie,t){return this.reply?(this.publisher.publish(this.reply,e,t),!0):!1}size(){const e=this._msg.subject.length,t=this._msg.reply?.length||0,s=this._msg.size===-1?0:this._msg.size;return e+t+s}json(e){return $e(e).decode(this.data)}string(){return ke.decode(this.data)}requestInfo(){const e=this.headers?.get("Nats-Request-Info");return e?JSON.parse(e,function(t,s){return(t==="start"||t==="stop")&&s!==""?new Date(Date.parse(s)):s}):null}}function yt(r){return vs("durable",r)}function pe(r){return vs("stream",r)}function vs(r,e=""){if(e==="")throw Error(`${r} name required`);return[".","*",">","/","\\"," ","	",`
`,"\r"].forEach(s=>{if(e.indexOf(s)!==-1){switch(s){case`
`:s="\\n";break;case"\r":s="\\r";break;case"	":s="\\t";break}throw Error(`invalid ${r} name - ${r} name cannot contain '${s}'`)}}),""}function Nt(r,e=""){if(e==="")throw Error(`${r} name required`);const t=xn(e);if(t.length)throw new Error(`invalid ${r} name - ${r} name ${t}`)}function xn(r=""){if(r==="")throw Error("name required");const e=/^[-\w]+$/g;if(r.match(e)===null){for(const s of r.split(""))if(s.match(e)===null)return`cannot contain '${s}'`}return""}function Vs(r){if(r.data.length>0)return!1;const e=r.headers;return e?e.code>=100&&e.code<200:!1}function Ws(r){return Vs(r)&&r.headers?.description==="Idle Heartbeat"}function _n(r,e,t){const s=ze(r,e),i={hdr:1,sid:0,size:0},n=new ui(i,Ie,{});return n._headers=s,n._subject=t,n}function wt(r){if(r.data.length!==0)return null;const e=r.headers;return e?hi(e.code,e.description):null}var Re;(function(r){r.MaxBatchExceeded="exceeded maxrequestbatch of",r.MaxExpiresExceeded="exceeded maxrequestexpires of",r.MaxBytesExceeded="exceeded maxrequestmaxbytes of",r.MaxMessageSizeExceeded="message size exceeds maxbytes",r.PushConsumer="consumer is push based",r.MaxWaitingExceeded="exceeded maxwaiting",r.IdleHeartbeatMissed="idle heartbeats missed",r.ConsumerDeleted="consumer deleted"})(Re||(Re={}));function vn(r){return r.code!==E.JetStream409?!1:[Re.MaxBatchExceeded,Re.MaxExpiresExceeded,Re.MaxBytesExceeded,Re.MaxMessageSizeExceeded,Re.PushConsumer,Re.IdleHeartbeatMissed,Re.ConsumerDeleted].find(t=>r.message.indexOf(t)!==-1)!==void 0}function hi(r,e=""){if(r<300)return null;switch(e=e.toLowerCase(),r){case 404:return new j(e,E.JetStream404NoMessages);case 408:return new j(e,E.JetStream408RequestTimeout);case 409:{const t=e.startsWith(Re.IdleHeartbeatMissed)?E.JetStreamIdleHeartBeat:E.JetStream409;return new j(e,t)}case 503:return j.errorForCode(E.JetStreamNotEnabled,new Error(e));default:return e===""&&(e=E.Unknown),new j(e,`${r}`)}}class oe{inflight;processed;received;noIterator;iterClosed;done;signal;yields;filtered;pendingFiltered;ingestionFilterFn;protocolFilterFn;dispatchedFn;ctx;_data;err;time;yielding;constructor(){this.inflight=0,this.filtered=0,this.pendingFiltered=0,this.processed=0,this.received=0,this.noIterator=!1,this.done=!1,this.signal=W(),this.yields=[],this.iterClosed=W(),this.time=0,this.yielding=!1}[Symbol.asyncIterator](){return this.iterate()}push(e){if(this.done)return;if(typeof e=="function"){this.yields.push(e),this.signal.resolve();return}const{ingest:t,protocol:s}=this.ingestionFilterFn?this.ingestionFilterFn(e,this.ctx||this):{ingest:!0,protocol:!1};t&&(s&&(this.filtered++,this.pendingFiltered++),this.yields.push(e),this.signal.resolve())}async*iterate(){if(this.noIterator)throw new j("unsupported iterator",E.ApiError);if(this.yielding)throw new j("already yielding",E.ApiError);this.yielding=!0;try{for(;;){if(this.yields.length===0&&await this.signal,this.err)throw this.err;const e=this.yields;this.inflight=e.length,this.yields=[];for(let t=0;t<e.length;t++){if(typeof e[t]=="function"){const i=e[t];try{i()}catch(n){throw n}if(this.err)throw this.err;continue}if(this.protocolFilterFn?this.protocolFilterFn(e[t]):!0){this.processed++;const i=Date.now();yield e[t],this.time=Date.now()-i,this.dispatchedFn&&e[t]&&this.dispatchedFn(e[t])}else this.pendingFiltered--;this.inflight--}if(this.done)break;this.yields.length===0&&(e.length=0,this.yields=e,this.signal=W())}}finally{this.stop()}}stop(e){this.done||(this.err=e,this.done=!0,this.signal.resolve(),this.iterClosed.resolve(e))}getProcessed(){return this.noIterator?this.received:this.processed}getPending(){return this.yields.length+this.inflight-this.pendingFiltered}getReceived(){return this.received-this.filtered}}class hr{interval;maxOut;cancelAfter;timer;autoCancelTimer;last;missed;count;callback;constructor(e,t,s={maxOut:2}){this.interval=e,this.maxOut=s?.maxOut||2,this.cancelAfter=s?.cancelAfter||0,this.last=Date.now(),this.missed=0,this.count=0,this.callback=t,this._schedule()}cancel(){this.autoCancelTimer&&clearTimeout(this.autoCancelTimer),this.timer&&clearInterval(this.timer),this.timer=0,this.autoCancelTimer=0,this.missed=0}work(){this.last=Date.now(),this.missed=0}_change(e,t=0,s=2){this.interval=e,this.maxOut=s,this.cancelAfter=t,this.restart()}restart(){this.cancel(),this._schedule()}_schedule(){this.cancelAfter>0&&(this.autoCancelTimer=setTimeout(()=>{this.cancel()},this.cancelAfter)),this.timer=setInterval(()=>{if(this.count++,Date.now()-this.last>this.interval&&this.missed++,this.missed>=this.maxOut)try{this.callback(this.missed)===!0&&this.cancel()}catch(e){console.log(e)}},this.interval)}}var Ys;(function(r){r.Limits="limits",r.Interest="interest",r.Workqueue="workqueue"})(Ys||(Ys={}));var Dt;(function(r){r.Old="old",r.New="new"})(Dt||(Dt={}));var Xs;(function(r){r.File="file",r.Memory="memory"})(Xs||(Xs={}));var Q;(function(r){r.All="all",r.Last="last",r.New="new",r.StartSequence="by_start_sequence",r.StartTime="by_start_time",r.LastPerSubject="last_per_subject"})(Q||(Q={}));var ae;(function(r){r.None="none",r.All="all",r.Explicit="explicit",r.NotSet=""})(ae||(ae={}));var St;(function(r){r.Instant="instant",r.Original="original"})(St||(St={}));var Qe;(function(r){r.None="none",r.S2="s2"})(Qe||(Qe={}));var ms;(function(r){r.CreateOrUpdate="",r.Update="update",r.Create="create"})(ms||(ms={}));function Sn(r,e={}){return Object.assign({name:r,deliver_policy:Q.All,ack_policy:ae.Explicit,ack_wait:V(30*1e3),replay_policy:St.Instant},e)}var qr;(function(r){r.API="api_audit",r.StreamAction="stream_action",r.ConsumerAction="consumer_action",r.SnapshotCreate="snapshot_create",r.SnapshotComplete="snapshot_complete",r.RestoreCreate="restore_create",r.RestoreComplete="restore_complete",r.MaxDeliver="max_deliver",r.Terminated="terminated",r.Ack="consumer_ack",r.StreamLeaderElected="stream_leader_elected",r.StreamQuorumLost="stream_quorum_lost",r.ConsumerLeaderElected="consumer_leader_elected",r.ConsumerQuorumLost="consumer_quorum_lost"})(qr||(qr={}));var ge;(function(r){r.StreamSourceHdr="Nats-Stream-Source",r.LastConsumerSeqHdr="Nats-Last-Consumer",r.LastStreamSeqHdr="Nats-Last-Stream",r.ConsumerStalledHdr="Nats-Consumer-Stalled",r.MessageSizeHdr="Nats-Msg-Size",r.RollupHdr="Nats-Rollup",r.RollupValueSubject="sub",r.RollupValueAll="all",r.PendingMessagesHdr="Nats-Pending-Messages",r.PendingBytesHdr="Nats-Pending-Bytes"})(ge||(ge={}));var Ne;(function(r){r.LastValue="",r.AllHistory="history",r.UpdatesOnly="updates"})(Ne||(Ne={}));var pt;(function(r){r.Stream="Nats-Stream",r.Sequence="Nats-Sequence",r.TimeStamp="Nats-Time-Stamp",r.Subject="Nats-Subject"})(pt||(pt={}));var Fr;(function(r){r.Stream="Nats-Stream",r.Subject="Nats-Subject",r.Sequence="Nats-Sequence",r.LastSequence="Nats-Last-Sequence",r.Size="Nats-Msg-Size"})(Fr||(Fr={}));const _e="KV_";class kn{config;ordered;mack;stream;callbackFn;max;qname;isBind;filters;constructor(e){this.stream="",this.mack=!1,this.ordered=!1,this.config=Sn("",e||{})}getOpts(){const e={};if(e.config=Object.assign({},this.config),e.config.filter_subject&&(this.filterSubject(e.config.filter_subject),e.config.filter_subject=void 0),e.config.filter_subjects&&(e.config.filter_subjects?.forEach(t=>{this.filterSubject(t)}),e.config.filter_subjects=void 0),e.mack=this.mack,e.stream=this.stream,e.callbackFn=this.callbackFn,e.max=this.max,e.queue=this.qname,e.ordered=this.ordered,e.config.ack_policy=e.ordered?ae.None:e.config.ack_policy,e.isBind=e.isBind||!1,this.filters)switch(this.filters.length){case 0:break;case 1:e.config.filter_subject=this.filters[0];break;default:e.config.filter_subjects=this.filters}return e}description(e){return this.config.description=e,this}deliverTo(e){return this.config.deliver_subject=e,this}durable(e){return yt(e),this.config.durable_name=e,this}startSequence(e){if(e<=0)throw new Error("sequence must be greater than 0");return this.config.deliver_policy=Q.StartSequence,this.config.opt_start_seq=e,this}startTime(e){return this.config.deliver_policy=Q.StartTime,this.config.opt_start_time=e.toISOString(),this}deliverAll(){return this.config.deliver_policy=Q.All,this}deliverLastPerSubject(){return this.config.deliver_policy=Q.LastPerSubject,this}deliverLast(){return this.config.deliver_policy=Q.Last,this}deliverNew(){return this.config.deliver_policy=Q.New,this}startAtTimeDelta(e){return this.startTime(new Date(Date.now()-e)),this}headersOnly(){return this.config.headers_only=!0,this}ackNone(){return this.config.ack_policy=ae.None,this}ackAll(){return this.config.ack_policy=ae.All,this}ackExplicit(){return this.config.ack_policy=ae.Explicit,this}ackWait(e){return this.config.ack_wait=V(e),this}maxDeliver(e){return this.config.max_deliver=e,this}filterSubject(e){return this.filters=this.filters||[],this.filters.push(e),this}replayInstantly(){return this.config.replay_policy=St.Instant,this}replayOriginal(){return this.config.replay_policy=St.Original,this}sample(e){if(e=Math.trunc(e),e<0||e>100)throw new Error("value must be between 0-100");return this.config.sample_freq=`${e}%`,this}limit(e){return this.config.rate_limit_bps=e,this}maxWaiting(e){return this.config.max_waiting=e,this}maxAckPending(e){return this.config.max_ack_pending=e,this}idleHeartbeat(e){return this.config.idle_heartbeat=V(e),this}flowControl(){return this.config.flow_control=!0,this}deliverGroup(e){return this.queue(e),this}manualAck(){return this.mack=!0,this}maxMessages(e){return this.max=e,this}callback(e){return this.callbackFn=e,this}queue(e){return this.qname=e,this.config.deliver_group=e,this}orderedConsumer(){return this.ordered=!0,this}bind(e,t){return this.stream=e,this.config.durable_name=t,this.isBind=!0,this}bindStream(e){return this.stream=e,this}inactiveEphemeralThreshold(e){return this.config.inactive_threshold=V(e),this}maxPullBatch(e){return this.config.max_batch=e,this}maxPullRequestExpires(e){return this.config.max_expires=V(e),this}memory(){return this.config.mem_storage=!0,this}numReplicas(e){return this.config.num_replicas=e,this}consumerName(e){return this.config.name=e,this}}function Ye(r){return new kn(r)}function Lr(r){return typeof r.getOpts=="function"}class En{static encode(e){if(typeof e=="string")return btoa(e);const t=Array.from(e);return btoa(String.fromCharCode(...t))}static decode(e,t=!1){const s=atob(e);return t?Uint8Array.from(s,i=>i.charCodeAt(0)):s}}class xt{static encode(e){return xt.toB64URLEncoding(En.encode(e))}static decode(e,t=!1){return xt.decode(xt.fromB64URLEncoding(e),t)}static toB64URLEncoding(e){return e.replace(/\+/g,"-").replace(/\//g,"_")}static fromB64URLEncoding(e){return e.replace(/_/g,"/").replace(/-/g,"+")}}class kt{buffers;byteLength;constructor(){this.buffers=[],this.byteLength=0}static concat(...e){let t=0;for(let n=0;n<e.length;n++)t+=e[n].length;const s=new Uint8Array(t);let i=0;for(let n=0;n<e.length;n++)s.set(e[n],i),i+=e[n].length;return s}static fromAscii(e){return e||(e=""),et.encode(e)}static toAscii(e){return ke.decode(e)}reset(){this.buffers.length=0,this.byteLength=0}pack(){if(this.buffers.length>1){const e=new Uint8Array(this.byteLength);let t=0;for(let s=0;s<this.buffers.length;s++)e.set(this.buffers[s],t),t+=this.buffers[s].length;this.buffers.length=0,this.buffers.push(e)}}shift(){if(this.buffers.length){const e=this.buffers.shift();if(e)return this.byteLength-=e.length,e}return new Uint8Array(0)}drain(e){if(this.buffers.length){this.pack();const t=this.buffers.pop();if(t){const s=this.byteLength;(e===void 0||e>s)&&(e=s);const i=t.subarray(0,e);return s>e&&this.buffers.push(t.subarray(e)),this.byteLength=s-e,i}}return new Uint8Array(0)}fill(e,...t){e&&(this.buffers.push(e),this.byteLength+=e.length);for(let s=0;s<t.length;s++)t[s]&&t[s].length&&(this.buffers.push(t[s]),this.byteLength+=t[s].length)}peek(){return this.buffers.length?(this.pack(),this.buffers[0]):new Uint8Array(0)}size(){return this.byteLength}length(){return this.buffers.length}}function In(r,e){return e.forEach(function(t){t&&typeof t!="string"&&!Array.isArray(t)&&Object.keys(t).forEach(function(s){if(s!=="default"&&!(s in r)){var i=Object.getOwnPropertyDescriptor(t,s);Object.defineProperty(r,s,i.get?i:{enumerable:!0,get:function(){return t[s]}})}})}),Object.freeze(r)}var An=typeof global<"u"?global:typeof self<"u"?self:typeof window<"u"?window:{},Ot=An.performance||{};Ot.now||Ot.mozNow||Ot.msNow||Ot.oNow||Ot.webkitNow;var Ur={versions:{}},Pn=typeof globalThis<"u"?globalThis:typeof window<"u"?window:typeof global<"u"?global:typeof self<"u"?self:{};function Rn(r){if(r.__esModule)return r;var e=Object.defineProperty({},"__esModule",{value:!0});return Object.keys(r).forEach(function(t){var s=Object.getOwnPropertyDescriptor(r,t);Object.defineProperty(e,t,s.get?s:{enumerable:!0,get:function(){return r[t]}})}),e}var Ls,Ss={exports:{}},Br={},Dr=Rn(In({__proto__:null,default:Br},[Br]));Ls=Ss,(function(){var r="input is invalid type",e=typeof window=="object",t=e?window:{};t.JS_SHA256_NO_WINDOW&&(e=!1);var s=!e&&typeof self=="object",i=!t.JS_SHA256_NO_NODE_JS&&Ur.versions&&Ur.versions.node;i?t=Pn:s&&(t=self);var n=!t.JS_SHA256_NO_COMMON_JS&&Ls.exports,c=!t.JS_SHA256_NO_ARRAY_BUFFER&&typeof ArrayBuffer<"u",a="0123456789abcdef".split(""),d=[-2147483648,8388608,32768,128],m=[24,16,8,0],x=[1116352408,1899447441,3049323471,3921009573,961987163,1508970993,2453635748,2870763221,3624381080,310598401,607225278,1426881987,1925078388,2162078206,2614888103,3248222580,3835390401,4022224774,264347078,604807628,770255983,1249150122,1555081692,1996064986,2554220882,2821834349,2952996808,3210313671,3336571891,3584528711,113926993,338241895,666307205,773529912,1294757372,1396182291,1695183700,1986661051,2177026350,2456956037,2730485921,2820302411,3259730800,3345764771,3516065817,3600352804,4094571909,275423344,430227734,506948616,659060556,883997877,958139571,1322822218,1537002063,1747873779,1955562222,2024104815,2227730452,2361852424,2428436474,2756734187,3204031479,3329325298],v=["hex","array","digest","arrayBuffer"],S=[];!t.JS_SHA256_NO_NODE_JS&&Array.isArray||(Array.isArray=function(g){return Object.prototype.toString.call(g)==="[object Array]"}),!c||!t.JS_SHA256_NO_ARRAY_BUFFER_IS_VIEW&&ArrayBuffer.isView||(ArrayBuffer.isView=function(g){return typeof g=="object"&&g.buffer&&g.buffer.constructor===ArrayBuffer});var O=function(g,R){return function(q){return new D(R,!0).update(q)[g]()}},$=function(g){var R=O("hex",g);i&&(R=K(R,g)),R.create=function(){return new D(g)},R.update=function(k){return R.create().update(k)};for(var q=0;q<v.length;++q){var P=v[q];R[P]=O(P,g)}return R},K=function(g,R){var q,P=Dr,k=Dr.Buffer,N=R?"sha224":"sha256";return q=k.from&&!t.JS_SHA256_NO_BUFFER_FROM?k.from:function(M){return new k(M)},function(M){if(typeof M=="string")return P.createHash(N).update(M,"utf8").digest("hex");if(M==null)throw new Error(r);return M.constructor===ArrayBuffer&&(M=new Uint8Array(M)),Array.isArray(M)||ArrayBuffer.isView(M)||M.constructor===k?P.createHash(N).update(q(M)).digest("hex"):g(M)}},ee=function(g,R){return function(q,P){return new de(q,R,!0).update(P)[g]()}},B=function(g){var R=ee("hex",g);R.create=function(k){return new de(k,g)},R.update=function(k,N){return R.create(k).update(N)};for(var q=0;q<v.length;++q){var P=v[q];R[P]=ee(P,g)}return R};function D(g,R){R?(S[0]=S[16]=S[1]=S[2]=S[3]=S[4]=S[5]=S[6]=S[7]=S[8]=S[9]=S[10]=S[11]=S[12]=S[13]=S[14]=S[15]=0,this.blocks=S):this.blocks=[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],g?(this.h0=3238371032,this.h1=914150663,this.h2=812702999,this.h3=4144912697,this.h4=4290775857,this.h5=1750603025,this.h6=1694076839,this.h7=3204075428):(this.h0=1779033703,this.h1=3144134277,this.h2=1013904242,this.h3=2773480762,this.h4=1359893119,this.h5=2600822924,this.h6=528734635,this.h7=1541459225),this.block=this.start=this.bytes=this.hBytes=0,this.finalized=this.hashed=!1,this.first=!0,this.is224=g}function de(g,R,q){var P,k=typeof g;if(k==="string"){var N,M=[],T=g.length,G=0;for(P=0;P<T;++P)(N=g.charCodeAt(P))<128?M[G++]=N:N<2048?(M[G++]=192|N>>>6,M[G++]=128|63&N):N<55296||N>=57344?(M[G++]=224|N>>>12,M[G++]=128|N>>>6&63,M[G++]=128|63&N):(N=65536+((1023&N)<<10|1023&g.charCodeAt(++P)),M[G++]=240|N>>>18,M[G++]=128|N>>>12&63,M[G++]=128|N>>>6&63,M[G++]=128|63&N);g=M}else{if(k!=="object")throw new Error(r);if(g===null)throw new Error(r);if(c&&g.constructor===ArrayBuffer)g=new Uint8Array(g);else if(!(Array.isArray(g)||c&&ArrayBuffer.isView(g)))throw new Error(r)}g.length>64&&(g=new D(R,!0).update(g).array());var te=[],re=[];for(P=0;P<64;++P){var se=g[P]||0;te[P]=92^se,re[P]=54^se}D.call(this,R,q),this.update(re),this.oKeyPad=te,this.inner=!0,this.sharedMemory=q}D.prototype.update=function(g){if(!this.finalized){var R,q=typeof g;if(q!=="string"){if(q!=="object")throw new Error(r);if(g===null)throw new Error(r);if(c&&g.constructor===ArrayBuffer)g=new Uint8Array(g);else if(!(Array.isArray(g)||c&&ArrayBuffer.isView(g)))throw new Error(r);R=!0}for(var P,k,N=0,M=g.length,T=this.blocks;N<M;){if(this.hashed&&(this.hashed=!1,T[0]=this.block,this.block=T[16]=T[1]=T[2]=T[3]=T[4]=T[5]=T[6]=T[7]=T[8]=T[9]=T[10]=T[11]=T[12]=T[13]=T[14]=T[15]=0),R)for(k=this.start;N<M&&k<64;++N)T[k>>>2]|=g[N]<<m[3&k++];else for(k=this.start;N<M&&k<64;++N)(P=g.charCodeAt(N))<128?T[k>>>2]|=P<<m[3&k++]:P<2048?(T[k>>>2]|=(192|P>>>6)<<m[3&k++],T[k>>>2]|=(128|63&P)<<m[3&k++]):P<55296||P>=57344?(T[k>>>2]|=(224|P>>>12)<<m[3&k++],T[k>>>2]|=(128|P>>>6&63)<<m[3&k++],T[k>>>2]|=(128|63&P)<<m[3&k++]):(P=65536+((1023&P)<<10|1023&g.charCodeAt(++N)),T[k>>>2]|=(240|P>>>18)<<m[3&k++],T[k>>>2]|=(128|P>>>12&63)<<m[3&k++],T[k>>>2]|=(128|P>>>6&63)<<m[3&k++],T[k>>>2]|=(128|63&P)<<m[3&k++]);this.lastByteIndex=k,this.bytes+=k-this.start,k>=64?(this.block=T[16],this.start=k-64,this.hash(),this.hashed=!0):this.start=k}return this.bytes>4294967295&&(this.hBytes+=this.bytes/4294967296|0,this.bytes=this.bytes%4294967296),this}},D.prototype.finalize=function(){if(!this.finalized){this.finalized=!0;var g=this.blocks,R=this.lastByteIndex;g[16]=this.block,g[R>>>2]|=d[3&R],this.block=g[16],R>=56&&(this.hashed||this.hash(),g[0]=this.block,g[16]=g[1]=g[2]=g[3]=g[4]=g[5]=g[6]=g[7]=g[8]=g[9]=g[10]=g[11]=g[12]=g[13]=g[14]=g[15]=0),g[14]=this.hBytes<<3|this.bytes>>>29,g[15]=this.bytes<<3,this.hash()}},D.prototype.hash=function(){var g,R,q,P,k,N,M,T,G,te=this.h0,re=this.h1,se=this.h2,ie=this.h3,ce=this.h4,ue=this.h5,Y=this.h6,Z=this.h7,ne=this.blocks;for(g=16;g<64;++g)R=((k=ne[g-15])>>>7|k<<25)^(k>>>18|k<<14)^k>>>3,q=((k=ne[g-2])>>>17|k<<15)^(k>>>19|k<<13)^k>>>10,ne[g]=ne[g-16]+R+ne[g-7]+q|0;for(G=re&se,g=0;g<64;g+=4)this.first?(this.is224?(N=300032,Z=(k=ne[0]-1413257819)-150054599|0,ie=k+24177077|0):(N=704751109,Z=(k=ne[0]-210244248)-1521486534|0,ie=k+143694565|0),this.first=!1):(R=(te>>>2|te<<30)^(te>>>13|te<<19)^(te>>>22|te<<10),P=(N=te&re)^te&se^G,Z=ie+(k=Z+(q=(ce>>>6|ce<<26)^(ce>>>11|ce<<21)^(ce>>>25|ce<<7))+(ce&ue^~ce&Y)+x[g]+ne[g])|0,ie=k+(R+P)|0),R=(ie>>>2|ie<<30)^(ie>>>13|ie<<19)^(ie>>>22|ie<<10),P=(M=ie&te)^ie&re^N,Y=se+(k=Y+(q=(Z>>>6|Z<<26)^(Z>>>11|Z<<21)^(Z>>>25|Z<<7))+(Z&ce^~Z&ue)+x[g+1]+ne[g+1])|0,R=((se=k+(R+P)|0)>>>2|se<<30)^(se>>>13|se<<19)^(se>>>22|se<<10),P=(T=se&ie)^se&te^M,ue=re+(k=ue+(q=(Y>>>6|Y<<26)^(Y>>>11|Y<<21)^(Y>>>25|Y<<7))+(Y&Z^~Y&ce)+x[g+2]+ne[g+2])|0,R=((re=k+(R+P)|0)>>>2|re<<30)^(re>>>13|re<<19)^(re>>>22|re<<10),P=(G=re&se)^re&ie^T,ce=te+(k=ce+(q=(ue>>>6|ue<<26)^(ue>>>11|ue<<21)^(ue>>>25|ue<<7))+(ue&Y^~ue&Z)+x[g+3]+ne[g+3])|0,te=k+(R+P)|0,this.chromeBugWorkAround=!0;this.h0=this.h0+te|0,this.h1=this.h1+re|0,this.h2=this.h2+se|0,this.h3=this.h3+ie|0,this.h4=this.h4+ce|0,this.h5=this.h5+ue|0,this.h6=this.h6+Y|0,this.h7=this.h7+Z|0},D.prototype.hex=function(){this.finalize();var g=this.h0,R=this.h1,q=this.h2,P=this.h3,k=this.h4,N=this.h5,M=this.h6,T=this.h7,G=a[g>>>28&15]+a[g>>>24&15]+a[g>>>20&15]+a[g>>>16&15]+a[g>>>12&15]+a[g>>>8&15]+a[g>>>4&15]+a[15&g]+a[R>>>28&15]+a[R>>>24&15]+a[R>>>20&15]+a[R>>>16&15]+a[R>>>12&15]+a[R>>>8&15]+a[R>>>4&15]+a[15&R]+a[q>>>28&15]+a[q>>>24&15]+a[q>>>20&15]+a[q>>>16&15]+a[q>>>12&15]+a[q>>>8&15]+a[q>>>4&15]+a[15&q]+a[P>>>28&15]+a[P>>>24&15]+a[P>>>20&15]+a[P>>>16&15]+a[P>>>12&15]+a[P>>>8&15]+a[P>>>4&15]+a[15&P]+a[k>>>28&15]+a[k>>>24&15]+a[k>>>20&15]+a[k>>>16&15]+a[k>>>12&15]+a[k>>>8&15]+a[k>>>4&15]+a[15&k]+a[N>>>28&15]+a[N>>>24&15]+a[N>>>20&15]+a[N>>>16&15]+a[N>>>12&15]+a[N>>>8&15]+a[N>>>4&15]+a[15&N]+a[M>>>28&15]+a[M>>>24&15]+a[M>>>20&15]+a[M>>>16&15]+a[M>>>12&15]+a[M>>>8&15]+a[M>>>4&15]+a[15&M];return this.is224||(G+=a[T>>>28&15]+a[T>>>24&15]+a[T>>>20&15]+a[T>>>16&15]+a[T>>>12&15]+a[T>>>8&15]+a[T>>>4&15]+a[15&T]),G},D.prototype.toString=D.prototype.hex,D.prototype.digest=function(){this.finalize();var g=this.h0,R=this.h1,q=this.h2,P=this.h3,k=this.h4,N=this.h5,M=this.h6,T=this.h7,G=[g>>>24&255,g>>>16&255,g>>>8&255,255&g,R>>>24&255,R>>>16&255,R>>>8&255,255&R,q>>>24&255,q>>>16&255,q>>>8&255,255&q,P>>>24&255,P>>>16&255,P>>>8&255,255&P,k>>>24&255,k>>>16&255,k>>>8&255,255&k,N>>>24&255,N>>>16&255,N>>>8&255,255&N,M>>>24&255,M>>>16&255,M>>>8&255,255&M];return this.is224||G.push(T>>>24&255,T>>>16&255,T>>>8&255,255&T),G},D.prototype.array=D.prototype.digest,D.prototype.arrayBuffer=function(){this.finalize();var g=new ArrayBuffer(this.is224?28:32),R=new DataView(g);return R.setUint32(0,this.h0),R.setUint32(4,this.h1),R.setUint32(8,this.h2),R.setUint32(12,this.h3),R.setUint32(16,this.h4),R.setUint32(20,this.h5),R.setUint32(24,this.h6),this.is224||R.setUint32(28,this.h7),g},de.prototype=new D,de.prototype.finalize=function(){if(D.prototype.finalize.call(this),this.inner){this.inner=!1;var g=this.array();D.call(this,this.is224,this.sharedMemory),this.update(this.oKeyPad),this.update(g),D.prototype.finalize.call(this)}};var le=$();le.sha256=le,le.sha224=$(!0),le.sha256.hmac=B(),le.sha224.hmac=B(!0),n?Ls.exports=le:(t.sha256=le.sha256,t.sha224=le.sha224)})();Ss.exports;Ss.exports.sha224;var Hr=Ss.exports.sha256;function Zs(r){return Tn(r)}function jn(r){if(!/^[0-9A-Fa-f]+$/.test(r))return!1;const t=/^[0-9A-F]+$/.test(r),s=/^[0-9a-f]+$/.test(r);return t||s?r.length%2===0:!1}function On(r){return/^[A-Za-z0-9\-_]*(={0,2})?$/.test(r)||/^[A-Za-z0-9+/]*(={0,2})?$/.test(r)}function Cn(r){return jn(r)?"hex":On(r)?"b64":""}function Nn(r){if(r.length%2!==0)throw new Error("hex string must have an even length");const e=new Uint8Array(r.length/2);for(let t=0;t<r.length;t+=2)e[t/2]=parseInt(r.substring(t,t+2),16);return e}function Mn(r){r=r.replace(/-/g,"+"),r=r.replace(/_/g,"/");const e=atob(r);return Uint8Array.from(e,t=>t.charCodeAt(0))}function Tn(r){switch(Cn(r)){case"hex":return Nn(r);case"b64":return Mn(r)}return null}function $n(r,e){const t=typeof r=="string"?Zs(r):r,s=typeof e=="string"?Zs(e):e;if(t===null||s===null||t.length!==s.length)return!1;for(let i=0;i<t.length;i++)if(t[i]!==s[i])return!1;return!0}class li{token;received;ctx;requestSubject;mux;constructor(e,t,s=!0){this.mux=e,this.requestSubject=t,this.received=0,this.token=Ze.next(),s&&(this.ctx=new Error)}}class qn extends li{callback;done;timer;max;opts;constructor(e,t,s={maxWait:1e3}){if(super(e,t),this.opts=s,typeof this.opts.callback!="function")throw new Error("callback is required");this.callback=this.opts.callback,this.max=typeof s.maxMessages=="number"&&s.maxMessages>0?s.maxMessages:-1,this.done=W(),this.done.then(()=>{this.callback(null,null)}),this.timer=setTimeout(()=>{this.cancel()},s.maxWait)}cancel(e){e&&this.callback(e,null),clearTimeout(this.timer),this.mux.cancel(this),this.done.resolve()}resolver(e,t){e?(this.ctx&&(e.stack+=`

${this.ctx.stack}`),this.cancel(e)):(this.callback(null,t),this.opts.strategy===Te.Count&&(this.max--,this.max===0&&this.cancel()),this.opts.strategy===Te.JitterTimer&&(clearTimeout(this.timer),this.timer=setTimeout(()=>{this.cancel()},this.opts.jitter||300)),this.opts.strategy===Te.SentinelMsg&&t&&t.data.length===0&&this.cancel())}}class di extends li{deferred;timer;constructor(e,t,s={timeout:1e3},i=!0){super(e,t,i),this.deferred=W(),this.timer=vt(s.timeout,i)}resolver(e,t){this.timer&&this.timer.cancel(),e?(this.ctx&&(e.stack+=`

${this.ctx.stack}`),this.deferred.reject(e)):this.deferred.resolve(t),this.cancel()}cancel(e){this.timer&&this.timer.cancel(),this.mux.cancel(this),this.deferred.reject(e||j.errorForCode(E.Cancelled))}}const Fn="$JS.API";function Ln(r){return r=r||{},r.domain&&(r.apiPrefix=`$JS.${r.domain}.API`,delete r.domain),_s({apiPrefix:Fn,timeout:5e3},r)}class Gt{nc;opts;prefix;timeout;jc;constructor(e,t){this.nc=e,this.opts=Ln(t),this._parseOpts(),this.prefix=this.opts.apiPrefix,this.timeout=this.opts.timeout,this.jc=$e()}getOptions(){return Object.assign({},this.opts)}_parseOpts(){let e=this.opts.apiPrefix;if(!e||e.length===0)throw new Error("invalid empty prefix");e[e.length-1]==="."&&(e=e.substr(0,e.length-1)),this.opts.apiPrefix=e}async _request(e,t=null,s){s=s||{},s.timeout=this.timeout;let i=Ie;t&&(i=this.jc.encode(t));let{retries:n}=s;n=n||1,n=n===-1?Number.MAX_SAFE_INTEGER:n;const c=cr();for(let a=0;a<n;a++)try{const d=await this.nc.request(e,i,s);return this.parseJsResponse(d)}catch(d){const m=d;if((m.code==="503"||m.code===E.Timeout)&&a+1<n)await Et(c.backoff(a));else throw d}}async findStream(e){const t={subject:e},i=await this._request(`${this.prefix}.STREAM.NAMES`,t);if(!i.streams||i.streams.length!==1)throw new Error("no stream matches subject");return i.streams[0]}getConnection(){return this.nc}parseJsResponse(e){const t=this.jc.decode(e.data),s=t;if(s.error){const i=hi(s.error.code,s.error.description);if(i!==null)throw i.api_error=s.error,i}return t}}class Mt{err;offset;pageInfo;subject;jsm;filter;payload;constructor(e,t,s,i){if(!e)throw new Error("subject is required");this.subject=e,this.jsm=s,this.offset=0,this.pageInfo={},this.filter=t,this.payload=i||{}}async next(){if(this.err)return[];if(this.pageInfo&&this.offset>=this.pageInfo.total)return[];const e={offset:this.offset};this.payload&&Object.assign(e,this.payload);try{const t=await this.jsm._request(this.subject,e,{timeout:this.jsm.timeout});this.pageInfo=t;const s=this.countResponse(t);return s===0?[]:(this.offset+=s,this.filter(t))}catch(t){throw this.err=t,t}}countResponse(e){switch(e?.type){case"io.nats.jetstream.api.v1.stream_names_response":case"io.nats.jetstream.api.v1.stream_list_response":return e.streams?.length||0;case"io.nats.jetstream.api.v1.consumer_list_response":return e.consumers?.length||0;default:return console.error(`jslister.ts: unknown API response for paged output: ${e?.type}`),e.streams?.length||0}return 0}async*[Symbol.asyncIterator](){let e=await this.next();for(;e.length>0;){for(const t of e)yield t;e=await this.next()}}}function rt(r=""){const e=r.match(/(\d+).(\d+).(\d+)/);if(e)return{major:parseInt(e[1]),minor:parseInt(e[2]),micro:parseInt(e[3])};throw new Error(`'${r}' is not a semver value`)}function Qs(r,e){return r.major<e.major?-1:r.major>e.major?1:r.minor<e.minor?-1:r.minor>e.minor?1:r.micro<e.micro?-1:r.micro>e.micro?1:0}var U;(function(r){r.JS_KV="js_kv",r.JS_OBJECTSTORE="js_objectstore",r.JS_PULL_MAX_BYTES="js_pull_max_bytes",r.JS_NEW_CONSUMER_CREATE_API="js_new_consumer_create",r.JS_ALLOW_DIRECT="js_allow_direct",r.JS_MULTIPLE_CONSUMER_FILTER="js_multiple_consumer_filter",r.JS_SIMPLIFICATION="js_simplification",r.JS_STREAM_CONSUMER_METADATA="js_stream_consumer_metadata",r.JS_CONSUMER_FILTER_SUBJECTS="js_consumer_filter_subjects",r.JS_STREAM_FIRST_SEQ="js_stream_first_seq",r.JS_STREAM_SUBJECT_TRANSFORM="js_stream_subject_transform",r.JS_STREAM_SOURCE_SUBJECT_TRANSFORM="js_stream_source_subject_transform",r.JS_STREAM_COMPRESSION="js_stream_compression",r.JS_DEFAULT_CONSUMER_LIMITS="js_default_consumer_limits",r.JS_BATCH_DIRECT_GET="js_batch_direct_get"})(U||(U={}));class Un{server;features;disabled;constructor(e){this.features=new Map,this.disabled=[],this.update(e)}resetDisabled(){this.disabled.length=0,this.update(this.server)}disable(e){this.disabled.push(e),this.update(this.server)}isDisabled(e){return this.disabled.indexOf(e)!==-1}update(e){typeof e=="string"&&(e=rt(e)),this.server=e,this.set(U.JS_KV,"2.6.2"),this.set(U.JS_OBJECTSTORE,"2.6.3"),this.set(U.JS_PULL_MAX_BYTES,"2.8.3"),this.set(U.JS_NEW_CONSUMER_CREATE_API,"2.9.0"),this.set(U.JS_ALLOW_DIRECT,"2.9.0"),this.set(U.JS_MULTIPLE_CONSUMER_FILTER,"2.10.0"),this.set(U.JS_SIMPLIFICATION,"2.9.4"),this.set(U.JS_STREAM_CONSUMER_METADATA,"2.10.0"),this.set(U.JS_CONSUMER_FILTER_SUBJECTS,"2.10.0"),this.set(U.JS_STREAM_FIRST_SEQ,"2.10.0"),this.set(U.JS_STREAM_SUBJECT_TRANSFORM,"2.10.0"),this.set(U.JS_STREAM_SOURCE_SUBJECT_TRANSFORM,"2.10.0"),this.set(U.JS_STREAM_COMPRESSION,"2.10.0"),this.set(U.JS_DEFAULT_CONSUMER_LIMITS,"2.10.0"),this.set(U.JS_BATCH_DIRECT_GET,"2.11.0"),this.disabled.forEach(t=>{this.features.delete(t)})}set(e,t){this.features.set(e,{min:t,ok:Qs(this.server,rt(t))>=0})}get(e){return this.features.get(e)||{min:"unknown",ok:!1}}supports(e){return this.get(e)?.ok||!1}require(e){return typeof e=="string"&&(e=rt(e)),Qs(this.server,e)>=0}}class bs extends Gt{constructor(e,t){super(e,t)}async add(e,t,s=ms.Create){if(pe(e),t.deliver_group&&t.flow_control)throw new Error("jetstream flow control is not supported with queue groups");if(t.deliver_group&&t.idle_heartbeat)throw new Error("jetstream idle heartbeat is not supported with queue groups");const i={};i.config=t,i.stream_name=e,i.action=s,i.config.durable_name&&yt(i.config.durable_name);const n=this.nc;let{min:c,ok:a}=n.features.get(U.JS_NEW_CONSUMER_CREATE_API);const d=t.name===""?void 0:t.name;if(d&&!a)throw new Error(`consumer 'name' requires server ${c}`);if(d)try{vs("name",d)}catch(S){const O=S.message,$=O.indexOf("cannot contain");throw $!==-1?new Error(`consumer 'name' ${O.substring($)}`):S}let m,x="";if(Array.isArray(t.filter_subjects)){const{min:S,ok:O}=n.features.get(U.JS_MULTIPLE_CONSUMER_FILTER);if(!O)throw new Error(`consumer 'filter_subjects' requires server ${S}`);a=!1}if(t.metadata){const{min:S,ok:O}=n.features.get(U.JS_STREAM_CONSUMER_METADATA);if(!O)throw new Error(`consumer 'metadata' requires server ${S}`)}if(a&&(x=t.name??t.durable_name??""),x!==""){let S=t.filter_subject??void 0;S===">"&&(S=void 0),m=S!==void 0?`${this.prefix}.CONSUMER.CREATE.${e}.${x}.${S}`:`${this.prefix}.CONSUMER.CREATE.${e}.${x}`}else m=t.durable_name?`${this.prefix}.CONSUMER.DURABLE.CREATE.${e}.${t.durable_name}`:`${this.prefix}.CONSUMER.CREATE.${e}`;return await this._request(m,i)}async update(e,t,s){const i=await this.info(e,t),n=s;return this.add(e,Object.assign(i.config,n),ms.Update)}async info(e,t){return pe(e),yt(t),await this._request(`${this.prefix}.CONSUMER.INFO.${e}.${t}`)}async delete(e,t){return pe(e),yt(t),(await this._request(`${this.prefix}.CONSUMER.DELETE.${e}.${t}`)).success}list(e){pe(e);const t=i=>i.consumers,s=`${this.prefix}.CONSUMER.LIST.${e}`;return new Mt(s,t,this)}pause(e,t,s){const i=`${this.prefix}.CONSUMER.PAUSE.${e}.${t}`,n={pause_until:s.toISOString()};return this._request(i,n)}resume(e,t){return this.pause(e,t,new Date(0))}}function lt(r,e,t=!1){if(t===!0&&!r)throw j.errorForCode(E.ApiError,new Error(`${e} is not a function`));if(r&&typeof r!="function")throw j.errorForCode(E.ApiError,new Error(`${e} is not a function`))}class Bn extends oe{sub;adapter;subIterDone;constructor(e,t,s){super(),lt(s.adapter,"adapter",!0),this.adapter=s.adapter,s.callback&&lt(s.callback,"callback"),this.noIterator=typeof s.callback=="function",s.ingestionFilterFn&&(lt(s.ingestionFilterFn,"ingestionFilterFn"),this.ingestionFilterFn=s.ingestionFilterFn),s.protocolFilterFn&&(lt(s.protocolFilterFn,"protocolFilterFn"),this.protocolFilterFn=s.protocolFilterFn),s.dispatchedFn&&(lt(s.dispatchedFn,"dispatchedFn"),this.dispatchedFn=s.dispatchedFn),s.cleanupFn&&lt(s.cleanupFn,"cleanupFn");let i=(m,x)=>{this.callback(m,x)};if(s.callback){const m=s.callback;i=(x,v)=>{const[S,O]=this.adapter(x,v);if(S){m(S,null);return}const{ingest:$}=this.ingestionFilterFn?this.ingestionFilterFn(O,this):{ingest:!0};$&&(!this.protocolFilterFn||this.protocolFilterFn(O))&&(m(S,O),this.dispatchedFn&&O&&this.dispatchedFn(O))}}const{max:n,queue:c,timeout:a}=s,d={queue:c,timeout:a,callback:i};n&&n>0&&(d.max=n),this.sub=e.subscribe(t,d),s.cleanupFn&&(this.sub.cleanupFn=s.cleanupFn),this.noIterator||this.iterClosed.then(()=>{this.unsubscribe()}),this.subIterDone=W(),Promise.all([this.sub.closed,this.iterClosed]).then(()=>{this.subIterDone.resolve()}).catch(()=>{this.subIterDone.resolve()}),(async m=>{await m.closed,this.stop()})(this.sub).then().catch()}unsubscribe(e){this.sub.unsubscribe(e)}drain(){return this.sub.drain()}isDraining(){return this.sub.isDraining()}isClosed(){return this.sub.isClosed()}callback(e,t){this.sub.cancelTimeout();const[s,i]=this.adapter(e,t);s&&this.stop(s),i&&this.push(i)}getSubject(){return this.sub.getSubject()}getReceived(){return this.sub.getReceived()}getProcessed(){return this.sub.getProcessed()}getPending(){return this.sub.getPending()}getID(){return this.sub.getID()}getMax(){return this.sub.getMax()}get closed(){return this.sub.closed}}let Ee;function Dn(r){Ee=r}function fi(){return Ee!==void 0&&Ee.defaultPort!==void 0?Ee.defaultPort:4222}function Us(){return Ee!==void 0&&Ee.urlParseFn?Ee.urlParseFn:void 0}function Hn(){if(!Ee||typeof Ee.factory!="function")throw new Error("transport fn is not set");return Ee.factory()}function er(){return Ee!==void 0&&Ee.dnsResolveFn?Ee.dnsResolveFn:void 0}const us=`\r
`,gs=kt.fromAscii(us),zn=new Uint8Array(gs)[0],Jn=new Uint8Array(gs)[1];function Kn(r){for(let e=0;e<r.length;e++){const t=e+1;if(r.byteLength>t&&r[e]===zn&&r[t]===Jn)return t+1}return 0}function Gn(r){const e=Kn(r);if(e>0){const s=new Uint8Array(r).slice(0,e);return ke.decode(s)}return""}const Vn=4,pi=48,Wn=65,Yn=97;function Xn(r,e,t,s){const i=new Uint8Array(16);return[0,0,0,0,0,0,0,0,0,0,255,255].forEach((c,a)=>{i[a]=c}),i[12]=r,i[13]=e,i[14]=t,i[15]=s,i}function tr(r){return Zn(r)!==void 0}function Zn(r){for(let e=0;e<r.length;e++)switch(r[e]){case".":return mi(r);case":":return Qn(r)}}function mi(r){const e=new Uint8Array(4);for(let t=0;t<4;t++){if(r.length===0)return;if(t>0){if(r[0]!==".")return;r=r.substring(1)}const{n:s,c:i,ok:n}=ea(r);if(!n||s>255)return;r=r.substring(i),e[t]=s}return Xn(e[0],e[1],e[2],e[3])}function Qn(r){const e=new Uint8Array(16);let t=-1;if(r.length>=2&&r[0]===":"&&r[1]===":"&&(t=0,r=r.substring(2),r.length===0))return e;let s=0;for(;s<16;){const{n:i,c:n,ok:c}=ta(r);if(!c||i>65535)return;if(n<r.length&&r[n]==="."){if(t<0&&s!=12||s+4>16)return;const a=mi(r);if(a===void 0)return;e[s]=a[12],e[s+1]=a[13],e[s+2]=a[14],e[s+3]=a[15],r="",s+=Vn;break}if(e[s]=i>>8,e[s+1]=i,s+=2,r=r.substring(n),r.length===0)break;if(r[0]!==":"||r.length==1)return;if(r=r.substring(1),r[0]===":"){if(t>=0)return;if(t=s,r=r.substring(1),r.length===0)break}}if(r.length===0){if(s<16){if(t<0)return;const i=16-s;for(let n=s-1;n>=t;n--)e[n+i]=e[n];for(let n=t+i-1;n>=t;n--)e[n]=0}else if(t>=0)return;return e}}function ea(r){let e=0,t=0;for(e=0;e<r.length&&48<=r.charCodeAt(e)&&r.charCodeAt(e)<=57;e++)if(t=t*10+(r.charCodeAt(e)-pi),t>=16777215)return{n:16777215,c:e,ok:!1};return e===0?{n:0,c:0,ok:!1}:{n:t,c:e,ok:!0}}function ta(r){let e=0,t=0;for(t=0;t<r.length;t++){if(48<=r.charCodeAt(t)&&r.charCodeAt(t)<=57)e*=16,e+=r.charCodeAt(t)-pi;else if(97<=r.charCodeAt(t)&&r.charCodeAt(t)<=102)e*=16,e+=r.charCodeAt(t)-Yn+10;else if(65<=r.charCodeAt(t)&&r.charCodeAt(t)<=70)e*=16,e+=r.charCodeAt(t)-Wn+10;else break;if(e>=16777215)return{n:0,c:t,ok:!1}}return t===0?{n:0,c:t,ok:!1}:{n:e,c:t,ok:!0}}function sa(r){return r.indexOf("[")!==-1||r.indexOf("::")!==-1?!1:r.indexOf(".")!==-1||r.split(":").length<=2}function sr(r){return!sa(r)}function ra(r){const e="::FFFF:",t=r.toUpperCase().indexOf(e);if(t!==-1&&r.indexOf(".")!==-1){let s=r.substring(t+e.length);return s=s.replace("[",""),s.replace("]","")}return r}function ia(r){r=r.trim(),r.match(/^(.*:\/\/)(.*)/m)&&(r=r.replace(/^(.*:\/\/)(.*)/gm,"$2")),r=ra(r),sr(r)&&r.indexOf("[")===-1&&(r=`[${r}]`);const e=sr(r)?r.match(/(]:)(\d+)/):r.match(/(:)(\d+)/),t=e&&e.length===3&&e[1]&&e[2]?parseInt(e[2]):4222,s=t===80?"https":"http",i=new URL(`${s}://${r}`);i.port=`${t}`;let n=i.hostname;return n.charAt(0)==="["&&(n=n.substring(1,n.length-1)),{listen:i.host,hostname:n,port:t}}class qt{src;listen;hostname;port;didConnect;reconnects;lastConnect;gossiped;tlsName;resolves;constructor(e,t=!1){this.src=e,this.tlsName="";const s=ia(e);this.listen=s.listen,this.hostname=s.hostname,this.port=s.port,this.didConnect=!1,this.reconnects=0,this.lastConnect=0,this.gossiped=t}toString(){return this.listen}async resolve(e){if(!e.fn||e.resolve===!1)return[this];const t=[];if(tr(this.hostname))return[this];{const s=await e.fn(this.hostname);e.debug&&console.log(`resolve ${this.hostname} = ${s.join(",")}`);for(const i of s){const n=this.port===80?"https":"http",c=new URL(`${n}://${sr(i)?"["+i+"]":i}`);c.port=`${this.port}`;const a=new qt(c.host,!1);a.tlsName=this.hostname,t.push(a)}}return e.randomize&&oi(t),this.resolves=t,t}}class na{firstSelect;servers;currentServer;tlsName;randomize;constructor(e=[],t={}){this.firstSelect=!0,this.servers=[],this.tlsName="",this.randomize=t.randomize||!1;const s=Us();e&&(e.forEach(i=>{i=s?s(i):i,this.servers.push(new qt(i))}),this.randomize&&(this.servers=oi(this.servers))),this.servers.length===0&&this.addServer(`${Gs}:${fi()}`,!1),this.currentServer=this.servers[0]}clear(){this.servers.length=0}updateTLSName(){const e=this.getCurrentServer();tr(e.hostname)||(this.tlsName=e.hostname,this.servers.forEach(t=>{t.gossiped&&(t.tlsName=this.tlsName)}))}getCurrentServer(){return this.currentServer}addServer(e,t=!1){const s=Us();e=s?s(e):e;const i=new qt(e,t);tr(i.hostname)&&(i.tlsName=this.tlsName),this.servers.push(i)}selectServer(){if(this.firstSelect)return this.firstSelect=!1,this.currentServer;const e=this.servers.shift();return e&&(this.servers.push(e),this.currentServer=e),e}removeCurrentServer(){this.removeServer(this.currentServer)}removeServer(e){if(e){const t=this.servers.indexOf(e);this.servers.splice(t,1)}}length(){return this.servers.length}next(){return this.servers.length?this.servers[0]:void 0}getServers(){return this.servers}update(e,t){const s=[];let i=[];const n=Us(),c=new Map;e.connect_urls&&e.connect_urls.length>0&&e.connect_urls.forEach(d=>{d=n?n(d,t):d;const m=new qt(d,!0);c.set(d,m)});const a=[];return this.servers.forEach((d,m)=>{const x=d.listen;d.gossiped&&this.currentServer.listen!==x&&c.get(x)===void 0&&a.push(m),c.delete(x)}),a.reverse(),a.forEach(d=>{const m=this.servers.splice(d,1);i=i.concat(m[0].listen)}),c.forEach((d,m)=>{this.servers.push(d),s.push(m)}),{added:s,deleted:i}}}class aa{baseInbox;reqs;constructor(){this.reqs=new Map}size(){return this.reqs.size}init(e){return this.baseInbox=`${He(e)}.`,this.baseInbox}add(e){isNaN(e.received)||(e.received=0),this.reqs.set(e.token,e)}get(e){return this.reqs.get(e)}cancel(e){this.reqs.delete(e.token)}getToken(e){const t=e.subject||"";return t.indexOf(this.baseInbox)===0?t.substring(this.baseInbox.length):null}all(){return Array.from(this.reqs.values())}handleError(e,t){if(t&&t.permissionContext){if(e)return this.all().forEach(i=>{i.resolver(t,{})}),!0;const s=t.permissionContext;if(s.operation==="publish"){const i=this.all().find(n=>n.requestSubject===s.subject);if(i)return i.resolver(t,{}),!0}}return!1}dispatcher(){return(e,t)=>{const s=this.getToken(t);if(s){const i=this.get(s);i&&(e===null&&t.headers&&(e=ci(t)),i.resolver(e,t))}}}close(){const e=j.errorForCode(E.Timeout);this.reqs.forEach(t=>{t.resolver(e,{})})}}class oa{ph;interval;maxOut;timer;pendings;constructor(e,t,s){this.ph=e,this.interval=t,this.maxOut=s,this.pendings=[]}start(){this.cancel(),this._schedule()}cancel(e){this.timer&&(clearTimeout(this.timer),this.timer=void 0),this._reset(),e&&this.ph.disconnect()}_schedule(){this.timer=setTimeout(()=>{if(this.ph.dispatchStatus({type:gt.PingTimer,data:`${this.pendings.length+1}`}),this.pendings.length===this.maxOut){this.cancel(!0);return}const e=W();this.ph.flush(e).then(()=>{this._reset()}).catch(()=>{this.cancel()}),this.pendings.push(e),this._schedule()},this.interval)}_reset(){this.pendings=this.pendings.filter(e=>(e.resolve(),!1))}}class ca extends Error{constructor(e){super(e),this.name="AssertionError"}}function ua(r,e="Assertion failed."){if(!r)throw new ca(e)}const zr=32*1024,Bs=2**32-2;function ns(r,e,t=0){const s=e.byteLength-t;return r.byteLength>s&&(r=r.subarray(0,s)),e.set(r,t),r.byteLength}class Ds{_buf;_off;constructor(e){if(this._off=0,e==null){this._buf=new Uint8Array(0);return}this._buf=new Uint8Array(e)}bytes(e={copy:!0}){return e.copy===!1?this._buf.subarray(this._off):this._buf.slice(this._off)}empty(){return this._buf.byteLength<=this._off}get length(){return this._buf.byteLength-this._off}get capacity(){return this._buf.buffer.byteLength}truncate(e){if(e===0){this.reset();return}if(e<0||e>this.length)throw Error("bytes.Buffer: truncation out of range");this._reslice(this._off+e)}reset(){this._reslice(0),this._off=0}_tryGrowByReslice(e){const t=this._buf.byteLength;return e<=this.capacity-t?(this._reslice(t+e),t):-1}_reslice(e){ua(e<=this._buf.buffer.byteLength),this._buf=new Uint8Array(this._buf.buffer,0,e)}readByte(){const e=new Uint8Array(1);return this.read(e)?e[0]:null}read(e){if(this.empty())return this.reset(),e.byteLength===0?0:null;const t=ns(this._buf.subarray(this._off),e);return this._off+=t,t}writeByte(e){return this.write(Uint8Array.of(e))}writeString(e){return this.write(et.encode(e))}write(e){const t=this._grow(e.byteLength);return ns(e,this._buf,t)}_grow(e){const t=this.length;t===0&&this._off!==0&&this.reset();const s=this._tryGrowByReslice(e);if(s>=0)return s;const i=this.capacity;if(e<=Math.floor(i/2)-t)ns(this._buf.subarray(this._off),this._buf);else{if(i+e>Bs)throw new Error("The buffer cannot be grown beyond the maximum size.");{const n=new Uint8Array(Math.min(2*i+e,Bs));ns(this._buf.subarray(this._off),n),this._buf=n}}return this._off=0,this._reslice(Math.min(t+e,Bs)),t}grow(e){if(e<0)throw Error("Buffer._grow: negative count");const t=this._grow(e);this._reslice(t)}readFrom(e){let t=0;const s=new Uint8Array(zr);for(;;){const i=this.capacity-this.length<zr,n=i?s:new Uint8Array(this._buf.buffer,this.length),c=e.read(n);if(c===null)return t;i?this.write(n.subarray(0,c)):this._reslice(this.length+c),t+=c}}}var me;(function(r){r[r.OK=0]="OK",r[r.ERR=1]="ERR",r[r.MSG=2]="MSG",r[r.INFO=3]="INFO",r[r.PING=4]="PING",r[r.PONG=5]="PONG"})(me||(me={}));function Jr(){const r={};return r.sid=-1,r.hdr=-1,r.size=-1,r}const ha=48;class Kr{dispatcher;state;as;drop;hdr;ma;argBuf;msgBuf;constructor(e){this.dispatcher=e,this.state=I.OP_START,this.as=0,this.drop=0,this.hdr=0}parse(e){let t;for(t=0;t<e.length;t++){const s=e[t];switch(this.state){case I.OP_START:switch(s){case A.M:case A.m:this.state=I.OP_M,this.hdr=-1,this.ma=Jr();break;case A.H:case A.h:this.state=I.OP_H,this.hdr=0,this.ma=Jr();break;case A.P:case A.p:this.state=I.OP_P;break;case A.PLUS:this.state=I.OP_PLUS;break;case A.MINUS:this.state=I.OP_MINUS;break;case A.I:case A.i:this.state=I.OP_I;break;default:throw this.fail(e.subarray(t))}break;case I.OP_H:switch(s){case A.M:case A.m:this.state=I.OP_M;break;default:throw this.fail(e.subarray(t))}break;case I.OP_M:switch(s){case A.S:case A.s:this.state=I.OP_MS;break;default:throw this.fail(e.subarray(t))}break;case I.OP_MS:switch(s){case A.G:case A.g:this.state=I.OP_MSG;break;default:throw this.fail(e.subarray(t))}break;case I.OP_MSG:switch(s){case A.SPACE:case A.TAB:this.state=I.OP_MSG_SPC;break;default:throw this.fail(e.subarray(t))}break;case I.OP_MSG_SPC:switch(s){case A.SPACE:case A.TAB:continue;default:this.state=I.MSG_ARG,this.as=t}break;case I.MSG_ARG:switch(s){case A.CR:this.drop=1;break;case A.NL:{const i=this.argBuf?this.argBuf.bytes():e.subarray(this.as,t-this.drop);this.processMsgArgs(i),this.drop=0,this.as=t+1,this.state=I.MSG_PAYLOAD,t=this.as+this.ma.size-1;break}default:this.argBuf&&this.argBuf.writeByte(s)}break;case I.MSG_PAYLOAD:if(this.msgBuf)if(this.msgBuf.length>=this.ma.size){const i=this.msgBuf.bytes({copy:!1});this.dispatcher.push({kind:me.MSG,msg:this.ma,data:i}),this.argBuf=void 0,this.msgBuf=void 0,this.state=I.MSG_END}else{let i=this.ma.size-this.msgBuf.length;const n=e.length-t;n<i&&(i=n),i>0?(this.msgBuf.write(e.subarray(t,t+i)),t=t+i-1):this.msgBuf.writeByte(s)}else t-this.as>=this.ma.size&&(this.dispatcher.push({kind:me.MSG,msg:this.ma,data:e.subarray(this.as,t)}),this.argBuf=void 0,this.msgBuf=void 0,this.state=I.MSG_END);break;case I.MSG_END:if(s===A.NL)this.drop=0,this.as=t+1,this.state=I.OP_START;else continue;break;case I.OP_PLUS:switch(s){case A.O:case A.o:this.state=I.OP_PLUS_O;break;default:throw this.fail(e.subarray(t))}break;case I.OP_PLUS_O:switch(s){case A.K:case A.k:this.state=I.OP_PLUS_OK;break;default:throw this.fail(e.subarray(t))}break;case I.OP_PLUS_OK:s===A.NL&&(this.dispatcher.push({kind:me.OK}),this.drop=0,this.state=I.OP_START);break;case I.OP_MINUS:switch(s){case A.E:case A.e:this.state=I.OP_MINUS_E;break;default:throw this.fail(e.subarray(t))}break;case I.OP_MINUS_E:switch(s){case A.R:case A.r:this.state=I.OP_MINUS_ER;break;default:throw this.fail(e.subarray(t))}break;case I.OP_MINUS_ER:switch(s){case A.R:case A.r:this.state=I.OP_MINUS_ERR;break;default:throw this.fail(e.subarray(t))}break;case I.OP_MINUS_ERR:switch(s){case A.SPACE:case A.TAB:this.state=I.OP_MINUS_ERR_SPC;break;default:throw this.fail(e.subarray(t))}break;case I.OP_MINUS_ERR_SPC:switch(s){case A.SPACE:case A.TAB:continue;default:this.state=I.MINUS_ERR_ARG,this.as=t}break;case I.MINUS_ERR_ARG:switch(s){case A.CR:this.drop=1;break;case A.NL:{let i;this.argBuf?(i=this.argBuf.bytes(),this.argBuf=void 0):i=e.subarray(this.as,t-this.drop),this.dispatcher.push({kind:me.ERR,data:i}),this.drop=0,this.as=t+1,this.state=I.OP_START;break}default:this.argBuf&&this.argBuf.write(Uint8Array.of(s))}break;case I.OP_P:switch(s){case A.I:case A.i:this.state=I.OP_PI;break;case A.O:case A.o:this.state=I.OP_PO;break;default:throw this.fail(e.subarray(t))}break;case I.OP_PO:switch(s){case A.N:case A.n:this.state=I.OP_PON;break;default:throw this.fail(e.subarray(t))}break;case I.OP_PON:switch(s){case A.G:case A.g:this.state=I.OP_PONG;break;default:throw this.fail(e.subarray(t))}break;case I.OP_PONG:s===A.NL&&(this.dispatcher.push({kind:me.PONG}),this.drop=0,this.state=I.OP_START);break;case I.OP_PI:switch(s){case A.N:case A.n:this.state=I.OP_PIN;break;default:throw this.fail(e.subarray(t))}break;case I.OP_PIN:switch(s){case A.G:case A.g:this.state=I.OP_PING;break;default:throw this.fail(e.subarray(t))}break;case I.OP_PING:s===A.NL&&(this.dispatcher.push({kind:me.PING}),this.drop=0,this.state=I.OP_START);break;case I.OP_I:switch(s){case A.N:case A.n:this.state=I.OP_IN;break;default:throw this.fail(e.subarray(t))}break;case I.OP_IN:switch(s){case A.F:case A.f:this.state=I.OP_INF;break;default:throw this.fail(e.subarray(t))}break;case I.OP_INF:switch(s){case A.O:case A.o:this.state=I.OP_INFO;break;default:throw this.fail(e.subarray(t))}break;case I.OP_INFO:switch(s){case A.SPACE:case A.TAB:this.state=I.OP_INFO_SPC;break;default:throw this.fail(e.subarray(t))}break;case I.OP_INFO_SPC:switch(s){case A.SPACE:case A.TAB:continue;default:this.state=I.INFO_ARG,this.as=t}break;case I.INFO_ARG:switch(s){case A.CR:this.drop=1;break;case A.NL:{let i;this.argBuf?(i=this.argBuf.bytes(),this.argBuf=void 0):i=e.subarray(this.as,t-this.drop),this.dispatcher.push({kind:me.INFO,data:i}),this.drop=0,this.as=t+1,this.state=I.OP_START;break}default:this.argBuf&&this.argBuf.writeByte(s)}break;default:throw this.fail(e.subarray(t))}}(this.state===I.MSG_ARG||this.state===I.MINUS_ERR_ARG||this.state===I.INFO_ARG)&&!this.argBuf&&(this.argBuf=new Ds(e.subarray(this.as,t-this.drop))),this.state===I.MSG_PAYLOAD&&!this.msgBuf&&(this.argBuf||this.cloneMsgArg(),this.msgBuf=new Ds(e.subarray(this.as)))}cloneMsgArg(){const e=this.ma.subject.length,t=this.ma.reply?this.ma.reply.length:0,s=new Uint8Array(e+t);s.set(this.ma.subject),this.ma.reply&&s.set(this.ma.reply,e),this.argBuf=new Ds(s),this.ma.subject=s.subarray(0,e),this.ma.reply&&(this.ma.reply=s.subarray(e))}processMsgArgs(e){if(this.hdr>=0)return this.processHeaderMsgArgs(e);const t=[];let s=-1;for(let i=0;i<e.length;i++)switch(e[i]){case A.SPACE:case A.TAB:case A.CR:case A.NL:s>=0&&(t.push(e.subarray(s,i)),s=-1);break;default:s<0&&(s=i)}switch(s>=0&&t.push(e.subarray(s)),t.length){case 3:this.ma.subject=t[0],this.ma.sid=this.protoParseInt(t[1]),this.ma.reply=void 0,this.ma.size=this.protoParseInt(t[2]);break;case 4:this.ma.subject=t[0],this.ma.sid=this.protoParseInt(t[1]),this.ma.reply=t[2],this.ma.size=this.protoParseInt(t[3]);break;default:throw this.fail(e,"processMsgArgs Parse Error")}if(this.ma.sid<0)throw this.fail(e,"processMsgArgs Bad or Missing Sid Error");if(this.ma.size<0)throw this.fail(e,"processMsgArgs Bad or Missing Size Error")}fail(e,t=""){return t?t=`${t} [${this.state}]`:t=`parse error [${this.state}]`,new Error(`${t}: ${ke.decode(e)}`)}processHeaderMsgArgs(e){const t=[];let s=-1;for(let i=0;i<e.length;i++)switch(e[i]){case A.SPACE:case A.TAB:case A.CR:case A.NL:s>=0&&(t.push(e.subarray(s,i)),s=-1);break;default:s<0&&(s=i)}switch(s>=0&&t.push(e.subarray(s)),t.length){case 4:this.ma.subject=t[0],this.ma.sid=this.protoParseInt(t[1]),this.ma.reply=void 0,this.ma.hdr=this.protoParseInt(t[2]),this.ma.size=this.protoParseInt(t[3]);break;case 5:this.ma.subject=t[0],this.ma.sid=this.protoParseInt(t[1]),this.ma.reply=t[2],this.ma.hdr=this.protoParseInt(t[3]),this.ma.size=this.protoParseInt(t[4]);break;default:throw this.fail(e,"processHeaderMsgArgs Parse Error")}if(this.ma.sid<0)throw this.fail(e,"processHeaderMsgArgs Bad or Missing Sid Error");if(this.ma.hdr<0||this.ma.hdr>this.ma.size)throw this.fail(e,"processHeaderMsgArgs Bad or Missing Header Size Error");if(this.ma.size<0)throw this.fail(e,"processHeaderMsgArgs Bad or Missing Size Error")}protoParseInt(e){if(e.length===0)return-1;let t=0;for(let s=0;s<e.length;s++){if(e[s]<48||e[s]>57)return-1;t=t*10+(e[s]-ha)}return t}}var I;(function(r){r[r.OP_START=0]="OP_START",r[r.OP_PLUS=1]="OP_PLUS",r[r.OP_PLUS_O=2]="OP_PLUS_O",r[r.OP_PLUS_OK=3]="OP_PLUS_OK",r[r.OP_MINUS=4]="OP_MINUS",r[r.OP_MINUS_E=5]="OP_MINUS_E",r[r.OP_MINUS_ER=6]="OP_MINUS_ER",r[r.OP_MINUS_ERR=7]="OP_MINUS_ERR",r[r.OP_MINUS_ERR_SPC=8]="OP_MINUS_ERR_SPC",r[r.MINUS_ERR_ARG=9]="MINUS_ERR_ARG",r[r.OP_M=10]="OP_M",r[r.OP_MS=11]="OP_MS",r[r.OP_MSG=12]="OP_MSG",r[r.OP_MSG_SPC=13]="OP_MSG_SPC",r[r.MSG_ARG=14]="MSG_ARG",r[r.MSG_PAYLOAD=15]="MSG_PAYLOAD",r[r.MSG_END=16]="MSG_END",r[r.OP_H=17]="OP_H",r[r.OP_P=18]="OP_P",r[r.OP_PI=19]="OP_PI",r[r.OP_PIN=20]="OP_PIN",r[r.OP_PING=21]="OP_PING",r[r.OP_PO=22]="OP_PO",r[r.OP_PON=23]="OP_PON",r[r.OP_PONG=24]="OP_PONG",r[r.OP_I=25]="OP_I",r[r.OP_IN=26]="OP_IN",r[r.OP_INF=27]="OP_INF",r[r.OP_INFO=28]="OP_INFO",r[r.OP_INFO_SPC=29]="OP_INFO_SPC",r[r.INFO_ARG=30]="INFO_ARG"})(I||(I={}));var A;(function(r){r[r.CR=13]="CR",r[r.E=69]="E",r[r.e=101]="e",r[r.F=70]="F",r[r.f=102]="f",r[r.G=71]="G",r[r.g=103]="g",r[r.H=72]="H",r[r.h=104]="h",r[r.I=73]="I",r[r.i=105]="i",r[r.K=75]="K",r[r.k=107]="k",r[r.M=77]="M",r[r.m=109]="m",r[r.MINUS=45]="MINUS",r[r.N=78]="N",r[r.n=110]="n",r[r.NL=10]="NL",r[r.O=79]="O",r[r.o=111]="o",r[r.P=80]="P",r[r.p=112]="p",r[r.PLUS=43]="PLUS",r[r.R=82]="R",r[r.r=114]="r",r[r.S=83]="S",r[r.s=115]="s",r[r.SPACE=32]="SPACE",r[r.TAB=9]="TAB"})(A||(A={}));(function(r){var e=function(o,h){this.hi=o|0,this.lo=h|0},t=function(o){var h,u=new Float64Array(16);if(o)for(h=0;h<o.length;h++)u[h]=o[h];return u},s=function(){throw new Error("no PRNG")},i=new Uint8Array(16),n=new Uint8Array(32);n[0]=9;var c=t(),a=t([1]),d=t([56129,1]),m=t([30883,4953,19914,30187,55467,16705,2637,112,59544,30585,16505,36039,65139,11119,27886,20995]),x=t([61785,9906,39828,60374,45398,33411,5274,224,53552,61171,33010,6542,64743,22239,55772,9222]),v=t([54554,36645,11616,51542,42930,38181,51040,26924,56412,64982,57905,49316,21502,52590,14035,8553]),S=t([26200,26214,26214,26214,26214,26214,26214,26214,26214,26214,26214,26214,26214,26214,26214,26214]),O=t([41136,18958,6951,50414,58488,44335,6150,12099,55207,15867,153,11085,57099,20417,9344,11139]);function $(o,h){return o<<h|o>>>32-h}function K(o,h){var u=o[h+3]&255;return u=u<<8|o[h+2]&255,u=u<<8|o[h+1]&255,u<<8|o[h+0]&255}function ee(o,h){var u=o[h]<<24|o[h+1]<<16|o[h+2]<<8|o[h+3],l=o[h+4]<<24|o[h+5]<<16|o[h+6]<<8|o[h+7];return new e(u,l)}function B(o,h,u){var l;for(l=0;l<4;l++)o[h+l]=u&255,u>>>=8}function D(o,h,u){o[h]=u.hi>>24&255,o[h+1]=u.hi>>16&255,o[h+2]=u.hi>>8&255,o[h+3]=u.hi&255,o[h+4]=u.lo>>24&255,o[h+5]=u.lo>>16&255,o[h+6]=u.lo>>8&255,o[h+7]=u.lo&255}function de(o,h,u,l,f){var p,w=0;for(p=0;p<f;p++)w|=o[h+p]^u[l+p];return(1&w-1>>>8)-1}function le(o,h,u,l){return de(o,h,u,l,16)}function g(o,h,u,l){return de(o,h,u,l,32)}function R(o,h,u,l,f){var p=new Uint32Array(16),w=new Uint32Array(16),_=new Uint32Array(16),b=new Uint32Array(4),y,C,L;for(y=0;y<4;y++)w[5*y]=K(l,4*y),w[1+y]=K(u,4*y),w[6+y]=K(h,4*y),w[11+y]=K(u,16+4*y);for(y=0;y<16;y++)_[y]=w[y];for(y=0;y<20;y++){for(C=0;C<4;C++){for(L=0;L<4;L++)b[L]=w[(5*C+4*L)%16];for(b[1]^=$(b[0]+b[3]|0,7),b[2]^=$(b[1]+b[0]|0,9),b[3]^=$(b[2]+b[1]|0,13),b[0]^=$(b[3]+b[2]|0,18),L=0;L<4;L++)p[4*C+(C+L)%4]=b[L]}for(L=0;L<16;L++)w[L]=p[L]}if(f){for(y=0;y<16;y++)w[y]=w[y]+_[y]|0;for(y=0;y<4;y++)w[5*y]=w[5*y]-K(l,4*y)|0,w[6+y]=w[6+y]-K(h,4*y)|0;for(y=0;y<4;y++)B(o,4*y,w[5*y]),B(o,16+4*y,w[6+y])}else for(y=0;y<16;y++)B(o,4*y,w[y]+_[y]|0)}function q(o,h,u,l){return R(o,h,u,l,!1),0}function P(o,h,u,l){return R(o,h,u,l,!0),0}var k=new Uint8Array([101,120,112,97,110,100,32,51,50,45,98,121,116,101,32,107]);function N(o,h,u,l,f,p,w){var _=new Uint8Array(16),b=new Uint8Array(64),y,C;if(!f)return 0;for(C=0;C<16;C++)_[C]=0;for(C=0;C<8;C++)_[C]=p[C];for(;f>=64;){for(q(b,_,w,k),C=0;C<64;C++)o[h+C]=(u?u[l+C]:0)^b[C];for(y=1,C=8;C<16;C++)y=y+(_[C]&255)|0,_[C]=y&255,y>>>=8;f-=64,h+=64,u&&(l+=64)}if(f>0)for(q(b,_,w,k),C=0;C<f;C++)o[h+C]=(u?u[l+C]:0)^b[C];return 0}function M(o,h,u,l,f){return N(o,h,null,0,u,l,f)}function T(o,h,u,l,f){var p=new Uint8Array(32);return P(p,l,f,k),M(o,h,u,l.subarray(16),p)}function G(o,h,u,l,f,p,w){var _=new Uint8Array(32);return P(_,p,w,k),N(o,h,u,l,f,p.subarray(16),_)}function te(o,h){var u,l=0;for(u=0;u<17;u++)l=l+(o[u]+h[u]|0)|0,o[u]=l&255,l>>>=8}var re=new Uint32Array([5,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,252]);function se(o,h,u,l,f,p){var w,_,b,y,C=new Uint32Array(17),L=new Uint32Array(17),X=new Uint32Array(17),Ce=new Uint32Array(17),ht=new Uint32Array(17);for(b=0;b<17;b++)L[b]=X[b]=0;for(b=0;b<16;b++)L[b]=p[b];for(L[3]&=15,L[4]&=252,L[7]&=15,L[8]&=252,L[11]&=15,L[12]&=252,L[15]&=15;f>0;){for(b=0;b<17;b++)Ce[b]=0;for(b=0;b<16&&b<f;++b)Ce[b]=u[l+b];for(Ce[b]=1,l+=b,f-=b,te(X,Ce),_=0;_<17;_++)for(C[_]=0,b=0;b<17;b++)C[_]=C[_]+X[b]*(b<=_?L[_-b]:320*L[_+17-b]|0)|0|0;for(_=0;_<17;_++)X[_]=C[_];for(y=0,b=0;b<16;b++)y=y+X[b]|0,X[b]=y&255,y>>>=8;for(y=y+X[16]|0,X[16]=y&3,y=5*(y>>>2)|0,b=0;b<16;b++)y=y+X[b]|0,X[b]=y&255,y>>>=8;y=y+X[16]|0,X[16]=y}for(b=0;b<17;b++)ht[b]=X[b];for(te(X,re),w=-(X[16]>>>7)|0,b=0;b<17;b++)X[b]^=w&(ht[b]^X[b]);for(b=0;b<16;b++)Ce[b]=p[b+16];for(Ce[16]=0,te(X,Ce),b=0;b<16;b++)o[h+b]=X[b];return 0}function ie(o,h,u,l,f,p){var w=new Uint8Array(16);return se(w,0,u,l,f,p),le(o,h,w,0)}function ce(o,h,u,l,f){var p;if(u<32)return-1;for(G(o,0,h,0,u,l,f),se(o,16,o,32,u-32,o),p=0;p<16;p++)o[p]=0;return 0}function ue(o,h,u,l,f){var p,w=new Uint8Array(32);if(u<32||(T(w,0,32,l,f),ie(h,16,h,32,u-32,w)!==0))return-1;for(G(o,0,h,0,u,l,f),p=0;p<32;p++)o[p]=0;return 0}function Y(o,h){var u;for(u=0;u<16;u++)o[u]=h[u]|0}function Z(o){var h,u;for(u=0;u<16;u++)o[u]+=65536,h=Math.floor(o[u]/65536),o[(u+1)*(u<15?1:0)]+=h-1+37*(h-1)*(u===15?1:0),o[u]-=h*65536}function ne(o,h,u){for(var l,f=~(u-1),p=0;p<16;p++)l=f&(o[p]^h[p]),o[p]^=l,h[p]^=l}function ot(o,h){var u,l,f,p=t(),w=t();for(u=0;u<16;u++)w[u]=h[u];for(Z(w),Z(w),Z(w),l=0;l<2;l++){for(p[0]=w[0]-65517,u=1;u<15;u++)p[u]=w[u]-65535-(p[u-1]>>16&1),p[u-1]&=65535;p[15]=w[15]-32767-(p[14]>>16&1),f=p[15]>>16&1,p[14]&=65535,ne(w,p,1-f)}for(u=0;u<16;u++)o[2*u]=w[u]&255,o[2*u+1]=w[u]>>8}function br(o,h){var u=new Uint8Array(32),l=new Uint8Array(32);return ot(u,o),ot(l,h),g(u,0,l,0)}function gr(o){var h=new Uint8Array(32);return ot(h,o),h[0]&1}function As(o,h){var u;for(u=0;u<16;u++)o[u]=h[2*u]+(h[2*u+1]<<8);o[15]&=32767}function je(o,h,u){var l;for(l=0;l<16;l++)o[l]=h[l]+u[l]|0}function Oe(o,h,u){var l;for(l=0;l<16;l++)o[l]=h[l]-u[l]|0}function F(o,h,u){var l,f,p=new Float64Array(31);for(l=0;l<31;l++)p[l]=0;for(l=0;l<16;l++)for(f=0;f<16;f++)p[l+f]+=h[l]*u[f];for(l=0;l<15;l++)p[l]+=38*p[l+16];for(l=0;l<16;l++)o[l]=p[l];Z(o),Z(o)}function Ae(o,h){F(o,h,h)}function yr(o,h){var u=t(),l;for(l=0;l<16;l++)u[l]=h[l];for(l=253;l>=0;l--)Ae(u,u),l!==2&&l!==4&&F(u,u,h);for(l=0;l<16;l++)o[l]=u[l]}function wr(o,h){var u=t(),l;for(l=0;l<16;l++)u[l]=h[l];for(l=250;l>=0;l--)Ae(u,u),l!==1&&F(u,u,h);for(l=0;l<16;l++)o[l]=u[l]}function Wt(o,h,u){var l=new Uint8Array(32),f=new Float64Array(80),p,w,_=t(),b=t(),y=t(),C=t(),L=t(),X=t();for(w=0;w<31;w++)l[w]=h[w];for(l[31]=h[31]&127|64,l[0]&=248,As(f,u),w=0;w<16;w++)b[w]=f[w],C[w]=_[w]=y[w]=0;for(_[0]=C[0]=1,w=254;w>=0;--w)p=l[w>>>3]>>>(w&7)&1,ne(_,b,p),ne(y,C,p),je(L,_,y),Oe(_,_,y),je(y,b,C),Oe(b,b,C),Ae(C,L),Ae(X,_),F(_,y,_),F(y,b,L),je(L,_,y),Oe(_,_,y),Ae(b,_),Oe(y,C,X),F(_,y,d),je(_,_,C),F(y,y,_),F(_,C,X),F(C,b,f),Ae(b,L),ne(_,b,p),ne(y,C,p);for(w=0;w<16;w++)f[w+16]=_[w],f[w+32]=y[w],f[w+48]=b[w],f[w+64]=C[w];var Ce=f.subarray(32),ht=f.subarray(16);return yr(Ce,Ce),F(ht,ht,Ce),ot(o,ht),0}function Yt(o,h){return Wt(o,h,n)}function xr(o,h){return s(h,32),Yt(o,h)}function Xt(o,h,u){var l=new Uint8Array(32);return Wt(l,u,h),P(o,i,l,k)}var _r=ce,Ti=ue;function $i(o,h,u,l,f,p){var w=new Uint8Array(32);return Xt(w,f,p),_r(o,h,u,l,w)}function qi(o,h,u,l,f,p){var w=new Uint8Array(32);return Xt(w,f,p),Ti(o,h,u,l,w)}function It(){var o=0,h=0,u=0,l=0,f=65535,p,w,_;for(_=0;_<arguments.length;_++)p=arguments[_].lo,w=arguments[_].hi,o+=p&f,h+=p>>>16,u+=w&f,l+=w>>>16;return h+=o>>>16,u+=h>>>16,l+=u>>>16,new e(u&f|l<<16,o&f|h<<16)}function vr(o,h){return new e(o.hi>>>h,o.lo>>>h|o.hi<<32-h)}function Zt(){var o=0,h=0,u;for(u=0;u<arguments.length;u++)o^=arguments[u].lo,h^=arguments[u].hi;return new e(h,o)}function qe(o,h){var u,l,f=32-h;return h<32?(u=o.hi>>>h|o.lo<<f,l=o.lo>>>h|o.hi<<f):h<64&&(u=o.lo>>>h|o.hi<<f,l=o.hi>>>h|o.lo<<f),new e(u,l)}function Fi(o,h,u){var l=o.hi&h.hi^~o.hi&u.hi,f=o.lo&h.lo^~o.lo&u.lo;return new e(l,f)}function Li(o,h,u){var l=o.hi&h.hi^o.hi&u.hi^h.hi&u.hi,f=o.lo&h.lo^o.lo&u.lo^h.lo&u.lo;return new e(l,f)}function Ui(o){return Zt(qe(o,28),qe(o,34),qe(o,39))}function Bi(o){return Zt(qe(o,14),qe(o,18),qe(o,41))}function Di(o){return Zt(qe(o,1),qe(o,8),vr(o,7))}function Hi(o){return Zt(qe(o,19),qe(o,61),vr(o,6))}var zi=[new e(1116352408,3609767458),new e(1899447441,602891725),new e(3049323471,3964484399),new e(3921009573,2173295548),new e(961987163,4081628472),new e(1508970993,3053834265),new e(2453635748,2937671579),new e(2870763221,3664609560),new e(3624381080,2734883394),new e(310598401,1164996542),new e(607225278,1323610764),new e(1426881987,3590304994),new e(1925078388,4068182383),new e(2162078206,991336113),new e(2614888103,633803317),new e(3248222580,3479774868),new e(3835390401,2666613458),new e(4022224774,944711139),new e(264347078,2341262773),new e(604807628,2007800933),new e(770255983,1495990901),new e(1249150122,1856431235),new e(1555081692,3175218132),new e(1996064986,2198950837),new e(2554220882,3999719339),new e(2821834349,766784016),new e(2952996808,2566594879),new e(3210313671,3203337956),new e(3336571891,1034457026),new e(3584528711,2466948901),new e(113926993,3758326383),new e(338241895,168717936),new e(666307205,1188179964),new e(773529912,1546045734),new e(1294757372,1522805485),new e(1396182291,2643833823),new e(1695183700,2343527390),new e(1986661051,1014477480),new e(2177026350,1206759142),new e(2456956037,344077627),new e(2730485921,1290863460),new e(2820302411,3158454273),new e(3259730800,3505952657),new e(3345764771,106217008),new e(3516065817,3606008344),new e(3600352804,1432725776),new e(4094571909,1467031594),new e(275423344,851169720),new e(430227734,3100823752),new e(506948616,1363258195),new e(659060556,3750685593),new e(883997877,3785050280),new e(958139571,3318307427),new e(1322822218,3812723403),new e(1537002063,2003034995),new e(1747873779,3602036899),new e(1955562222,1575990012),new e(2024104815,1125592928),new e(2227730452,2716904306),new e(2361852424,442776044),new e(2428436474,593698344),new e(2756734187,3733110249),new e(3204031479,2999351573),new e(3329325298,3815920427),new e(3391569614,3928383900),new e(3515267271,566280711),new e(3940187606,3454069534),new e(4118630271,4000239992),new e(116418474,1914138554),new e(174292421,2731055270),new e(289380356,3203993006),new e(460393269,320620315),new e(685471733,587496836),new e(852142971,1086792851),new e(1017036298,365543100),new e(1126000580,2618297676),new e(1288033470,3409855158),new e(1501505948,4234509866),new e(1607167915,987167468),new e(1816402316,1246189591)];function Sr(o,h,u){var l=[],f=[],p=[],w=[],_,b,y;for(b=0;b<8;b++)l[b]=p[b]=ee(o,8*b);for(var C=0;u>=128;){for(b=0;b<16;b++)w[b]=ee(h,8*b+C);for(b=0;b<80;b++){for(y=0;y<8;y++)f[y]=p[y];for(_=It(p[7],Bi(p[4]),Fi(p[4],p[5],p[6]),zi[b],w[b%16]),f[7]=It(_,Ui(p[0]),Li(p[0],p[1],p[2])),f[3]=It(f[3],_),y=0;y<8;y++)p[(y+1)%8]=f[y];if(b%16===15)for(y=0;y<16;y++)w[y]=It(w[y],w[(y+9)%16],Di(w[(y+1)%16]),Hi(w[(y+14)%16]))}for(b=0;b<8;b++)p[b]=It(p[b],l[b]),l[b]=p[b];C+=128,u-=128}for(b=0;b<8;b++)D(o,8*b,l[b]);return u}var Ji=new Uint8Array([106,9,230,103,243,188,201,8,187,103,174,133,132,202,167,59,60,110,243,114,254,148,248,43,165,79,245,58,95,29,54,241,81,14,82,127,173,230,130,209,155,5,104,140,43,62,108,31,31,131,217,171,251,65,189,107,91,224,205,25,19,126,33,121]);function tt(o,h,u){var l=new Uint8Array(64),f=new Uint8Array(256),p,w=u;for(p=0;p<64;p++)l[p]=Ji[p];for(Sr(l,h,u),u%=128,p=0;p<256;p++)f[p]=0;for(p=0;p<u;p++)f[p]=h[w-u+p];for(f[u]=128,u=256-128*(u<112?1:0),f[u-9]=0,D(f,u-8,new e(w/536870912|0,w<<3)),Sr(l,f,u),p=0;p<64;p++)o[p]=l[p];return 0}function Qt(o,h){var u=t(),l=t(),f=t(),p=t(),w=t(),_=t(),b=t(),y=t(),C=t();Oe(u,o[1],o[0]),Oe(C,h[1],h[0]),F(u,u,C),je(l,o[0],o[1]),je(C,h[0],h[1]),F(l,l,C),F(f,o[3],h[3]),F(f,f,x),F(p,o[2],h[2]),je(p,p,p),Oe(w,l,u),Oe(_,p,f),je(b,p,f),je(y,l,u),F(o[0],w,_),F(o[1],y,b),F(o[2],b,_),F(o[3],w,y)}function kr(o,h,u){var l;for(l=0;l<4;l++)ne(o[l],h[l],u)}function Ps(o,h){var u=t(),l=t(),f=t();yr(f,h[2]),F(u,h[0],f),F(l,h[1],f),ot(o,l),o[31]^=gr(u)<<7}function Rs(o,h,u){var l,f;for(Y(o[0],c),Y(o[1],a),Y(o[2],a),Y(o[3],c),f=255;f>=0;--f)l=u[f/8|0]>>(f&7)&1,kr(o,h,l),Qt(h,o),Qt(o,o),kr(o,h,l)}function es(o,h){var u=[t(),t(),t(),t()];Y(u[0],v),Y(u[1],S),Y(u[2],a),F(u[3],v,S),Rs(o,u,h)}function js(o,h,u){var l=new Uint8Array(64),f=[t(),t(),t(),t()],p;for(u||s(h,32),tt(l,h,32),l[0]&=248,l[31]&=127,l[31]|=64,es(f,l),Ps(o,f),p=0;p<32;p++)h[p+32]=o[p];return 0}var ts=new Float64Array([237,211,245,92,26,99,18,88,214,156,247,162,222,249,222,20,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,16]);function Os(o,h){var u,l,f,p;for(l=63;l>=32;--l){for(u=0,f=l-32,p=l-12;f<p;++f)h[f]+=u-16*h[l]*ts[f-(l-32)],u=Math.floor((h[f]+128)/256),h[f]-=u*256;h[f]+=u,h[l]=0}for(u=0,f=0;f<32;f++)h[f]+=u-(h[31]>>4)*ts[f],u=h[f]>>8,h[f]&=255;for(f=0;f<32;f++)h[f]-=u*ts[f];for(l=0;l<32;l++)h[l+1]+=h[l]>>8,o[l]=h[l]&255}function Cs(o){var h=new Float64Array(64),u;for(u=0;u<64;u++)h[u]=o[u];for(u=0;u<64;u++)o[u]=0;Os(o,h)}function Er(o,h,u,l){var f=new Uint8Array(64),p=new Uint8Array(64),w=new Uint8Array(64),_,b,y=new Float64Array(64),C=[t(),t(),t(),t()];tt(f,l,32),f[0]&=248,f[31]&=127,f[31]|=64;var L=u+64;for(_=0;_<u;_++)o[64+_]=h[_];for(_=0;_<32;_++)o[32+_]=f[32+_];for(tt(w,o.subarray(32),u+32),Cs(w),es(C,w),Ps(o,C),_=32;_<64;_++)o[_]=l[_];for(tt(p,o,u+64),Cs(p),_=0;_<64;_++)y[_]=0;for(_=0;_<32;_++)y[_]=w[_];for(_=0;_<32;_++)for(b=0;b<32;b++)y[_+b]+=p[_]*f[b];return Os(o.subarray(32),y),L}function Ki(o,h){var u=t(),l=t(),f=t(),p=t(),w=t(),_=t(),b=t();return Y(o[2],a),As(o[1],h),Ae(f,o[1]),F(p,f,m),Oe(f,f,o[2]),je(p,o[2],p),Ae(w,p),Ae(_,w),F(b,_,w),F(u,b,f),F(u,u,p),wr(u,u),F(u,u,f),F(u,u,p),F(u,u,p),F(o[0],u,p),Ae(l,o[0]),F(l,l,p),br(l,f)&&F(o[0],o[0],O),Ae(l,o[0]),F(l,l,p),br(l,f)?-1:(gr(o[0])===h[31]>>7&&Oe(o[0],c,o[0]),F(o[3],o[0],o[1]),0)}function Ns(o,h,u,l){var f,p=new Uint8Array(32),w=new Uint8Array(64),_=[t(),t(),t(),t()],b=[t(),t(),t(),t()];if(u<64||Ki(b,l))return-1;for(f=0;f<u;f++)o[f]=h[f];for(f=0;f<32;f++)o[f+32]=l[f];if(tt(w,o,u),Cs(w),Rs(_,b,w),es(b,h.subarray(32)),Qt(_,b),Ps(p,_),u-=64,g(h,0,p,0)){for(f=0;f<u;f++)o[f]=0;return-1}for(f=0;f<u;f++)o[f]=h[f+64];return u}var Ms=32,ss=24,At=32,ct=16,Pt=32,rs=32,Rt=32,jt=32,Ts=32,Ir=ss,Gi=At,Vi=ct,Ue=64,st=32,ut=64,$s=32,qs=64;r.lowlevel={crypto_core_hsalsa20:P,crypto_stream_xor:G,crypto_stream:T,crypto_stream_salsa20_xor:N,crypto_stream_salsa20:M,crypto_onetimeauth:se,crypto_onetimeauth_verify:ie,crypto_verify_16:le,crypto_verify_32:g,crypto_secretbox:ce,crypto_secretbox_open:ue,crypto_scalarmult:Wt,crypto_scalarmult_base:Yt,crypto_box_beforenm:Xt,crypto_box_afternm:_r,crypto_box:$i,crypto_box_open:qi,crypto_box_keypair:xr,crypto_hash:tt,crypto_sign:Er,crypto_sign_keypair:js,crypto_sign_open:Ns,crypto_secretbox_KEYBYTES:Ms,crypto_secretbox_NONCEBYTES:ss,crypto_secretbox_ZEROBYTES:At,crypto_secretbox_BOXZEROBYTES:ct,crypto_scalarmult_BYTES:Pt,crypto_scalarmult_SCALARBYTES:rs,crypto_box_PUBLICKEYBYTES:Rt,crypto_box_SECRETKEYBYTES:jt,crypto_box_BEFORENMBYTES:Ts,crypto_box_NONCEBYTES:Ir,crypto_box_ZEROBYTES:Gi,crypto_box_BOXZEROBYTES:Vi,crypto_sign_BYTES:Ue,crypto_sign_PUBLICKEYBYTES:st,crypto_sign_SECRETKEYBYTES:ut,crypto_sign_SEEDBYTES:$s,crypto_hash_BYTES:qs,gf:t,D:m,L:ts,pack25519:ot,unpack25519:As,M:F,A:je,S:Ae,Z:Oe,pow2523:wr,add:Qt,set25519:Y,modL:Os,scalarmult:Rs,scalarbase:es};function Ar(o,h){if(o.length!==Ms)throw new Error("bad key size");if(h.length!==ss)throw new Error("bad nonce size")}function Wi(o,h){if(o.length!==Rt)throw new Error("bad public key size");if(h.length!==jt)throw new Error("bad secret key size")}function we(){for(var o=0;o<arguments.length;o++)if(!(arguments[o]instanceof Uint8Array))throw new TypeError("unexpected type, use Uint8Array")}function Pr(o){for(var h=0;h<o.length;h++)o[h]=0}r.randomBytes=function(o){var h=new Uint8Array(o);return s(h,o),h},r.secretbox=function(o,h,u){we(o,h,u),Ar(u,h);for(var l=new Uint8Array(At+o.length),f=new Uint8Array(l.length),p=0;p<o.length;p++)l[p+At]=o[p];return ce(f,l,l.length,h,u),f.subarray(ct)},r.secretbox.open=function(o,h,u){we(o,h,u),Ar(u,h);for(var l=new Uint8Array(ct+o.length),f=new Uint8Array(l.length),p=0;p<o.length;p++)l[p+ct]=o[p];return l.length<32||ue(f,l,l.length,h,u)!==0?null:f.subarray(At)},r.secretbox.keyLength=Ms,r.secretbox.nonceLength=ss,r.secretbox.overheadLength=ct,r.scalarMult=function(o,h){if(we(o,h),o.length!==rs)throw new Error("bad n size");if(h.length!==Pt)throw new Error("bad p size");var u=new Uint8Array(Pt);return Wt(u,o,h),u},r.scalarMult.base=function(o){if(we(o),o.length!==rs)throw new Error("bad n size");var h=new Uint8Array(Pt);return Yt(h,o),h},r.scalarMult.scalarLength=rs,r.scalarMult.groupElementLength=Pt,r.box=function(o,h,u,l){var f=r.box.before(u,l);return r.secretbox(o,h,f)},r.box.before=function(o,h){we(o,h),Wi(o,h);var u=new Uint8Array(Ts);return Xt(u,o,h),u},r.box.after=r.secretbox,r.box.open=function(o,h,u,l){var f=r.box.before(u,l);return r.secretbox.open(o,h,f)},r.box.open.after=r.secretbox.open,r.box.keyPair=function(){var o=new Uint8Array(Rt),h=new Uint8Array(jt);return xr(o,h),{publicKey:o,secretKey:h}},r.box.keyPair.fromSecretKey=function(o){if(we(o),o.length!==jt)throw new Error("bad secret key size");var h=new Uint8Array(Rt);return Yt(h,o),{publicKey:h,secretKey:new Uint8Array(o)}},r.box.publicKeyLength=Rt,r.box.secretKeyLength=jt,r.box.sharedKeyLength=Ts,r.box.nonceLength=Ir,r.box.overheadLength=r.secretbox.overheadLength,r.sign=function(o,h){if(we(o,h),h.length!==ut)throw new Error("bad secret key size");var u=new Uint8Array(Ue+o.length);return Er(u,o,o.length,h),u},r.sign.open=function(o,h){if(we(o,h),h.length!==st)throw new Error("bad public key size");var u=new Uint8Array(o.length),l=Ns(u,o,o.length,h);if(l<0)return null;for(var f=new Uint8Array(l),p=0;p<f.length;p++)f[p]=u[p];return f},r.sign.detached=function(o,h){for(var u=r.sign(o,h),l=new Uint8Array(Ue),f=0;f<l.length;f++)l[f]=u[f];return l},r.sign.detached.verify=function(o,h,u){if(we(o,h,u),h.length!==Ue)throw new Error("bad signature size");if(u.length!==st)throw new Error("bad public key size");var l=new Uint8Array(Ue+o.length),f=new Uint8Array(Ue+o.length),p;for(p=0;p<Ue;p++)l[p]=h[p];for(p=0;p<o.length;p++)l[p+Ue]=o[p];return Ns(f,l,l.length,u)>=0},r.sign.keyPair=function(){var o=new Uint8Array(st),h=new Uint8Array(ut);return js(o,h),{publicKey:o,secretKey:h}},r.sign.keyPair.fromSecretKey=function(o){if(we(o),o.length!==ut)throw new Error("bad secret key size");for(var h=new Uint8Array(st),u=0;u<h.length;u++)h[u]=o[32+u];return{publicKey:h,secretKey:new Uint8Array(o)}},r.sign.keyPair.fromSeed=function(o){if(we(o),o.length!==$s)throw new Error("bad seed size");for(var h=new Uint8Array(st),u=new Uint8Array(ut),l=0;l<32;l++)u[l]=o[l];return js(h,u,!0),{publicKey:h,secretKey:u}},r.sign.publicKeyLength=st,r.sign.secretKeyLength=ut,r.sign.seedLength=$s,r.sign.signatureLength=Ue,r.hash=function(o){we(o);var h=new Uint8Array(qs);return tt(h,o,o.length),h},r.hash.hashLength=qs,r.verify=function(o,h){return we(o,h),o.length===0||h.length===0||o.length!==h.length?!1:de(o,0,h,0,o.length)===0},r.setPRNG=function(o){s=o},(function(){var o=typeof globalThis<"u"?globalThis.crypto||globalThis.msCrypto:null;if(o&&o.getRandomValues){var h=65536;r.setPRNG(function(u,l){var f,p=new Uint8Array(l);for(f=0;f<l;f+=h)o.getRandomValues(p.subarray(f,f+Math.min(l-f,h)));for(f=0;f<l;f++)u[f]=p[f];Pr(p)})}else typeof require<"u"&&(o=require("crypto"),o&&o.randomBytes&&r.setPRNG(function(u,l){var f,p=o.randomBytes(l);for(f=0;f<l;f++)u[f]=p[f];Pr(p)}))})()})(typeof at<"u"&&at.exports?at.exports:globalThis.nacl=globalThis.nacl||{});const as=typeof at<"u"&&at.exports?at.exports:globalThis.nacl,la={fromSeed:as.sign.keyPair.fromSeed,sign:as.sign.detached,verify:as.sign.detached.verify,randomBytes:as.randomBytes};let bi;function da(r){bi=r}function Je(){return bi}const fa=new Uint16Array([0,4129,8258,12387,16516,20645,24774,28903,33032,37161,41290,45419,49548,53677,57806,61935,4657,528,12915,8786,21173,17044,29431,25302,37689,33560,45947,41818,54205,50076,62463,58334,9314,13379,1056,5121,25830,29895,17572,21637,42346,46411,34088,38153,58862,62927,50604,54669,13907,9842,5649,1584,30423,26358,22165,18100,46939,42874,38681,34616,63455,59390,55197,51132,18628,22757,26758,30887,2112,6241,10242,14371,51660,55789,59790,63919,35144,39273,43274,47403,23285,19156,31415,27286,6769,2640,14899,10770,56317,52188,64447,60318,39801,35672,47931,43802,27814,31879,19684,23749,11298,15363,3168,7233,60846,64911,52716,56781,44330,48395,36200,40265,32407,28342,24277,20212,15891,11826,7761,3696,65439,61374,57309,53244,48923,44858,40793,36728,37256,33193,45514,41451,53516,49453,61774,57711,4224,161,12482,8419,20484,16421,28742,24679,33721,37784,41979,46042,49981,54044,58239,62302,689,4752,8947,13010,16949,21012,25207,29270,46570,42443,38312,34185,62830,58703,54572,50445,13538,9411,5280,1153,29798,25671,21540,17413,42971,47098,34713,38840,59231,63358,50973,55100,9939,14066,1681,5808,26199,30326,17941,22068,55628,51565,63758,59695,39368,35305,47498,43435,22596,18533,30726,26663,6336,2273,14466,10403,52093,56156,60223,64286,35833,39896,43963,48026,19061,23124,27191,31254,2801,6864,10931,14994,64814,60687,56684,52557,48554,44427,40424,36297,31782,27655,23652,19525,15522,11395,7392,3265,61215,65342,53085,57212,44955,49082,36825,40952,28183,32310,20053,24180,11923,16050,3793,7920]);class ys{static checksum(e){let t=0;for(let s=0;s<e.byteLength;s++){let i=e[s];t=t<<8&65535^fa[(t>>8^i)&255]}return t}static validate(e,t){return ys.checksum(e)==t}}const Hs="ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";class Gr{static encode(e){let t=0,s=0,i=new Uint8Array(e),n=new Uint8Array(e.byteLength*2),c=0;for(let a=0;a<i.byteLength;a++)for(s=s<<8|i[a],t+=8;t>=5;){let d=s>>>t-5&31;n[c++]=Hs.charAt(d).charCodeAt(0),t-=5}if(t>0){let a=s<<5-t&31;n[c++]=Hs.charAt(a).charCodeAt(0)}return n.slice(0,c)}static decode(e){let t=0,s=0,i=0,n=new Uint8Array(e),c=new Uint8Array(n.byteLength*5/8|0);for(let a=0;a<n.byteLength;a++){let d=String.fromCharCode(n[a]),m=Hs.indexOf(d);if(m===-1)throw new Error("Illegal Base32 character: "+n[a]);s=s<<5|m,t+=5,t>=8&&(c[i++]=s>>>t-8&255,t-=8)}return c.slice(0,i)}}class J extends Error{name;code;chainedError;constructor(e,t){super(e),this.name="NKeysError",this.code=e,this.chainedError=t}}function pa(){return ks(z.Operator)}function ma(){return ks(z.Account)}function ba(){return ks(z.User)}var H;(function(r){r.InvalidPrefixByte="nkeys: invalid prefix byte",r.InvalidKey="nkeys: invalid key",r.InvalidPublicKey="nkeys: invalid public key",r.InvalidSeedLen="nkeys: invalid seed length",r.InvalidSeed="nkeys: invalid seed",r.InvalidEncoding="nkeys: invalid encoded key",r.InvalidSignature="nkeys: signature verification failed",r.CannotSign="nkeys: cannot sign, no private key available",r.PublicKeyOnly="nkeys: no seed or private key available",r.InvalidChecksum="nkeys: invalid checksum",r.SerializationError="nkeys: serialization error",r.ApiError="nkeys: api error",r.ClearedPair="nkeys: pair is cleared"})(H||(H={}));var z;(function(r){r[r.Seed=144]="Seed",r[r.Private=120]="Private",r[r.Operator=112]="Operator",r[r.Server=104]="Server",r[r.Cluster=16]="Cluster",r[r.Account=0]="Account",r[r.User=160]="User"})(z||(z={}));class mt{static isValidPublicPrefix(e){return e==z.Server||e==z.Operator||e==z.Cluster||e==z.Account||e==z.User}static startsWithValidPrefix(e){let t=e[0];return t=="S"||t=="P"||t=="O"||t=="N"||t=="C"||t=="A"||t=="U"}static isValidPrefix(e){return this.parsePrefix(e)!=-1}static parsePrefix(e){switch(e){case z.Seed:return z.Seed;case z.Private:return z.Private;case z.Operator:return z.Operator;case z.Server:return z.Server;case z.Cluster:return z.Cluster;case z.Account:return z.Account;case z.User:return z.User;default:return-1}}}class be{static encode(e,t){if(!t||!(t instanceof Uint8Array))throw new J(H.SerializationError);if(!mt.isValidPrefix(e))throw new J(H.InvalidPrefixByte);return be._encode(!1,e,t)}static encodeSeed(e,t){if(!t)throw new J(H.ApiError);if(!mt.isValidPublicPrefix(e))throw new J(H.InvalidPrefixByte);if(t.byteLength!==32)throw new J(H.InvalidSeedLen);return be._encode(!0,e,t)}static decode(e,t){if(!mt.isValidPrefix(e))throw new J(H.InvalidPrefixByte);const s=be._decode(t);if(s[0]!==e)throw new J(H.InvalidPrefixByte);return s.slice(1)}static decodeSeed(e){const t=be._decode(e),s=be._decodePrefix(t);if(s[0]!=z.Seed)throw new J(H.InvalidSeed);if(!mt.isValidPublicPrefix(s[1]))throw new J(H.InvalidPrefixByte);return{buf:t.slice(2),prefix:s[1]}}static _encode(e,t,s){const i=e?2:1,n=s.byteLength,c=i+n+2,a=i+n,d=new Uint8Array(c);if(e){const v=be._encodePrefix(z.Seed,t);d.set(v)}else d[0]=t;d.set(s,i);const m=ys.checksum(d.slice(0,a));return new DataView(d.buffer).setUint16(a,m,!0),Gr.encode(d)}static _decode(e){if(e.byteLength<4)throw new J(H.InvalidEncoding);let t;try{t=Gr.decode(e)}catch(a){throw new J(H.InvalidEncoding,a)}const s=t.byteLength-2,n=new DataView(t.buffer).getUint16(s,!0),c=t.slice(0,s);if(!ys.validate(c,n))throw new J(H.InvalidChecksum);return c}static _encodePrefix(e,t){const s=e|t>>5,i=(t&31)<<3;return new Uint8Array([s,i])}static _decodePrefix(e){const t=e[0]&248,s=(e[0]&7)<<5|(e[1]&248)>>3;return new Uint8Array([t,s])}}class gi{seed;constructor(e){this.seed=e}getRawSeed(){if(!this.seed)throw new J(H.ClearedPair);return be.decodeSeed(this.seed).buf}getSeed(){if(!this.seed)throw new J(H.ClearedPair);return this.seed}getPublicKey(){if(!this.seed)throw new J(H.ClearedPair);const e=be.decodeSeed(this.seed),t=Je().fromSeed(this.getRawSeed()),s=be.encode(e.prefix,t.publicKey);return new TextDecoder().decode(s)}getPrivateKey(){if(!this.seed)throw new J(H.ClearedPair);const e=Je().fromSeed(this.getRawSeed());return be.encode(z.Private,e.secretKey)}sign(e){if(!this.seed)throw new J(H.ClearedPair);const t=Je().fromSeed(this.getRawSeed());return Je().sign(e,t.secretKey)}verify(e,t){if(!this.seed)throw new J(H.ClearedPair);const s=Je().fromSeed(this.getRawSeed());return Je().verify(e,t,s.publicKey)}clear(){this.seed&&(this.seed.fill(0),this.seed=void 0)}}function ks(r){const e=Je().randomBytes(32);let t=be.encodeSeed(r,new Uint8Array(e));return new gi(t)}class ga{publicKey;constructor(e){this.publicKey=e}getPublicKey(){if(!this.publicKey)throw new J(H.ClearedPair);return new TextDecoder().decode(this.publicKey)}getPrivateKey(){throw this.publicKey?new J(H.PublicKeyOnly):new J(H.ClearedPair)}getSeed(){throw this.publicKey?new J(H.PublicKeyOnly):new J(H.ClearedPair)}sign(e){throw this.publicKey?new J(H.CannotSign):new J(H.ClearedPair)}verify(e,t){if(!this.publicKey)throw new J(H.ClearedPair);let s=be._decode(this.publicKey);return Je().verify(e,t,s.slice(1))}clear(){this.publicKey&&(this.publicKey.fill(0),this.publicKey=void 0)}}function ya(r){const e=new TextEncoder().encode(r),t=be._decode(e),s=mt.parsePrefix(t[0]);if(mt.isValidPublicPrefix(s))return new ga(e);throw new J(H.InvalidPublicKey)}function wa(r){return be.decodeSeed(r),new gi(r)}function xa(r){return btoa(String.fromCharCode(...r))}function _a(r){const e=atob(r),t=new Uint8Array(e.length);for(let s=0;s<e.length;s++)t[s]=e.charCodeAt(s);return t}da(la);const va={createAccount:ma,createOperator:pa,createPair:ks,createUser:ba,fromPublic:ya,fromSeed:wa,NKeysError:J,NKeysErrorCode:H,Prefix:z,decode:_a,encode:xa};function Sa(r){return e=>{let t={};return r.forEach(s=>{const i=s(e)||{};t=Object.assign(t,i)}),t}}function ka(){return()=>{}}function Ea(r,e){return()=>{const t=typeof r=="function"?r():r,s=typeof e=="function"?e():e;return{user:t,pass:s}}}function Ia(r){return()=>({auth_token:typeof r=="function"?r():r})}function Aa(r){return e=>{et.encode(e||"");const s=void 0;return{nkey:"",sig:s?va.encode(s):""}}}function yi(r,e){return t=>{const s=typeof r=="function"?r():r,i=Aa(),{nkey:n,sig:c}=i(t);return{jwt:s,nkey:n,sig:c}}}const wi=120*1e3,Pa=2,xi=2*1e3;function Ra(){return{maxPingOut:2,maxReconnectAttempts:10,noRandomize:!1,pedantic:!1,pingInterval:wi,reconnect:!0,reconnectJitter:100,reconnectJitterTLS:1e3,reconnectTimeWait:xi,tls:void 0,verbose:!1,waitOnFirstConnect:!1,ignoreAuthErrorAbort:!1}}function ja(r){const e=[];return typeof r.authenticator=="function"&&e.push(r.authenticator),Array.isArray(r.authenticator)&&e.push(...r.authenticator),r.token&&e.push(Ia(r.token)),r.user&&e.push(Ea(r.user,r.pass)),e.length===0?ka():Sa(e)}function Oa(r){const e=`${Gs}:${fi()}`;if(r=r||{servers:[e]},r.servers=r.servers||[],typeof r.servers=="string"&&(r.servers=[r.servers]),r.servers.length>0&&r.port)throw new j("port and servers options are mutually exclusive",E.InvalidOption);r.servers.length===0&&r.port&&(r.servers=[`${Gs}:${r.port}`]),r.servers&&r.servers.length===0&&(r.servers=[e]);const t=_s(Ra(),r);if(t.authenticator=ja(t),["reconnectDelayHandler","authenticator"].forEach(s=>{if(t[s]&&typeof t[s]!="function")throw new j(`${s} option should be a function`,E.NotFunction)}),t.reconnectDelayHandler||(t.reconnectDelayHandler=()=>{let s=t.tls?t.reconnectJitterTLS:t.reconnectJitter;return s&&(s++,s=Math.floor(Math.random()*s)),t.reconnectTimeWait+s}),t.inboxPrefix)try{He(t.inboxPrefix)}catch(s){throw new j(s.message,E.ApiError)}if(t.resolve===void 0&&(t.resolve=typeof er()=="function"),t.resolve&&typeof er()!="function")throw new j("'resolve' is not supported on this client",E.InvalidOption);return t}function Ca(r,e){const{proto:t,tls_required:s,tls_available:i}=r;if((t===void 0||t<1)&&e.noEcho)throw new j("noEcho",E.ServerOptionNotAvailable);const n=s||i||!1;if(e.tls&&!n)throw new j("tls",E.ServerOptionNotAvailable)}const Na=1024*32,Ma=/^INFO\s+([^\r\n]+)\r\n/i,Ta=Tt(`PONG\r
`),Vr=Tt(`PING\r
`);class $a{echo;no_responders;protocol;verbose;pedantic;jwt;nkey;sig;user;pass;auth_token;tls_required;name;lang;version;headers;constructor(e,t,s){this.protocol=1,this.version=e.version,this.lang=e.lang,this.echo=t.noEcho?!1:void 0,this.verbose=t.verbose,this.pedantic=t.pedantic,this.tls_required=t.tls?!0:void 0,this.name=t.name;const i=(t&&typeof t.authenticator=="function"?t.authenticator(s):{})||{};_s(this,i)}}class _i extends oe{sid;queue;draining;max;subject;drained;protocol;timer;info;cleanupFn;closed;requestSubject;constructor(e,t,s={}){super(),_s(this,s),this.protocol=e,this.subject=t,this.draining=!1,this.noIterator=typeof s.callback=="function",this.closed=W();const i=!e.options?.noAsyncTraces;s.timeout&&(this.timer=vt(s.timeout,i),this.timer.then(()=>{this.timer=void 0}).catch(n=>{this.stop(n),this.noIterator&&this.callback(n,{})})),this.noIterator||this.iterClosed.then(()=>{this.closed.resolve(),this.unsubscribe()})}setPrePostHandlers(e){if(this.noIterator){const t=this.callback,s=e.ingestionFilterFn?e.ingestionFilterFn:()=>({ingest:!0,protocol:!1}),i=e.protocolFilterFn?e.protocolFilterFn:()=>!0,n=e.dispatchedFn?e.dispatchedFn:()=>{};this.callback=(c,a)=>{const{ingest:d}=s(a);d&&i(a)&&(t(c,a),n(a))}}else this.protocolFilterFn=e.protocolFilterFn,this.dispatchedFn=e.dispatchedFn}callback(e,t){this.cancelTimeout(),e?this.stop(e):this.push(t)}close(){if(!this.isClosed()){this.cancelTimeout();const e=()=>{if(this.stop(),this.cleanupFn)try{this.cleanupFn(this,this.info)}catch{}this.closed.resolve()};this.noIterator?e():this.push(e)}}unsubscribe(e){this.protocol.unsubscribe(this,e)}cancelTimeout(){this.timer&&(this.timer.cancel(),this.timer=void 0)}drain(){return this.protocol.isClosed()?Promise.reject(j.errorForCode(E.ConnectionClosed)):this.isClosed()?Promise.reject(j.errorForCode(E.SubClosed)):(this.drained||(this.draining=!0,this.protocol.unsub(this),this.drained=this.protocol.flush(W()).then(()=>{this.protocol.subscriptions.cancel(this)}).catch(()=>{this.protocol.subscriptions.cancel(this)})),this.drained)}isDraining(){return this.draining}isClosed(){return this.done}getSubject(){return this.subject}getMax(){return this.max}getID(){return this.sid}}class qa{mux;subs;sidCounter;constructor(){this.sidCounter=0,this.mux=null,this.subs=new Map}size(){return this.subs.size}add(e){return this.sidCounter++,e.sid=this.sidCounter,this.subs.set(e.sid,e),e}setMux(e){return this.mux=e,e}getMux(){return this.mux}get(e){return this.subs.get(e)}resub(e){return this.sidCounter++,this.subs.delete(e.sid),e.sid=this.sidCounter,this.subs.set(e.sid,e),e}all(){return Array.from(this.subs.values())}cancel(e){e&&(e.close(),this.subs.delete(e.sid))}handleError(e){if(e&&e.permissionContext){const t=e.permissionContext,s=this.all();let i;if(t.operation==="subscription"&&(i=s.find(n=>n.subject===t.subject&&n.queue===t.queue)),t.operation==="publish"&&(i=s.find(n=>n.requestSubject===t.subject)),i)return i.callback(e,{}),i.close(),this.subs.delete(i.sid),i!==this.mux}return!1}close(){this.subs.forEach(e=>{e.close()})}}class ws{connected;connectedOnce;infoReceived;info;muxSubscriptions;options;outbound;pongs;subscriptions;transport;noMorePublishing;connectError;publisher;_closed;closed;listeners;heartbeats;parser;outMsgs;inMsgs;outBytes;inBytes;pendingLimit;lastError;abortReconnect;whyClosed;servers;server;features;connectPromise;constructor(e,t){this._closed=!1,this.connected=!1,this.connectedOnce=!1,this.infoReceived=!1,this.noMorePublishing=!1,this.abortReconnect=!1,this.listeners=[],this.pendingLimit=Na,this.outMsgs=0,this.inMsgs=0,this.outBytes=0,this.inBytes=0,this.options=e,this.publisher=t,this.subscriptions=new qa,this.muxSubscriptions=new aa,this.outbound=new kt,this.pongs=[],this.whyClosed="",this.pendingLimit=e.pendingLimit||this.pendingLimit,this.features=new Un({major:0,minor:0,micro:0}),this.connectPromise=null;const s=typeof e.servers=="string"?[e.servers]:e.servers;this.servers=new na(s,{randomize:!e.noRandomize}),this.closed=W(),this.parser=new Kr(this),this.heartbeats=new oa(this,this.options.pingInterval||wi,this.options.maxPingOut||Pa)}resetOutbound(){this.outbound.reset();const e=this.pongs;this.pongs=[];const t=j.errorForCode(E.Disconnect);t.stack="",e.forEach(s=>{s.reject(t)}),this.parser=new Kr(this),this.infoReceived=!1}dispatchStatus(e){this.listeners.forEach(t=>{t.push(e)})}status(){const e=new oe;return this.listeners.push(e),e}prepare(){this.transport&&this.transport.discard(),this.info=void 0,this.resetOutbound();const e=W();return e.catch(()=>{}),this.pongs.unshift(e),this.connectError=t=>{e.reject(t)},this.transport=Hn(),this.transport.closed().then(async t=>{if(this.connected=!1,!this.isClosed()){await this.disconnected(this.transport.closeError||this.lastError);return}}),e}disconnect(){this.dispatchStatus({type:gt.StaleConnection,data:""}),this.transport.disconnect()}reconnect(){return this.connected&&(this.dispatchStatus({type:gt.ClientInitiatedReconnect,data:""}),this.transport.disconnect()),Promise.resolve()}async disconnected(e){this.dispatchStatus({type:De.Disconnect,data:this.servers.getCurrentServer().toString()}),this.options.reconnect?await this.dialLoop().then(()=>{this.dispatchStatus({type:De.Reconnect,data:this.servers.getCurrentServer().toString()}),this.lastError?.code===E.AuthenticationExpired&&(this.lastError=void 0)}).catch(t=>{this._close(t)}):await this._close(e)}async dial(e){const t=this.prepare();let s;try{s=vt(this.options.timeout||2e4);const i=this.transport.connect(e,this.options);await Promise.race([i,s]),(async()=>{try{for await(const n of this.transport)this.parser.parse(n)}catch(n){console.log("reader closed",n)}})().then()}catch(i){t.reject(i)}try{await Promise.race([s,t]),s&&s.cancel(),this.connected=!0,this.connectError=void 0,this.sendSubscriptions(),this.connectedOnce=!0,this.server.didConnect=!0,this.server.reconnects=0,this.flushPending(),this.heartbeats.start()}catch(i){throw s&&s.cancel(),await this.transport.close(i),i}}async _doDial(e){const{resolve:t}=this.options,s=await e.resolve({fn:er(),debug:this.options.debug,randomize:!this.options.noRandomize,resolve:t});let i=null;for(const n of s)try{i=null,this.dispatchStatus({type:gt.Reconnecting,data:n.toString()}),await this.dial(n);return}catch(c){i=c}throw i}dialLoop(){return this.connectPromise===null&&(this.connectPromise=this.dodialLoop(),this.connectPromise.then(()=>{}).catch(()=>{}).finally(()=>{this.connectPromise=null})),this.connectPromise}async dodialLoop(){let e;for(;;){this._closed&&this.servers.clear();const t=this.options.reconnectDelayHandler?this.options.reconnectDelayHandler():xi;let s=t;const i=this.selectServer();if(!i||this.abortReconnect)throw e||(this.lastError?this.lastError:j.errorForCode(E.ConnectionRefused));const n=Date.now();if(i.lastConnect===0||i.lastConnect+t<=n){i.lastConnect=Date.now();try{await this._doDial(i);break}catch(c){if(e=c,!this.connectedOnce){if(this.options.waitOnFirstConnect)continue;this.servers.removeCurrentServer()}i.reconnects++;const a=this.options.maxReconnectAttempts||0;a!==-1&&i.reconnects>=a&&this.servers.removeCurrentServer()}}else s=Math.min(s,i.lastConnect+t-n),await Et(s)}}static async connect(e,t){const s=new ws(e,t);return await s.dialLoop(),s}static toError(e){const t=e?e.toLowerCase():"";if(t.indexOf("permissions violation")!==-1){const s=new j(e,E.PermissionsViolation),i=e.match(/(Publish|Subscription) to "(\S+)"/);if(i){s.permissionContext={operation:i[1].toLowerCase(),subject:i[2],queue:void 0};const n=e.match(/using queue "(\S+)"/);n&&(s.permissionContext.queue=n[1])}return s}else return t.indexOf("authorization violation")!==-1?new j(e,E.AuthorizationViolation):t.indexOf("user authentication expired")!==-1?new j(e,E.AuthenticationExpired):t.indexOf("account authentication expired")!=-1?new j(e,E.AccountExpired):t.indexOf("authentication timeout")!==-1?new j(e,E.AuthenticationTimeout):new j(e,E.ProtocolError)}processMsg(e,t){if(this.inMsgs++,this.inBytes+=t.length,!this.subscriptions.sidCounter)return;const s=this.subscriptions.get(e.sid);s&&(s.received+=1,s.callback&&s.callback(null,new ui(e,t,this)),s.max!==void 0&&s.received>=s.max&&s.unsubscribe())}processError(e){const t=jr(e),s=ws.toError(t),i={type:De.Error,data:s.code};if(s.isPermissionError()){let n=!1;s.permissionContext&&(i.permissionContext=s.permissionContext,n=this.subscriptions.getMux()?.subject===s.permissionContext.subject),this.subscriptions.handleError(s),this.muxSubscriptions.handleError(n,s),n&&this.subscriptions.setMux(null)}this.dispatchStatus(i),this.handleError(s)}handleError(e){e.isAuthError()?this.handleAuthError(e):e.isProtocolError()?this.lastError=e:e.isAuthTimeout()&&(this.lastError=e),e.isPermissionError()||(this.lastError=e)}handleAuthError(e){this.lastError&&e.code===this.lastError.code&&this.options.ignoreAuthErrorAbort===!1&&(this.abortReconnect=!0),this.connectError?this.connectError(e):this.disconnect()}processPing(){this.transport.send(Ta)}processPong(){const e=this.pongs.shift();e&&e.resolve()}processInfo(e){const t=JSON.parse(jr(e));this.info=t;const s=this.options&&this.options.ignoreClusterUpdates?void 0:this.servers.update(t,this.transport.isEncrypted());if(!this.infoReceived){this.features.update(rt(t.version)),this.infoReceived=!0,this.transport.isEncrypted()&&this.servers.updateTLSName();const{version:n,lang:c}=this.transport;try{const a=new $a({version:n,lang:c},this.options,t.nonce);t.headers&&(a.headers=!0,a.no_responders=!0);const d=JSON.stringify(a);this.transport.send(Tt(`CONNECT ${d}${us}`)),this.transport.send(Vr)}catch(a){this._close(a)}}s&&this.dispatchStatus({type:De.Update,data:s}),t.ldm!==void 0&&t.ldm&&this.dispatchStatus({type:De.LDM,data:this.servers.getCurrentServer().toString()})}push(e){switch(e.kind){case me.MSG:{const{msg:t,data:s}=e;this.processMsg(t,s);break}case me.OK:break;case me.ERR:this.processError(e.data);break;case me.PING:this.processPing();break;case me.PONG:this.processPong();break;case me.INFO:this.processInfo(e.data);break}}sendCommand(e,...t){const s=this.outbound.length();let i;typeof e=="string"?i=Tt(e):i=e,this.outbound.fill(i,...t),s===0?queueMicrotask(()=>{this.flushPending()}):this.outbound.size()>=this.pendingLimit&&this.flushPending()}publish(e,t=Ie,s){let i;if(t instanceof Uint8Array)i=t;else if(typeof t=="string")i=et.encode(t);else throw j.errorForCode(E.BadPayload);let n=i.length;s=s||{},s.reply=s.reply||"";let c=Ie,a=0;if(s.headers){if(this.info&&!this.info.headers)throw new j("headers",E.ServerOptionNotAvailable);c=s.headers.encode(),a=c.length,n=i.length+a}if(this.info&&n>this.info.max_payload)throw j.errorForCode(E.MaxPayloadExceeded);this.outBytes+=n,this.outMsgs++;let d;s.headers?(s.reply?d=`HPUB ${e} ${s.reply} ${a} ${n}\r
`:d=`HPUB ${e} ${a} ${n}\r
`,this.sendCommand(d,c,i,gs)):(s.reply?d=`PUB ${e} ${s.reply} ${n}\r
`:d=`PUB ${e} ${n}\r
`,this.sendCommand(d,i,gs))}request(e){return this.initMux(),this.muxSubscriptions.add(e),e}subscribe(e){return this.subscriptions.add(e),this._subunsub(e),e}_sub(e){e.queue?this.sendCommand(`SUB ${e.subject} ${e.queue} ${e.sid}\r
`):this.sendCommand(`SUB ${e.subject} ${e.sid}\r
`)}_subunsub(e){return this._sub(e),e.max&&this.unsubscribe(e,e.max),e}unsubscribe(e,t){this.unsub(e,t),(e.max===void 0||e.received>=e.max)&&this.subscriptions.cancel(e)}unsub(e,t){!e||this.isClosed()||(t?this.sendCommand(`UNSUB ${e.sid} ${t}\r
`):this.sendCommand(`UNSUB ${e.sid}\r
`),e.max=t)}resub(e,t){!e||this.isClosed()||(this.unsub(e),e.subject=t,this.subscriptions.resub(e),this._sub(e))}flush(e){return e||(e=W()),this.pongs.push(e),this.outbound.fill(Vr),this.flushPending(),e}sendSubscriptions(){const e=[];this.subscriptions.all().forEach(t=>{const s=t;s.queue?e.push(`SUB ${s.subject} ${s.queue} ${s.sid}${us}`):e.push(`SUB ${s.subject} ${s.sid}${us}`)}),e.length&&this.transport.send(Tt(e.join("")))}async _close(e){this._closed||(this.whyClosed=new Error("close trace").stack||"",this.heartbeats.cancel(),this.connectError&&(this.connectError(e),this.connectError=void 0),this.muxSubscriptions.close(),this.subscriptions.close(),this.listeners.forEach(t=>{t.stop()}),this._closed=!0,await this.transport.close(e),await this.closed.resolve(e))}close(){return this._close()}isClosed(){return this._closed}drain(){const e=this.subscriptions.all(),t=[];return e.forEach(s=>{t.push(s.drain())}),Promise.all(t).then(async()=>(this.noMorePublishing=!0,await this.flush(),this.close())).catch(()=>{})}flushPending(){if(!(!this.infoReceived||!this.connected)&&this.outbound.size()){const e=this.outbound.drain();this.transport.send(e)}}initMux(){if(!this.subscriptions.getMux()){const t=this.muxSubscriptions.init(this.options.inboxPrefix),s=new _i(this,`${t}*`);s.callback=this.muxSubscriptions.dispatcher(),this.subscriptions.setMux(s),this.subscribe(s)}}selectServer(){const e=this.servers.selectServer();if(e!==void 0)return this.server=e,this.server}getServer(){return this.server}}const Fa="$SRV";class Wr{msg;constructor(e){this.msg=e}get data(){return this.msg.data}get sid(){return this.msg.sid}get subject(){return this.msg.subject}get reply(){return this.msg.reply||""}get headers(){return this.msg.headers}respond(e,t){return this.msg.respond(e,t)}respondError(e,t,s,i){return i=i||{},i.headers=i.headers||ze(),i.headers?.set(fs,`${e}`),i.headers?.set(ds,t),this.msg.respond(s,i)}json(e){return this.msg.json(e)}string(){return this.msg.string()}}class Ht{subject;queue;srv;constructor(e,t="",s=""){t!==""&&Ua("service group",t);let i="";if(e instanceof Vt)this.srv=e,i="";else if(e instanceof Ht){const n=e;this.srv=n.srv,s===""&&n.queue!==""&&(s=n.queue),i=n.subject}else throw new Error("unknown ServiceGroup type");this.subject=this.calcSubject(i,t),this.queue=s}calcSubject(e,t=""){return t===""?e:e!==""?`${e}.${t}`:t}addEndpoint(e="",t){t=t||{subject:e};const s=typeof t=="function"?{handler:t,subject:e}:t;Nt("endpoint",e);let{subject:i,handler:n,metadata:c,queue:a}=s;i=i||e,a=a||this.queue,La("endpoint subject",i),i=this.calcSubject(this.subject,i);const d={name:e,subject:i,queue:a,handler:n,metadata:c};return this.srv._addEndpoint(d)}addGroup(e="",t=""){return new Ht(this,e,t)}}function La(r,e){if(e==="")throw new Error(`${r} cannot be empty`);if(e.indexOf(" ")!==-1)throw new Error(`${r} cannot contain spaces: '${e}'`);const t=e.split(".");t.forEach((s,i)=>{if(s===">"&&i!==t.length-1)throw new Error(`${r} cannot have internal '>': '${e}'`)})}function Ua(r,e){if(e.indexOf(" ")!==-1)throw new Error(`${r} cannot contain spaces: '${e}'`);e.split(".").forEach(s=>{if(s===">")throw new Error(`${r} name cannot contain internal '>': '${e}'`)})}class Vt{nc;_id;config;handlers;internal;_stopped;_done;started;static controlSubject(e,t="",s="",i){const n=i??Fa;return t===""&&s===""?`${n}.${e}`:(Nt("control subject name",t),s!==""?(Nt("control subject id",s),`${n}.${e}.${t}.${s}`):`${n}.${e}.${t}`)}constructor(e,t={name:"",version:""}){this.nc=e,this.config=Object.assign({},t),this.config.queue||(this.config.queue="q"),Nt("name",this.config.name),Nt("queue",this.config.queue),rt(this.config.version),this._id=Ze.next(),this.internal=[],this._done=W(),this._stopped=!1,this.handlers=[],this.started=new Date().toISOString(),this.reset(),this.nc.closed().then(()=>{this.close().catch()}).catch(s=>{this.close(s).catch()})}get subjects(){return this.handlers.filter(e=>e.internal===!1).map(e=>e.subject)}get id(){return this._id}get name(){return this.config.name}get description(){return this.config.description??""}get version(){return this.config.version}get metadata(){return this.config.metadata}errorToHeader(e){const t=ze();if(e instanceof ps){const s=e;t.set(ds,s.message),t.set(fs,`${s.code}`)}else t.set(ds,e.message),t.set(fs,"500");return t}setupHandler(e,t=!1){const s=t?"":e.queue?e.queue:this.config.queue,{name:i,subject:n,handler:c}=e,a=e;a.internal=t,t&&this.internal.push(a),a.stats=new Ba(i,n,s),a.queue=s;const d=c?(m,x)=>{if(m){this.close(m);return}const v=Date.now();try{c(m,new Wr(x))}catch(S){a.stats.countError(S),x?.respond(Ie,{headers:this.errorToHeader(S)})}finally{a.stats.countLatency(v)}}:void 0;return a.sub=this.nc.subscribe(n,{callback:d,queue:s}),a.sub.closed.then(()=>{this._stopped||this.close(new Error(`required subscription ${e.subject} stopped`)).catch()}).catch(m=>{if(!this._stopped){const x=new Error(`required subscription ${e.subject} errored: ${m.message}`);x.stack=m.stack,this.close(x).catch()}}),a}info(){return{type:$t.INFO,name:this.name,id:this.id,version:this.version,description:this.description,metadata:this.metadata,endpoints:this.endpoints()}}endpoints(){return this.handlers.map(e=>{const{subject:t,metadata:s,name:i,queue:n}=e;return{subject:t,metadata:s,name:i,queue_group:n}})}async stats(){const e=[];for(const t of this.handlers){if(typeof this.config.statsHandler=="function")try{t.stats.data=await this.config.statsHandler(t)}catch(s){t.stats.countError(s)}e.push(t.stats.stats(t.qi))}return{type:$t.STATS,name:this.name,id:this.id,version:this.version,started:this.started,metadata:this.metadata,endpoints:e}}addInternalHandler(e,t){const s=`${e}`.toUpperCase();this._doAddInternalHandler(`${s}-all`,e,t),this._doAddInternalHandler(`${s}-kind`,e,t,this.name),this._doAddInternalHandler(`${s}`,e,t,this.name,this.id)}_doAddInternalHandler(e,t,s,i="",n=""){const c={};c.name=e,c.subject=Vt.controlSubject(t,i,n),c.handler=s,this.setupHandler(c,!0)}start(){const e=$e(),t=(c,a)=>c?(this.close(c),Promise.reject(c)):this.stats().then(d=>(a?.respond(e.encode(d)),Promise.resolve())),s=(c,a)=>c?(this.close(c),Promise.reject(c)):(a?.respond(e.encode(this.info())),Promise.resolve()),i=e.encode(this.ping()),n=(c,a)=>c?(this.close(c).then().catch(),Promise.reject(c)):(a.respond(i),Promise.resolve());return this.addInternalHandler(Ve.PING,n),this.addInternalHandler(Ve.STATS,t),this.addInternalHandler(Ve.INFO,s),this.handlers.forEach(c=>{const{subject:a}=c;typeof a=="string"&&c.handler!==null&&this.setupHandler(c)}),Promise.resolve(this)}close(e){if(this._stopped)return this._done;this._stopped=!0;let t=[];return this.nc.isClosed()||(t=this.handlers.concat(this.internal).map(s=>s.sub.drain())),Promise.allSettled(t).then(()=>{this._done.resolve(e||null)}),this._done}get stopped(){return this._done}get isStopped(){return this._stopped}stop(e){return this.close(e)}ping(){return{type:$t.PING,name:this.name,id:this.id,version:this.version,metadata:this.metadata}}reset(){if(this.started=new Date().toISOString(),this.handlers)for(const e of this.handlers)e.stats.reset(e.qi)}addGroup(e,t){return new Ht(this,e,t)}addEndpoint(e,t){return new Ht(this).addEndpoint(e,t)}_addEndpoint(e){const t=new oe;t.noIterator=typeof e.handler=="function",t.noIterator||(e.handler=(i,n)=>{i?this.stop(i).catch():t.push(new Wr(n))},t.iterClosed.then(()=>{this.close().catch()}));const s=this.setupHandler(e,!1);return s.qi=t,this.handlers.push(s),t}}class Ba{name;subject;average_processing_time;num_requests;processing_time;num_errors;last_error;data;metadata;queue;constructor(e,t,s=""){this.name=e,this.subject=t,this.average_processing_time=0,this.num_errors=0,this.num_requests=0,this.processing_time=0,this.queue=s}reset(e){this.num_requests=0,this.processing_time=0,this.average_processing_time=0,this.num_errors=0,this.last_error=void 0,this.data=void 0;const t=e;t&&(t.time=0,t.processed=0)}countLatency(e){this.num_requests++,this.processing_time+=V(Date.now()-e),this.average_processing_time=Math.round(this.processing_time/this.num_requests)}countError(e){this.num_errors++,this.last_error=e.message}_stats(){const{name:e,subject:t,average_processing_time:s,num_errors:i,num_requests:n,processing_time:c,last_error:a,data:d,queue:m}=this;return{name:e,subject:t,average_processing_time:s,num_errors:i,num_requests:n,processing_time:c,last_error:a,data:d,queue_group:m}}stats(e){const t=e;return t?.noIterator===!1&&(this.processing_time=V(t.time),this.num_requests=t.processed,this.average_processing_time=this.processing_time>0&&this.num_requests>0?this.processing_time/this.num_requests:0),this._stats()}}class Da{nc;prefix;opts;constructor(e,t={strategy:Te.JitterTimer,maxWait:2e3},s){this.nc=e,this.prefix=s,this.opts=t}ping(e="",t=""){return this.q(Ve.PING,e,t)}stats(e="",t=""){return this.q(Ve.STATS,e,t)}info(e="",t=""){return this.q(Ve.INFO,e,t)}async q(e,t="",s=""){const i=new oe,n=$e(),c=Vt.controlSubject(e,t,s,this.prefix),a=await this.nc.requestMany(c,Ie,this.opts);return(async()=>{for await(const d of a)try{const m=n.decode(d.data);i.push(m)}catch(m){i.push(()=>{i.stop(m)})}i.push(()=>{i.stop()})})().catch(d=>{i.stop(d)}),i}}function vi(){return{key:{encode(r){return r},decode(r){return r}},value:{encode(r){return r},decode(r){return r}}}}function Ha(){return{replicas:1,history:1,timeout:2e3,max_bytes:-1,maxValueSize:-1,codec:vi(),storage:Xs.File}}const xs="KV-Operation",Yr="$KV",za=/^[-/=.\w]+$/,Ja=/^[-/=.>*\w]+$/,Ka=/^[-\w]+$/;function Ga(r){if(r.startsWith(".")||r.endsWith(".")||!za.test(r))throw new Error(`invalid key: ${r}`)}function Va(r){if(r.startsWith(".")||r.endsWith(".")||!Ja.test(r))throw new Error(`invalid key: ${r}`)}function Wa(r){if(r.startsWith(".")||r.endsWith("."))throw new Error(`invalid key: ${r}`);const e=r.split(".");let t=!1;for(let s=0;s<e.length;s++)switch(e[s]){case"*":t=!0;break;case">":if(s!==e.length-1)throw new Error(`invalid key: ${r}`);t=!0;break}return t}function hs(r){if(!Ka.test(r))throw new Error(`invalid bucket name: ${r}`)}var Fe;(function(r){r.MsgIdHdr="Nats-Msg-Id",r.ExpectedStreamHdr="Nats-Expected-Stream",r.ExpectedLastSeqHdr="Nats-Expected-Last-Sequence",r.ExpectedLastMsgIdHdr="Nats-Expected-Last-Msg-Id",r.ExpectedLastSubjectSequenceHdr="Nats-Expected-Last-Subject-Sequence"})(Fe||(Fe={}));class zt{js;jsm;stream;bucket;direct;codec;prefix;editPrefix;useJsPrefix;_prefixLen;constructor(e,t,s){hs(e),this.js=t,this.jsm=s,this.bucket=e,this.prefix=Yr,this.editPrefix="",this.useJsPrefix=!1,this._prefixLen=0}static async create(e,t,s={}){hs(t);const i=await e.jetstreamManager(),n=new zt(t,e,i);return await n.init(s),n}static async bind(e,t,s={}){const i=await e.jetstreamManager(),n={config:{allow_direct:s.allow_direct}};hs(t);const c=new zt(t,e,i);return n.config.name=s.streamName??c.bucketName(),Object.assign(c,n),c.stream=n.config.name,c.codec=s.codec||vi(),c.direct=n.config.allow_direct??!1,c.initializePrefixes(n),c}async init(e={}){const t=Object.assign(Ha(),e);this.codec=t.codec;const s={};this.stream=s.name=e.streamName??this.bucketName(),s.retention=Ys.Limits,s.max_msgs_per_subject=t.history,t.maxBucketSize&&(t.max_bytes=t.maxBucketSize),t.max_bytes&&(s.max_bytes=t.max_bytes),s.max_msg_size=t.maxValueSize,s.storage=t.storage;const i=e.placementCluster??"";if(i&&(e.placement={},e.placement.cluster=i,e.placement.tags=[]),e.placement&&(s.placement=e.placement),e.republish&&(s.republish=e.republish),e.description&&(s.description=e.description),e.mirror){const v=Object.assign({},e.mirror);v.name.startsWith(_e)||(v.name=`${_e}${v.name}`),s.mirror=v,s.mirror_direct=!0}else if(e.sources){const v=e.sources.map(S=>{const O=Object.assign({},S),$=O.name.startsWith(_e)?O.name.substring(_e.length):O.name;return O.name.startsWith(_e)||(O.name=`${_e}${O.name}`),!S.external&&$!==this.bucket&&(O.subject_transforms=[{src:`$KV.${$}.>`,dest:`$KV.${this.bucket}.>`}]),O});s.sources=v,s.subjects=[this.subjectForBucket()]}else s.subjects=[this.subjectForBucket()];e.metadata&&(s.metadata=e.metadata),typeof e.compression=="boolean"&&(s.compression=e.compression?Qe.S2:Qe.None);const n=this.js.nc,c=n.getServerVersion(),a=c?Qs(c,rt("2.7.2"))>=0:!1;s.discard=a?Dt.New:Dt.Old;const{ok:d,min:m}=n.features.get(U.JS_ALLOW_DIRECT);if(!d&&e.allow_direct===!0){const v=c?`${c.major}.${c.minor}.${c.micro}`:"unknown";return Promise.reject(new Error(`allow_direct is not available on server version ${v} - requires ${m}`))}e.allow_direct=typeof e.allow_direct=="boolean"?e.allow_direct:d,s.allow_direct=e.allow_direct,this.direct=s.allow_direct,s.num_replicas=t.replicas,t.ttl&&(s.max_age=V(t.ttl)),s.allow_rollup_hdrs=!0;let x;try{x=await this.jsm.streams.info(s.name),!x.config.allow_direct&&this.direct===!0&&(this.direct=!1)}catch(v){if(v.message==="stream not found")x=await this.jsm.streams.add(s);else throw v}this.initializePrefixes(x)}initializePrefixes(e){this._prefixLen=0,this.prefix=`$KV.${this.bucket}`,this.useJsPrefix=this.js.apiPrefix!=="$JS.API";const{mirror:t}=e.config;if(t){let s=t.name;if(s.startsWith(_e)&&(s=s.substring(_e.length)),t.external&&t.external.api!==""){const i=t.name.substring(_e.length);this.useJsPrefix=!1,this.prefix=`$KV.${i}`,this.editPrefix=`${t.external.api}.$KV.${s}`}else this.editPrefix=this.prefix}}bucketName(){return this.stream??`${_e}${this.bucket}`}subjectForBucket(){return`${this.prefix}.${this.bucket}.>`}subjectForKey(e,t=!1){const s=[];return t?(this.useJsPrefix&&s.push(this.js.apiPrefix),this.editPrefix!==""?s.push(this.editPrefix):s.push(this.prefix)):this.prefix&&s.push(this.prefix),s.push(e),s.join(".")}fullKeyName(e){return this.prefix!==""?`${this.prefix}.${e}`:`${Yr}.${this.bucket}.${e}`}get prefixLen(){return this._prefixLen===0&&(this._prefixLen=this.prefix.length+1),this._prefixLen}encodeKey(e){const t=[];for(const s of e.split("."))switch(s){case">":case"*":t.push(s);break;default:t.push(this.codec.key.encode(s));break}return t.join(".")}decodeKey(e){const t=[];for(const s of e.split("."))switch(s){case">":case"*":t.push(s);break;default:t.push(this.codec.key.decode(s));break}return t.join(".")}validateKey=Ga;validateSearchKey=Va;hasWildcards=Wa;close(){return Promise.resolve()}dataLen(e,t){const s=t&&t.get(ge.MessageSizeHdr)||"";return s!==""?parseInt(s,10):e.length}smToEntry(e){return new lo(this.bucket,this.prefixLen,e)}jmToEntry(e){const t=this.decodeKey(e.subject.substring(this.prefixLen));return new fo(this.bucket,t,e)}async create(e,t){let s;try{const n=await this.put(e,t,{previousSeq:0});return Promise.resolve(n)}catch(n){if(s=n,n?.api_error?.err_code!==10071)return Promise.reject(n)}let i=0;try{const n=await this.get(e);return n?.operation==="DEL"||n?.operation==="PURGE"?(i=n!==null?n.revision:0,this.update(e,t,i)):Promise.reject(s)}catch(n){return Promise.reject(n)}}update(e,t,s){if(s<=0)throw new Error("version must be greater than 0");return this.put(e,t,{previousSeq:s})}async put(e,t,s={}){const i=this.encodeKey(e);this.validateKey(i);const n={};if(s.previousSeq!==void 0){const c=ze();n.headers=c,c.set(Fe.ExpectedLastSubjectSequenceHdr,`${s.previousSeq}`)}try{return(await this.js.publish(this.subjectForKey(i,!0),t,n)).seq}catch(c){const a=c;return a.isJetStreamError()?(a.message=a.api_error?.description,a.code=`${a.api_error?.code}`,Promise.reject(a)):Promise.reject(c)}}async get(e,t){const s=this.encodeKey(e);this.validateKey(s);let i={last_by_subj:this.subjectForKey(s)};t&&t.revision>0&&(i={seq:t.revision});let n;try{this.direct?n=await this.jsm.direct.getMessage(this.bucketName(),i):n=await this.jsm.streams.getMessage(this.bucketName(),i);const c=this.smToEntry(n);return c.key!==s?null:c}catch(c){if(c.code===E.JetStream404NoMessages)return null;throw c}}purge(e,t){return this._deleteOrPurge(e,"PURGE",t)}delete(e,t){return this._deleteOrPurge(e,"DEL",t)}async purgeDeletes(e=1800*1e3){const t=W(),s=[],i=await this.watch({key:">",initializedFn:()=>{t.resolve()}});(async()=>{for await(const d of i)(d.operation==="DEL"||d.operation==="PURGE")&&s.push(d)})().then(),await t,i.stop();const n=Date.now()-e,c=s.map(d=>{const m=this.subjectForKey(d.key);return d.created.getTime()>=n?this.jsm.streams.purge(this.stream,{filter:m,keep:1}):this.jsm.streams.purge(this.stream,{filter:m,keep:0})}),a=await Promise.all(c);return a.unshift({success:!0,purged:0}),a.reduce((d,m)=>(d.purged+=m.purged,d))}async _deleteOrPurge(e,t,s){if(!this.hasWildcards(e))return this._doDeleteOrPurge(e,t,s);const i=await this.keys(e),n=[];for await(const c of i)n.push(this._doDeleteOrPurge(c,t)),n.length===100&&(await Promise.all(n),n.length=0);n.length>0&&await Promise.all(n)}async _doDeleteOrPurge(e,t,s){const i=this.encodeKey(e);this.validateKey(i);const n=ze();n.set(xs,t),t==="PURGE"&&n.set(ge.RollupHdr,ge.RollupValueSubject),s?.previousSeq&&n.set(Fe.ExpectedLastSubjectSequenceHdr,`${s.previousSeq}`),await this.js.publish(this.subjectForKey(i,!0),Ie,{headers:n})}_buildCC(e,t,s={}){let n=(Array.isArray(e)?e:[e]).map(d=>{const m=this.encodeKey(d);return this.validateSearchKey(d),this.fullKeyName(m)}),c=Q.LastPerSubject;t===Ne.AllHistory&&(c=Q.All),t===Ne.UpdatesOnly&&(c=Q.New);let a;return n.length===1&&(a=n[0],n=void 0),Object.assign({deliver_policy:c,ack_policy:ae.None,filter_subjects:n,filter_subject:a,flow_control:!0,idle_heartbeat:V(5*1e3)},s)}remove(e){return this.purge(e)}async history(e={}){const t=e.key??">",s=new oe,i={};i.headers_only=e.headers_only||!1;let n;n=()=>{s.stop()};let c=0;const a=this._buildCC(t,Ne.AllHistory,i),d=a.filter_subject,m=Ye(a);m.bindStream(this.stream),m.orderedConsumer(),m.callback((v,S)=>{if(v){s.stop(v);return}if(S){const O=this.jmToEntry(S);s.push(O),s.received++,(n&&c>0&&s.received>=c||S.info.pending===0)&&(s.push(n),n=void 0)}});const x=await this.js.subscribe(d,m);if(n){const{info:{last:v}}=x,S=v.num_pending+v.delivered.consumer_seq;if(S===0||s.received>=S)try{n()}catch(O){s.stop(O)}finally{n=void 0}else c=S}return s._data=x,s.iterClosed.then(()=>{x.unsubscribe()}),x.closed.then(()=>{s.stop()}).catch(v=>{s.stop(v)}),s}canSetWatcherName(){const t=this.js.nc,{ok:s}=t.features.get(U.JS_NEW_CONSUMER_CREATE_API);return s}async watch(e={}){const t=e.key??">",s=new oe,i={};i.headers_only=e.headers_only||!1;let n=Ne.LastValue;e.include===Ne.AllHistory?n=Ne.AllHistory:e.include===Ne.UpdatesOnly&&(n=Ne.UpdatesOnly);const c=e.ignoreDeletes===!0;let a=e.initializedFn,d=0;const m=this._buildCC(t,n,i),x=m.filter_subject,v=Ye(m);this.canSetWatcherName()&&v.consumerName(Ze.next()),v.bindStream(this.stream),e.resumeFromRevision&&e.resumeFromRevision>0&&v.startSequence(e.resumeFromRevision),v.orderedConsumer(),v.callback((O,$)=>{if(O){s.stop(O);return}if($){const K=this.jmToEntry($);if(c&&K.operation==="DEL")return;s.push(K),s.received++,a&&(d>0&&s.received>=d||$.info.pending===0)&&(s.push(a),a=void 0)}});const S=await this.js.subscribe(x,v);if(a){const{info:{last:O}}=S,$=O.num_pending+O.delivered.consumer_seq;if($===0||s.received>=$)try{a()}catch(K){s.stop(K)}finally{a=void 0}else d=$}return s._data=S,s.iterClosed.then(()=>{S.unsubscribe()}),S.closed.then(()=>{s.stop()}).catch(O=>{s.stop(O)}),s}async keys(e=">"){const t=new oe,s=this._buildCC(e,Ne.LastValue,{headers_only:!0}),i=Array.isArray(e)?">":s.filter_subject,n=Ye(s);n.bindStream(this.stream),n.orderedConsumer();const c=await this.js.subscribe(i,n);return(async()=>{for await(const d of c){const m=d.headers?.get(xs);if(m!=="DEL"&&m!=="PURGE"){const x=this.decodeKey(d.subject.substring(this.prefixLen));t.push(x)}d.info.pending===0&&c.unsubscribe()}})().then(()=>{t.stop()}).catch(d=>{t.stop(d)}),c.info.last.num_pending===0&&c.unsubscribe(),t}purgeBucket(e){return this.jsm.streams.purge(this.bucketName(),e)}destroy(){return this.jsm.streams.delete(this.bucketName())}async status(){const t=this.js.nc.info?.cluster??"",s=this.bucketName(),i=await this.jsm.streams.info(s);return new Si(i,t)}}class Si{si;cluster;constructor(e,t=""){this.si=e,this.cluster=t}get bucket(){return this.si.config.name.startsWith(_e)?this.si.config.name.substring(_e.length):this.si.config.name}get values(){return this.si.state.messages}get history(){return this.si.config.max_msgs_per_subject}get ttl(){return ur(this.si.config.max_age)}get bucket_location(){return this.cluster}get backingStore(){return this.si.config.storage}get storage(){return this.si.config.storage}get replicas(){return this.si.config.num_replicas}get description(){return this.si.config.description??""}get maxBucketSize(){return this.si.config.max_bytes}get maxValueSize(){return this.si.config.max_msg_size}get max_bytes(){return this.si.config.max_bytes}get placement(){return this.si.config.placement||{cluster:"",tags:[]}}get placementCluster(){return this.si.config.placement?.cluster??""}get republish(){return this.si.config.republish??{src:"",dest:""}}get streamInfo(){return this.si}get size(){return this.si.state.bytes}get metadata(){return this.si.config.metadata??{}}get compression(){return this.si.config.compression?this.si.config.compression!==Qe.None:!1}}const lr="OBJ_",Xr="SHA-256=";function Ya(r){return hs(r),`${lr}${r}`}function Xa(r){return r.startsWith(lr)?r.substring(4):r}class rr{si;backingStore;constructor(e){this.si=e,this.backingStore="JetStream"}get bucket(){return Xa(this.si.config.name)}get description(){return this.si.config.description??""}get ttl(){return this.si.config.max_age}get storage(){return this.si.config.storage}get replicas(){return this.si.config.num_replicas}get sealed(){return this.si.config.sealed}get size(){return this.si.state.bytes}get streamInfo(){return this.si}get metadata(){return this.si.config.metadata}get compression(){return this.si.config.compression?this.si.config.compression!==Qe.None:!1}}function os(r){if(r===void 0)return;const{domain:e}=r;if(e===void 0)return r;const t=Object.assign({},r);if(delete t.domain,e==="")return t;if(t.external)throw new Error("domain and external are both set");return t.external={api:`$JS.${e}.API`},t}var Pe;(function(r){r[r.Unset=-1]="Unset",r[r.Consume=0]="Consume",r[r.Fetch=1]="Fetch"})(Pe||(Pe={}));var Le;(function(r){r.HeartbeatsMissed="heartbeats_missed",r.ConsumerNotFound="consumer_not_found",r.StreamNotFound="stream_not_found",r.ConsumerDeleted="consumer_deleted",r.OrderedConsumerRecreated="ordered_consumer_recreated",r.NoResponders="no_responders"})(Le||(Le={}));var _t;(function(r){r.DebugEvent="debug",r.Discard="discard",r.Reset="reset",r.Next="next"})(_t||(_t={}));const Zr=Uint8Array.of(43,65,67,75),Za=Uint8Array.of(45,78,65,75),Ct=Uint8Array.of(43,87,80,73),Qa=Uint8Array.of(43,78,88,84),eo=Uint8Array.of(43,84,69,82,77),to=Uint8Array.of(32);function Jt(r,e=5e3){return new xo(r,e)}class zs extends oe{consumer;opts;sub;monitor;pending;inbox;refilling;pong;callback;timeout;cleanupHandler;listeners;statusIterator;forOrderedConsumer;resetHandler;abortOnMissingResource;bind;inBackOff;constructor(e,t,s=!1){super(),this.consumer=e;const i=t;this.opts=this.parseOptions(t,s),this.callback=i.callback||null,this.noIterator=typeof this.callback=="function",this.monitor=null,this.pong=null,this.pending={msgs:0,bytes:0,requests:0},this.refilling=s,this.timeout=null,this.inbox=He(e.api.nc.options.inboxPrefix),this.listeners=[],this.forOrderedConsumer=!1,this.abortOnMissingResource=i.abort_on_missing_resource===!0,this.bind=i.bind===!0,this.inBackOff=!1,this.start()}start(){const{max_messages:e,max_bytes:t,idle_heartbeat:s,threshold_bytes:i,threshold_messages:n}=this.opts;this.closed().then(a=>{if(this.cleanupHandler)try{this.cleanupHandler(a)}catch{}});const{sub:c}=this;c&&c.unsubscribe(),this.sub=this.consumer.api.nc.subscribe(this.inbox,{callback:(a,d)=>{if(a){this.stop(a);return}if(this.monitor?.work(),d.subject===this.inbox){if(Ws(d))return;const x=d.headers?.code,v=d.headers?.description?.toLowerCase()||"unknown",{msgsLeft:S,bytesLeft:O}=this.parseDiscard(d.headers);if(S>0||O>0)this.pending.msgs-=S,this.pending.bytes-=O,this.pending.requests--,this.notify(_t.Discard,{msgsLeft:S,bytesLeft:O});else if(x===400){this.stop(new j(v,`${x}`));return}else if(x===409&&v==="consumer deleted"){if(this.notify(Le.ConsumerDeleted,`${x} ${v}`),!this.refilling||this.abortOnMissingResource){const $=new j(v,`${x}`);this.stop($);return}}else if(x===503){if(this.notify(Le.NoResponders,`${x} No Responders`),!this.refilling||this.abortOnMissingResource){const $=new j("no responders",`${x}`);this.stop($);return}}else this.notify(_t.DebugEvent,`${x} ${v}`)}else this._push(Jt(d,this.consumer.api.timeout)),this.received++,this.pending.msgs&&this.pending.msgs--,this.pending.bytes&&(this.pending.bytes-=d.size());if(this.pending.msgs===0&&this.pending.bytes===0&&(this.pending.requests=0),this.refilling){if(e&&this.pending.msgs<=n||t&&this.pending.bytes<=i){const x=this.pullOptions();this.pull(x)}}else this.pending.requests===0&&this._push(()=>{this.stop()})}}),this.sub.closed.then(()=>{this.sub.draining&&this._push(()=>{this.stop()})}),s&&(this.monitor=new hr(s,a=>(this.notify(Le.HeartbeatsMissed,a),this.resetPending().then(()=>{}).catch(()=>{}),!1),{maxOut:2})),(async()=>{const a=this.consumer.api.nc.status();this.statusIterator=a;for await(const d of a)switch(d.type){case De.Disconnect:this.monitor?.cancel();break;case De.Reconnect:this.resetPending().then(m=>{m&&this.monitor?.restart()}).catch(()=>{});break}})(),this.pull(this.pullOptions())}_push(e){if(!this.callback)super.push(e);else{const t=typeof e=="function"?e:null;try{t?t():this.callback(e)}catch(s){this.stop(s)}}}notify(e,t){this.listeners.length>0&&this.listeners.forEach(s=>{s.done||s.push({type:e,data:t})})}resetPending(){return this.bind?this.resetPendingNoInfo():this.resetPendingWithInfo()}resetPendingNoInfo(){return this.pending.msgs=0,this.pending.bytes=0,this.pending.requests=0,this.pull(this.pullOptions()),Promise.resolve(!0)}async resetPendingWithInfo(){if(this.inBackOff)return!1;let e=0,t=0;const s=cr([this.opts.expires]);let i=0;for(;;){if(this.done)return!1;if(this.consumer.api.nc.isClosed())return console.error("aborting resetPending - connection is closed"),!1;try{return await this.consumer.info(),this.inBackOff=!1,e=0,this.pending.msgs=0,this.pending.bytes=0,this.pending.requests=0,this.pull(this.pullOptions()),!0}catch(n){if(n.message==="stream not found"){if(t++,this.notify(Le.StreamNotFound,t),!this.refilling||this.abortOnMissingResource)return this.stop(n),!1}else if(n.message==="consumer not found"){if(e++,this.notify(Le.ConsumerNotFound,e),this.resetHandler)try{this.resetHandler()}catch{}if(!this.refilling||this.abortOnMissingResource)return this.stop(n),!1;if(this.forOrderedConsumer)return!1}else e=0,t=0;this.inBackOff=!0;const c=s.backoff(i),a=Et(c);await Promise.race([a,this.consumer.api.nc.closed()]),a.cancel(),i++}}}pull(e){this.pending.bytes+=e.max_bytes??0,this.pending.msgs+=e.batch??0,this.pending.requests++;const t=this.consumer.api.nc;this._push(()=>{t.publish(`${this.consumer.api.prefix}.CONSUMER.MSG.NEXT.${this.consumer.stream}.${this.consumer.name}`,this.consumer.api.jc.encode(e),{reply:this.inbox}),this.notify(_t.Next,e)})}pullOptions(){const e=this.opts.max_messages-this.pending.msgs,t=this.opts.max_bytes-this.pending.bytes,s=V(this.opts.idle_heartbeat),i=V(this.opts.expires);return{batch:e,max_bytes:t,idle_heartbeat:s,expires:i}}parseDiscard(e){const t={msgsLeft:0,bytesLeft:0},s=e?.get(ge.PendingMessagesHdr);s&&(t.msgsLeft=parseInt(s));const i=e?.get(ge.PendingBytesHdr);return i&&(t.bytesLeft=parseInt(i)),t}trackTimeout(e){this.timeout=e}close(){return this.stop(),this.iterClosed}closed(){return this.iterClosed}clearTimers(){this.monitor?.cancel(),this.monitor=null,this.timeout?.cancel(),this.timeout=null}setCleanupHandler(e){this.cleanupHandler=e}stop(e){this.done||(this.sub?.unsubscribe(),this.clearTimers(),this.statusIterator?.stop(),this._push(()=>{super.stop(e),this.listeners.forEach(t=>{t.stop()})}))}parseOptions(e,t=!1){const s=e||{};if(s.max_messages=s.max_messages||0,s.max_bytes=s.max_bytes||0,s.max_messages!==0&&s.max_bytes!==0)throw new Error("only specify one of max_messages or max_bytes");if(s.max_messages===0&&(s.max_messages=100),s.expires=s.expires||3e4,s.expires<1e3)throw new Error("expires should be at least 1000ms");if(s.idle_heartbeat=s.idle_heartbeat||s.expires/2,s.idle_heartbeat=s.idle_heartbeat>3e4?3e4:s.idle_heartbeat,t){const i=Math.round(s.max_messages*.75)||1;s.threshold_messages=s.threshold_messages||i;const n=Math.round(s.max_bytes*.75)||1;s.threshold_bytes=s.threshold_bytes||n}return s}status(){const e=new oe;return this.listeners.push(e),Promise.resolve(e)}}class so extends oe{src;listeners;constructor(){super(),this.listeners=[]}setSource(e){this.src&&(this.src.resetHandler=void 0,this.src.setCleanupHandler(),this.src.stop()),this.src=e,this.src.setCleanupHandler(t=>{this.stop(t||void 0)}),(async()=>{const t=await this.src.status();for await(const s of t)this.notify(s.type,s.data)})().catch(()=>{})}notify(e,t){this.listeners.length>0&&this.listeners.forEach(s=>{s.done||s.push({type:e,data:t})})}stop(e){this.done||(this.src?.stop(e),super.stop(e),this.listeners.forEach(t=>{t.stop()}))}close(){return this.stop(),this.iterClosed}closed(){return this.iterClosed}status(){const e=new oe;return this.listeners.push(e),Promise.resolve(e)}}class ir{api;_info;stream;name;constructor(e,t){this.api=e,this._info=t,this.stream=t.stream_name,this.name=t.name}consume(e={max_messages:100,expires:3e4}){return Promise.resolve(new zs(this,e,!0))}fetch(e={max_messages:100,expires:3e4}){const t=new zs(this,e,!1),s=Math.round(t.opts.expires*1.05),i=vt(s);return t.closed().catch(()=>{}).finally(()=>{i.cancel()}),i.catch(()=>{t.close().catch()}),t.trackTimeout(i),Promise.resolve(t)}next(e={expires:3e4}){const t=W(),s=e;s.max_messages=1;const i=new zs(this,s,!1),n=Math.round(i.opts.expires*1.05);n>=6e4&&(async()=>{for await(const a of await i.status())if(a.type===Le.HeartbeatsMissed&&a.data>=2){t.reject(new Error("consumer missed heartbeats"));break}})().catch(),(async()=>{for await(const a of i){t.resolve(a);break}})().catch(()=>{});const c=vt(n);return i.closed().then(a=>{a?t.reject(a):t.resolve(null)}).catch(a=>{t.reject(a)}).finally(()=>{c.cancel()}),c.catch(a=>{t.resolve(null),i.close().catch()}),i.trackTimeout(c),t}delete(){const{stream_name:e,name:t}=this._info;return this.api.delete(e,t)}info(e=!1){if(e)return Promise.resolve(this._info);const{stream_name:t,name:s}=this._info;return this.api.info(t,s).then(i=>(this._info=i,this._info))}}class ro{api;consumerOpts;consumer;opts;cursor;stream;namePrefix;serial;currentConsumer;userCallback;iter;type;startSeq;maxInitialReset;constructor(e,t,s={}){this.api=e,this.stream=t,this.cursor={stream_seq:1,deliver_seq:0},this.namePrefix=Ze.next(),typeof s.name_prefix=="string"&&(vs("name_prefix",s.name_prefix),this.namePrefix=s.name_prefix+this.namePrefix),this.serial=0,this.currentConsumer=null,this.userCallback=null,this.iter=null,this.type=Pe.Unset,this.consumerOpts=s,this.maxInitialReset=30,this.startSeq=this.consumerOpts.opt_start_seq||0,this.cursor.stream_seq=this.startSeq>0?this.startSeq-1:0}getConsumerOpts(e){this.serial++;const t=`${this.namePrefix}_${this.serial}`;e=e===0?1:e;const s={name:t,deliver_policy:Q.StartSequence,opt_start_seq:e,ack_policy:ae.None,inactive_threshold:V(300*1e3),num_replicas:1};return this.consumerOpts.headers_only===!0&&(s.headers_only=!0),Array.isArray(this.consumerOpts.filterSubjects)&&(s.filter_subjects=this.consumerOpts.filterSubjects),typeof this.consumerOpts.filterSubjects=="string"&&(s.filter_subject=this.consumerOpts.filterSubjects),this.consumerOpts.replay_policy&&(s.replay_policy=this.consumerOpts.replay_policy),e===this.startSeq+1&&(s.deliver_policy=this.consumerOpts.deliver_policy||Q.StartSequence,(this.consumerOpts.deliver_policy===Q.LastPerSubject||this.consumerOpts.deliver_policy===Q.New||this.consumerOpts.deliver_policy===Q.Last)&&(delete s.opt_start_seq,s.deliver_policy=this.consumerOpts.deliver_policy),s.deliver_policy===Q.LastPerSubject&&typeof s.filter_subjects>"u"&&typeof s.filter_subject>"u"&&(s.filter_subject=">"),this.consumerOpts.opt_start_time&&(delete s.opt_start_seq,s.deliver_policy=Q.StartTime,s.opt_start_time=this.consumerOpts.opt_start_time),this.consumerOpts.inactive_threshold&&(s.inactive_threshold=V(this.consumerOpts.inactive_threshold))),s}async resetConsumer(e=0){Ze.next();const t=this.serial===0;this.consumer?.delete().catch(()=>{}),e=e===0?1:e,this.cursor.deliver_seq=0;const s=this.getConsumerOpts(e);s.max_deliver=1,s.mem_storage=!0;const i=cr([this.opts?.expires||3e4]);let n;for(let c=0;;c++)try{n=await this.api.add(this.stream,s),this.iter?.notify(Le.OrderedConsumerRecreated,n.name);break}catch(a){if(a.message==="stream not found"&&(this.iter?.notify(Le.StreamNotFound,c),this.type===Pe.Fetch||this.opts.abort_on_missing_resource===!0))return this.iter?.stop(a),Promise.reject(a);if(t&&c>=this.maxInitialReset)throw a;await Et(i.backoff(c+1))}return n}internalHandler(e){return t=>{if(this.serial!==e)return;const s=t.info.deliverySequence;if(s!==this.cursor.deliver_seq+1){this.notifyOrderedResetAndReset();return}this.cursor.deliver_seq=s,this.cursor.stream_seq=t.info.streamSequence,this.userCallback?this.userCallback(t):this.iter?.push(t)}}async reset(e={max_messages:100,expires:3e4},t){t=t||{};const s=t.fromFetch||!1,i=t.orderedReset||!1;if(this.type===Pe.Fetch&&i){this.iter?.src.stop(),await this.iter?.closed(),this.currentConsumer=null;return}(this.currentConsumer===null||i)&&(this.currentConsumer=await this.resetConsumer(this.cursor.stream_seq+1)),(this.iter===null||s)&&(this.iter=new so),this.consumer=new ir(this.api,this.currentConsumer);const n=e;n.callback=this.internalHandler(this.serial);let c=null;this.type===Pe.Fetch&&s?c=await this.consumer.fetch(e):this.type===Pe.Consume&&(c=await this.consumer.consume(e));const a=c;a.forOrderedConsumer=!0,a.resetHandler=()=>{this.notifyOrderedResetAndReset()},this.iter.setSource(a)}notifyOrderedResetAndReset(){this.iter?.notify(_t.Reset,""),this.reset(this.opts,{orderedReset:!0})}async consume(e={max_messages:100,expires:3e4}){if(e.bind)return Promise.reject(new Error("bind is not supported"));if(this.type===Pe.Fetch)return Promise.reject(new Error("ordered consumer initialized as fetch"));if(this.type===Pe.Consume)return Promise.reject(new Error("ordered consumer doesn't support concurrent consume"));const{callback:s}=e;return s&&(this.userCallback=s),this.type=Pe.Consume,this.opts=e,await this.reset(e),this.iter}async fetch(e={max_messages:100,expires:3e4}){if(e.bind)return Promise.reject(new Error("bind is not supported"));if(this.type===Pe.Consume)return Promise.reject(new Error("ordered consumer already initialized as consume"));if(this.iter?.done===!1)return Promise.reject(new Error("ordered consumer doesn't support concurrent fetch"));const{callback:s}=e;return s&&(this.userCallback=s),this.type=Pe.Fetch,this.opts=e,await this.reset(e,{fromFetch:!0}),this.iter}async next(e={expires:3e4}){const t=e;if(t.bind)return Promise.reject(new Error("bind is not supported"));t.max_messages=1;const s=W();return t.callback=n=>{this.userCallback=null,s.resolve(n)},(await this.fetch(t)).iterClosed.then(n=>{n&&s.reject(n),s.resolve(null)}).catch(n=>{s.reject(n)}),s}delete(){return this.currentConsumer?this.api.delete(this.stream,this.currentConsumer.name).then(e=>Promise.resolve(e)).catch(e=>Promise.reject(e)).finally(()=>{this.currentConsumer=null}):Promise.resolve(!1)}async info(e){return this.currentConsumer==null?(this.currentConsumer=await this.resetConsumer(this.startSeq),Promise.resolve(this.currentConsumer)):e&&this.currentConsumer?Promise.resolve(this.currentConsumer):this.api.info(this.stream,this.currentConsumer.name)}}class nr{api;notified;constructor(e){this.api=e,this.notified=!1}checkVersion(){const e=this.api.nc.features.get(U.JS_SIMPLIFICATION);return e.ok?Promise.resolve():Promise.reject(new Error(`consumers framework is only supported on servers ${e.min} or better`))}getPullConsumerFor(e){if(e.config.deliver_subject!==void 0)throw new Error("push consumer not supported");return new ir(this.api,e)}async get(e,t={}){return typeof t=="object"?this.ordered(e,t):(await this.checkVersion(),this.api.info(e,t).then(s=>s.config.deliver_subject!==void 0?Promise.reject(new Error("push consumer not supported")):new ir(this.api,s)).catch(s=>Promise.reject(s)))}async ordered(e,t){await this.checkVersion();const s=this.api;return new dr(s.nc,s.opts).info(e).then(n=>Promise.resolve(new ro(this.api,e,t))).catch(n=>Promise.reject(n))}}class Es{api;_info;constructor(e,t){this.api=e,this._info=t}get name(){return this._info.config.name}alternates(){return this.info().then(e=>e.alternates?e.alternates:[])}async best(){if(await this.info(),this._info.alternates){const e=await this.api.info(this._info.alternates[0].name);return new Es(this.api,e)}else return this}info(e=!1,t){return e?Promise.resolve(this._info):this.api.info(this.name,t).then(s=>(this._info=s,this._info))}getConsumerFromInfo(e){return new nr(new bs(this.api.nc,this.api.opts)).getPullConsumerFor(e)}getConsumer(e){return new nr(new bs(this.api.nc,this.api.opts)).get(this.name,e)}getMessage(e){return this.api.getMessage(this.name,e)}deleteMessage(e,t){return this.api.deleteMessage(this.name,e,t)}}class dr extends Gt{constructor(e,t){super(e,t)}checkStreamConfigVersions(e){const t=this.nc;if(e.metadata){const{min:i,ok:n}=t.features.get(U.JS_STREAM_CONSUMER_METADATA);if(!n)throw new Error(`stream 'metadata' requires server ${i}`)}if(e.first_seq){const{min:i,ok:n}=t.features.get(U.JS_STREAM_FIRST_SEQ);if(!n)throw new Error(`stream 'first_seq' requires server ${i}`)}if(e.subject_transform){const{min:i,ok:n}=t.features.get(U.JS_STREAM_SUBJECT_TRANSFORM);if(!n)throw new Error(`stream 'subject_transform' requires server ${i}`)}if(e.compression){const{min:i,ok:n}=t.features.get(U.JS_STREAM_COMPRESSION);if(!n)throw new Error(`stream 'compression' requires server ${i}`)}if(e.consumer_limits){const{min:i,ok:n}=t.features.get(U.JS_DEFAULT_CONSUMER_LIMITS);if(!n)throw new Error(`stream 'consumer_limits' requires server ${i}`)}function s(i,n){if((n?.subject_transforms?.length||0)>0){const{min:a,ok:d}=t.features.get(U.JS_STREAM_SOURCE_SUBJECT_TRANSFORM);if(!d)throw new Error(`${i} 'subject_transforms' requires server ${a}`)}}e.sources&&e.sources.forEach(i=>{s("stream sources",i)}),e.mirror&&s("stream mirror",e.mirror)}async add(e={}){this.checkStreamConfigVersions(e),pe(e.name),e.mirror=os(e.mirror),e.sources=e.sources?.map(os);const s=await this._request(`${this.prefix}.STREAM.CREATE.${e.name}`,e);return this._fixInfo(s),s}async delete(e){return pe(e),(await this._request(`${this.prefix}.STREAM.DELETE.${e}`)).success}async update(e,t={}){if(typeof e=="object"){const a=e;e=a.name,t=a,console.trace("\x1B[33m >> streams.update(config: StreamConfig) api changed to streams.update(name: string, config: StreamUpdateConfig) - this shim will be removed - update your code.  \x1B[0m")}this.checkStreamConfigVersions(t),pe(e);const s=await this.info(e),i=Object.assign(s.config,t);i.mirror=os(i.mirror),i.sources=i.sources?.map(os);const c=await this._request(`${this.prefix}.STREAM.UPDATE.${e}`,i);return this._fixInfo(c),c}async info(e,t){pe(e);const s=`${this.prefix}.STREAM.INFO.${e}`;let n=await this._request(s,t),{total:c,limit:a}=n,d=n.state.subjects?Object.getOwnPropertyNames(n.state.subjects).length:1;if(c&&c>d){const m=[n],x=t||{};let v=0;for(;c>d;){v++,x.offset=a*v;const O=await this._request(s,x);c=O.total,m.push(O);const $=Object.getOwnPropertyNames(O.state.subjects).length;if(d+=$,$<a)break}let S={};for(let O=0;O<m.length;O++)n=m[O],n.state.subjects&&(S=Object.assign(S,n.state.subjects));n.offset=0,n.total=0,n.limit=0,n.state.subjects=S}return this._fixInfo(n),n}list(e=""){const t=e?.length?{subject:e}:{},s=n=>{const c=n;return c.streams.forEach(a=>{this._fixInfo(a)}),c.streams},i=`${this.prefix}.STREAM.LIST`;return new Mt(i,s,this,t)}_fixInfo(e){e.config.sealed=e.config.sealed||!1,e.config.deny_delete=e.config.deny_delete||!1,e.config.deny_purge=e.config.deny_purge||!1,e.config.allow_rollup_hdrs=e.config.allow_rollup_hdrs||!1}async purge(e,t){if(t){const{keep:i,seq:n}=t;if(typeof i=="number"&&typeof n=="number")throw new Error("can specify one of keep or seq")}return pe(e),await this._request(`${this.prefix}.STREAM.PURGE.${e}`,t)}async deleteMessage(e,t,s=!0){pe(e);const i={seq:t};return s||(i.no_erase=!0),(await this._request(`${this.prefix}.STREAM.MSG.DELETE.${e}`,i)).success}async getMessage(e,t){pe(e);const i=await this._request(`${this.prefix}.STREAM.MSG.GET.${e}`,t);return new ao(i)}find(e){return this.findStream(e)}listKvs(){const e=s=>{const n=s.streams.filter(d=>d.config.name.startsWith(_e));n.forEach(d=>{this._fixInfo(d)});let c="";return n.length&&(c=this.nc.info?.cluster??""),n.map(d=>new Si(d,c))},t=`${this.prefix}.STREAM.LIST`;return new Mt(t,e,this)}listObjectStores(){const e=s=>{const n=s.streams.filter(a=>a.config.name.startsWith(lr));return n.forEach(a=>{this._fixInfo(a)}),n.map(a=>new rr(a))},t=`${this.prefix}.STREAM.LIST`;return new Mt(t,e,this)}names(e=""){const t=e?.length?{subject:e}:{},s=n=>n.streams,i=`${this.prefix}.STREAM.NAMES`;return new Mt(i,s,this,t)}async get(e){const t=await this.info(e);return Promise.resolve(new Es(this,t))}}class io extends Gt{constructor(e,t){super(e,t)}async getMessage(e,t){pe(e);let s=t;const{last_by_subj:i}=s;i&&(s=null);const n=s?this.jc.encode(s):Ie,c=this.opts.apiPrefix||"$JS.API",a=i?`${c}.DIRECT.GET.${e}.${i}`:`${c}.DIRECT.GET.${e}`,d=await this.nc.request(a,n,{timeout:this.timeout}),m=wt(d);if(m)return Promise.reject(m);const x=new Qr(d);return Promise.resolve(x)}async getBatch(e,t){pe(e);const i=`${this.opts.apiPrefix||"$JS.API"}.DIRECT.GET.${e}`;if(!Array.isArray(t.multi_last)||t.multi_last.length===0)return Promise.reject("multi_last is required");const n=JSON.stringify(t,(d,m)=>d==="up_to_time"&&m instanceof Date?m.toISOString():m),c=new oe,a=await this.nc.requestMany(i,n,{strategy:Te.SentinelMsg});return(async()=>{let d=!1,m=!1,x;for await(const v of a){if(!d){d=!0;const S=v.headers?.code||0;if(S!==0&&S<200||S>299){x=v.headers?.description.toLowerCase();break}if(v.headers?.get("Nats-Num-Pending")===""){m=!0;break}}if(v.data.length===0)break;c.push(new Qr(v))}c.push(()=>{if(m)throw new Error("batch direct get not supported by the server");if(x)throw new Error(`bad request: ${x}`);c.stop()})})(),Promise.resolve(c)}}class Qr{data;header;static jc;constructor(e){if(!e.headers)throw new Error("headers expected");this.data=e.data,this.header=e.headers}get subject(){return this.header.last(pt.Subject)}get seq(){const e=this.header.last(pt.Sequence);return typeof e=="string"?parseInt(e):0}get time(){return new Date(Date.parse(this.timestamp))}get timestamp(){return this.header.last(pt.TimeStamp)}get stream(){return this.header.last(pt.Stream)}json(e){return $e(e).decode(this.data)}string(){return ke.decode(this.data)}}class no extends Gt{streams;consumers;direct;constructor(e,t){super(e,t),this.streams=new dr(e,t),this.consumers=new bs(e,t),this.direct=new io(e,t)}async getAccountInfo(){return await this._request(`${this.prefix}.INFO`)}jetstream(){return this.nc.jetstream(this.getOptions())}advisories(){const e=new oe;return this.nc.subscribe("$JS.EVENT.ADVISORY.>",{callback:(t,s)=>{if(t)throw t;try{const i=this.parseJsResponse(s),n=i.type.split("."),c=n[n.length-1];e.push({kind:c,data:i})}catch(i){e.stop(i)}}}),e}}class ao{_header;smr;static jc;constructor(e){this.smr=e}get subject(){return this.smr.message.subject}get seq(){return this.smr.message.seq}get timestamp(){return this.smr.message.time}get time(){return new Date(Date.parse(this.timestamp))}get data(){return this.smr.message.data?this._parse(this.smr.message.data):Ie}get header(){if(!this._header)if(this.smr.message.hdrs){const e=this._parse(this.smr.message.hdrs);this._header=We.decode(e)}else this._header=ze();return this._header}_parse(e){const t=atob(e),s=t.length,i=new Uint8Array(s);for(let n=0;n<s;n++)i[n]=t.charCodeAt(n);return i}json(e){return $e(e).decode(this.data)}string(){return ke.decode(this.data)}}class oo{api;constructor(e){this.api=e}get(e){return this.api.info(e).then(t=>new Es(this.api,t))}}class Js{info;hdrs;constructor(e){this.info=e}get name(){return this.info.name}get description(){return this.info.description??""}get headers(){return this.hdrs||(this.hdrs=We.fromRecord(this.info.headers||{})),this.hdrs}get options(){return this.info.options}get bucket(){return this.info.bucket}get chunks(){return this.info.chunks}get deleted(){return this.info.deleted??!1}get digest(){return this.info.digest}get mtime(){return this.info.mtime}get nuid(){return this.info.nuid}get size(){return this.info.size}get revision(){return this.info.revision}get metadata(){return this.info.metadata||{}}isLink(){return this.info.options?.link!==void 0&&this.info.options?.link!==null}}function ei(r){const e={name:r.name,description:r.description??"",options:r.options,metadata:r.metadata};if(r.headers){const t=r.headers;e.headers=t.toRecord()}return e}function co(){return new ReadableStream({pull(r){r.enqueue(new Uint8Array(0)),r.close()}})}class Ft{jsm;js;stream;name;constructor(e,t,s){this.name=e,this.jsm=t,this.js=s}_checkNotEmpty(e){return!e||e.length===0?{name:e,error:new Error("name cannot be empty")}:{name:e}}async info(e){const t=await this.rawInfo(e);return t?new Js(t):null}async list(){const e=[],t=await this.watch({ignoreDeletes:!0,includeHistory:!0});for await(const s of t){if(s===null)break;e.push(s)}return Promise.resolve(e)}async rawInfo(e){const{name:t,error:s}=this._checkNotEmpty(e);if(s)return Promise.reject(s);const i=this._metaSubject(t);try{const n=await this.jsm.streams.getMessage(this.stream,{last_by_subj:i}),a=$e().decode(n.data);return a.revision=n.seq,a}catch(n){return n.code==="404"?null:Promise.reject(n)}}async _si(e){try{return await this.jsm.streams.info(this.stream,e)}catch(t){return t.code==="404"?null:Promise.reject(t)}}async seal(){let e=await this._si();return e===null?Promise.reject(new Error("object store not found")):(e.config.sealed=!0,e=await this.jsm.streams.update(this.stream,e.config),Promise.resolve(new rr(e)))}async status(e){const t=await this._si(e);return t===null?Promise.reject(new Error("object store not found")):Promise.resolve(new rr(t))}destroy(){return this.jsm.streams.delete(this.stream)}async _put(e,t,s){const i=this.js.getOptions();s=s||{timeout:i.timeout},s.timeout=s.timeout||i.timeout,s.previousRevision=s.previousRevision??void 0;const{timeout:n,previousRevision:c}=s,d=this.js.nc.info?.max_payload||1024;e=e||{},e.options=e.options||{};let m=e.options?.max_chunk_size||128*1024;m=m>d?d:m,e.options.max_chunk_size=m;const x=await this.info(e.name),{name:v,error:S}=this._checkNotEmpty(e.name);if(S)return Promise.reject(S);const O=Ze.next(),$=this._chunkSubject(O),K=this._metaSubject(v),ee=Object.assign({bucket:this.name,nuid:O,size:0,chunks:0},ei(e)),B=W(),D=[],de=new kt;try{const le=t?t.getReader():null,g=Hr.create();for(;;){const{done:R,value:q}=le?await le.read():{done:!0,value:void 0};if(R){if(de.size()>0){const M=de.drain();g.update(M),ee.chunks++,ee.size+=M.length,D.push(this.js.publish($,M,{timeout:n}))}await Promise.all(D),D.length=0,ee.mtime=new Date().toISOString();const P=xt.encode(g.digest());ee.digest=`${Xr}${P}`,ee.deleted=!1;const k=ze();typeof c=="number"&&k.set(Fe.ExpectedLastSubjectSequenceHdr,`${c}`),k.set(ge.RollupHdr,ge.RollupValueSubject);const N=await this.js.publish(K,$e().encode(ee),{headers:k,timeout:n});if(ee.revision=N.seq,x)try{await this.jsm.streams.purge(this.stream,{filter:`$O.${this.name}.C.${x.nuid}`})}catch{}B.resolve(new Js(ee));break}if(q)for(de.fill(q);de.size()>m;){ee.chunks++,ee.size+=m;const P=de.drain(e.options.max_chunk_size);g.update(P),D.push(this.js.publish($,P,{timeout:n}))}}}catch(le){await this.jsm.streams.purge(this.stream,{filter:$}),B.reject(le)}return B}putBlob(e,t,s){function i(n){return new ReadableStream({pull(c){c.enqueue(n),c.close()}})}return t===null&&(t=new Uint8Array(0)),this.put(e,i(t),s)}put(e,t,s){return e?.options?.link?Promise.reject(new Error("link cannot be set when putting the object in bucket")):this._put(e,t,s)}async getBlob(e){async function t(n){const c=new kt,a=n.getReader();for(;;){const{done:d,value:m}=await a.read();if(d)return c.drain();m&&m.length&&c.fill(m)}}const s=await this.get(e);if(s===null)return Promise.resolve(null);const i=await Promise.all([s.error,t(s.data)]);return i[0]?Promise.reject(i[0]):Promise.resolve(i[1])}async get(e){const t=await this.rawInfo(e);if(t===null||t.deleted)return Promise.resolve(null);if(t.options&&t.options.link){const v=t.options.link.name||"";if(v==="")throw new Error("link is a bucket");return(t.options.link.bucket!==this.name?await Ft.create(this.js,t.options.link.bucket):this).get(v)}if(!t.digest.startsWith(Xr))return Promise.reject(new Error(`unknown digest type: ${t.digest}`));const s=Zs(t.digest.substring(8));if(s===null)return Promise.reject(new Error(`unable to parse digest: ${t.digest}`));const i=W(),n={info:new Js(t),error:i};if(t.size===0)return n.data=co(),i.resolve(null),Promise.resolve(n);let c;const a=Ye();a.orderedConsumer();const d=Hr.create(),m=`$O.${this.name}.C.${t.nuid}`,x=await this.js.subscribe(m,a);return(async()=>{for await(const v of x)v.data.length>0&&(d.update(v.data),c.enqueue(v.data)),v.info.pending===0&&($n(s,d.digest())?c.close():c.error(new Error(`received a corrupt object, digests do not match received: ${t.digest} calculated ${s}`)),x.unsubscribe())})().then(()=>{i.resolve()}).catch(v=>{c.error(v),i.reject(v)}),n.data=new ReadableStream({start(v){c=v},cancel(){x.unsubscribe()}}),n}linkStore(e,t){if(!(t instanceof Ft))return Promise.reject("bucket required");const s=t,{name:i,error:n}=this._checkNotEmpty(e);if(n)return Promise.reject(n);const c={name:i,options:{link:{bucket:s.name}}};return this._put(c,null)}async link(e,t){const{name:s,error:i}=this._checkNotEmpty(e);if(i)return Promise.reject(i);if(t.deleted)return Promise.reject(new Error("src object is deleted"));if(t.isLink())return Promise.reject(new Error("src object is a link"));const n=await this.rawInfo(e);if(n!==null&&!n.deleted)return Promise.reject(new Error("an object already exists with that name"));const c={bucket:t.bucket,name:t.name},a={name:s,bucket:t.bucket,options:{link:c}};await this.js.publish(this._metaSubject(e),JSON.stringify(a));const d=await this.info(e);return Promise.resolve(d)}async delete(e){const t=await this.rawInfo(e);if(t===null)return Promise.resolve({purged:0,success:!1});t.deleted=!0,t.size=0,t.chunks=0,t.digest="";const s=$e(),i=ze();return i.set(ge.RollupHdr,ge.RollupValueSubject),await this.js.publish(this._metaSubject(t.name),s.encode(t),{headers:i}),this.jsm.streams.purge(this.stream,{filter:this._chunkSubject(t.nuid)})}async update(e,t={}){const s=await this.rawInfo(e);if(s===null)return Promise.reject(new Error("object not found"));if(s.deleted)return Promise.reject(new Error("cannot update meta for a deleted object"));t.name=t.name??s.name;const{name:i,error:n}=this._checkNotEmpty(t.name);if(n)return Promise.reject(n);if(e!==t.name){const d=await this.info(t.name);if(d&&!d.deleted)return Promise.reject(new Error("an object already exists with that name"))}t.name=i;const c=Object.assign({},s,ei(t)),a=await this.js.publish(this._metaSubject(c.name),JSON.stringify(c));return e!==t.name&&await this.jsm.streams.purge(this.stream,{filter:this._metaSubject(e)}),Promise.resolve(a)}async watch(e={}){e.includeHistory=e.includeHistory??!1,e.ignoreDeletes=e.ignoreDeletes??!1;let t=!1;const s=new oe,i=this._metaSubjectAll();try{await this.jsm.streams.getMessage(this.stream,{last_by_subj:i})}catch(d){d.code==="404"?(s.push(null),t=!0):s.stop(d)}const n=$e(),c=Ye();c.orderedConsumer(),e.includeHistory?c.deliverLastPerSubject():(t=!0,c.deliverNew()),c.callback((d,m)=>{if(d){s.stop(d);return}if(m!==null){const x=n.decode(m.data);x.deleted&&e.ignoreDeletes===!0||s.push(x),m.info?.pending===0&&!t&&(t=!0,s.push(null))}});const a=await this.js.subscribe(i,c);return s._data=a,s.iterClosed.then(()=>{a.unsubscribe()}),a.closed.then(()=>{s.stop()}).catch(d=>{s.stop(d)}),s}_chunkSubject(e){return`$O.${this.name}.C.${e}`}_metaSubject(e){return`$O.${this.name}.M.${xt.encode(e)}`}_metaSubjectAll(){return`$O.${this.name}.M.>`}async init(e={}){try{this.stream=Ya(this.name)}catch(i){return Promise.reject(i)}const t=e?.ttl||0;delete e.ttl;const s=Object.assign({max_age:t},e);s.name=this.stream,s.num_replicas=e.replicas??1,s.allow_direct=!0,s.allow_rollup_hdrs=!0,s.discard=Dt.New,s.subjects=[`$O.${this.name}.C.>`,`$O.${this.name}.M.>`],e.placement&&(s.placement=e.placement),e.metadata&&(s.metadata=e.metadata),typeof e.compression=="boolean"&&(s.compression=e.compression?Qe.S2:Qe.None);try{await this.jsm.streams.info(s.name)}catch(i){i.message==="stream not found"&&await this.jsm.streams.add(s)}}static async create(e,t,s={}){const i=await e.jetstreamManager(),n=new Ft(t,i,e);return await n.init(s),Promise.resolve(n)}}class uo{js;constructor(e){this.js=e}kv(e,t={}){const s=this.js,{ok:i,min:n}=s.nc.features.get(U.JS_KV);return i?t.bindOnly?zt.bind(this.js,e,t):zt.create(this.js,e,t):Promise.reject(new Error(`kv is only supported on servers ${n} or better`))}os(e,t={}){if(typeof crypto?.subtle?.digest!="function")return Promise.reject(new Error("objectstore: unable to calculate hashes - crypto.subtle.digest with sha256 support is required"));const s=this.js,{ok:i,min:n}=s.nc.features.get(U.JS_OBJECTSTORE);return i?Ft.create(this.js,e,t):Promise.reject(new Error(`objectstore is only supported on servers ${n} or better`))}}class fr extends Gt{consumers;streams;consumerAPI;streamAPI;constructor(e,t){super(e,t),this.consumerAPI=new bs(e,t),this.streamAPI=new dr(e,t),this.consumers=new nr(this.consumerAPI),this.streams=new oo(this.streamAPI)}jetstreamManager(e){e===void 0&&(e=this.opts.checkAPI);const t=Object.assign({},this.opts,{checkAPI:e});return this.nc.jetstreamManager(t)}get apiPrefix(){return this.prefix}get views(){return new uo(this)}async publish(e,t=Ie,s){s=s||{},s.expect=s.expect||{};const i=s?.headers||ze();s&&(s.msgID&&i.set(Fe.MsgIdHdr,s.msgID),s.expect.lastMsgID&&i.set(Fe.ExpectedLastMsgIdHdr,s.expect.lastMsgID),s.expect.streamName&&i.set(Fe.ExpectedStreamHdr,s.expect.streamName),typeof s.expect.lastSequence=="number"&&i.set(Fe.ExpectedLastSeqHdr,`${s.expect.lastSequence}`),typeof s.expect.lastSubjectSequence=="number"&&i.set(Fe.ExpectedLastSubjectSequenceHdr,`${s.expect.lastSubjectSequence}`));const n=s.timeout||this.timeout,c={};n&&(c.timeout=n),s&&(c.headers=i);let{retries:a,retry_delay:d}=s;a=a||1,d=d||250;let m;for(let v=0;v<a;v++)try{m=await this.nc.request(e,t,c);break}catch(S){if(S.code==="503"&&v+1<a)await Et(d);else throw S}const x=this.parseJsResponse(m);if(x.stream==="")throw j.errorForCode(E.JetStreamInvalidAck);return x.duplicate=x.duplicate?x.duplicate:!1,x}async pull(e,t,s=0){pe(e),yt(t);let i=this.timeout;s>i&&(i=s),s=s<0?0:V(s);const n={batch:1,no_wait:s===0,expires:s},c=await this.nc.request(`${this.prefix}.CONSUMER.MSG.NEXT.${e}.${t}`,this.jc.encode(n),{noMux:!0,timeout:i}),a=wt(c);if(a)throw a;return Jt(c,this.timeout)}fetch(e,t,s={}){pe(e),yt(t);let i=null;const n=(s.max_bytes??0)>0;let c=0;const a=n?s.max_bytes:0;let d=null;const m={};if(m.batch=s.batch||1,a){const B=this.nc.features.get(U.JS_PULL_MAX_BYTES);if(!B.ok)throw new Error(`max_bytes is only supported on servers ${B.min} or better`);m.max_bytes=a}m.no_wait=s.no_wait||!1,m.no_wait&&m.expires&&(m.expires=0);const x=s.expires||0;if(x&&(m.expires=V(x)),x===0&&m.no_wait===!1)throw new Error("expires or no_wait is required");const v=s.idle_heartbeat||0;v&&(m.idle_heartbeat=V(v),s.delay_heartbeat===!0&&(m.idle_heartbeat=V(v*4)));const S=new oe,O=m.batch;let $=0;S.protocolFilterFn=(B,D=!1)=>Ws(B.msg)?(d?.work(),!1):!0,S.dispatchedFn=B=>{if(B){if(n&&(c+=B.data.length),$++,i&&B.info.pending===0)return;(S.getPending()===1&&B.info.pending===0||O===$||a>0&&c>=a)&&S.stop()}};const K=He(this.nc.options.inboxPrefix),ee=this.nc.subscribe(K,{max:s.batch,callback:(B,D)=>{B===null&&(B=wt(D)),B!==null?(i&&(i.cancel(),i=null),gn(B)?S.stop(Ei(B)===null?void 0:B):S.stop(B)):(d?.work(),S.received++,S.push(Jt(D,this.timeout)))}});return x&&(i=vt(x),i.catch(()=>{ee.isClosed()||(ee.drain().catch(()=>{}),i=null),d&&d.cancel()})),(async()=>{try{v&&(d=new hr(v,B=>(S.push(()=>{S.err=new j(`${Re.IdleHeartbeatMissed}: ${B}`,E.JetStreamIdleHeartBeat)}),!0)))}catch{}await ee.closed,i!==null&&(i.cancel(),i=null),d&&d.cancel(),S.stop()})().catch(),this.nc.publish(`${this.prefix}.CONSUMER.MSG.NEXT.${e}.${t}`,this.jc.encode(m),{reply:K}),S}async pullSubscribe(e,t=Ye()){const s=await this._processOptions(e,t);if(s.ordered)throw new Error("pull subscribers cannot be be ordered");if(s.config.deliver_subject)throw new Error("consumer info specifies deliver_subject - pull consumers cannot have deliver_subject set");const i=s.config.ack_policy;if(i===ae.None||i===ae.All)throw new Error("ack policy for pull consumers must be explicit");const n=this._buildTypedSubscriptionOpts(s),c=new po(this,s.deliver,n);c.info=s;try{await this._maybeCreateConsumer(s)}catch(a){throw c.unsubscribe(),a}return c}async subscribe(e,t=Ye()){const s=await this._processOptions(e,t);if(!s.isBind&&!s.config.deliver_subject)throw new Error("push consumer requires deliver_subject");const i=this._buildTypedSubscriptionOpts(s),n=new ki(this,s.deliver,i);n.info=s;try{await this._maybeCreateConsumer(s)}catch(c){throw n.unsubscribe(),c}return n._maybeSetupHbMonitoring(),n}async _processOptions(e,t=Ye()){const s=Lr(t)?t.getOpts():t;if(s.isBind=Lr(t)?t.isBind:!1,s.flow_control={heartbeat_count:0,fc_count:0,consumer_restarts:0},s.ordered){if(s.ordered_consumer_sequence={stream_seq:0,delivery_seq:0},s.config.ack_policy!==ae.NotSet&&s.config.ack_policy!==ae.None)throw new j("ordered consumer: ack_policy can only be set to 'none'",E.ApiError);if(s.config.durable_name&&s.config.durable_name.length>0)throw new j("ordered consumer: durable_name cannot be set",E.ApiError);if(s.config.deliver_subject&&s.config.deliver_subject.length>0)throw new j("ordered consumer: deliver_subject cannot be set",E.ApiError);if(s.config.max_deliver!==void 0&&s.config.max_deliver>1)throw new j("ordered consumer: max_deliver cannot be set",E.ApiError);if(s.config.deliver_group&&s.config.deliver_group.length>0)throw new j("ordered consumer: deliver_group cannot be set",E.ApiError);s.config.deliver_subject=He(this.nc.options.inboxPrefix),s.config.ack_policy=ae.None,s.config.max_deliver=1,s.config.flow_control=!0,s.config.idle_heartbeat=s.config.idle_heartbeat||V(5e3),s.config.ack_wait=V(1320*60*1e3),s.config.mem_storage=!0,s.config.num_replicas=1}if(s.config.ack_policy===ae.NotSet&&(s.config.ack_policy=ae.All),s.api=this,s.config=s.config||{},s.stream=s.stream?s.stream:await this.findStream(e),s.attached=!1,s.config.durable_name)try{const i=await this.consumerAPI.info(s.stream,s.config.durable_name);if(i){if(i.config.filter_subject&&i.config.filter_subject!==e)throw new Error("subject does not match consumer");const n=s.config.deliver_group??"";if(n===""&&i.push_bound===!0)throw new Error("duplicate subscription");const c=i.config.deliver_group??"";if(n!==c)throw c===""?new Error("durable requires no queue group"):new Error(`durable requires queue group '${c}'`);s.last=i,s.config=i.config,s.attached=!0,s.config.durable_name||(s.name=i.name)}}catch(i){if(i.code!=="404")throw i}return!s.attached&&s.config.filter_subject===void 0&&s.config.filter_subjects===void 0&&(s.config.filter_subject=e),s.deliver=s.config.deliver_subject||He(this.nc.options.inboxPrefix),s}_buildTypedSubscriptionOpts(e){const t={};return t.adapter=mo(e.callbackFn===void 0,this.timeout),t.ingestionFilterFn=fr.ingestionFn(e.ordered),t.protocolFilterFn=(s,i=!1)=>{const n=s;return Vs(n.msg)?(i||n.msg.respond(),!1):!0},!e.mack&&e.config.ack_policy!==ae.None&&(t.dispatchedFn=yo),e.callbackFn&&(t.callback=e.callbackFn),t.max=e.max||0,t.queue=e.queue,t}async _maybeCreateConsumer(e){if(e.attached)return;if(e.isBind)throw new Error(`unable to bind - durable consumer ${e.config.durable_name} doesn't exist in ${e.stream}`);e.config=Object.assign({deliver_policy:Q.All,ack_policy:ae.Explicit,ack_wait:V(30*1e3),replay_policy:St.Instant},e.config);const t=await this.consumerAPI.add(e.stream,e.config);if(Array.isArray(e.config.filter_subjects&&!Array.isArray(t.config.filter_subjects)))throw new Error("jetstream server doesn't support consumers with multiple filter subjects");e.name=t.name,e.config=t.config,e.last=t}static ingestionFn(e){return(t,s)=>{const i=s;if(!t)return{ingest:!1,protocol:!1};const n=t;if(wt(n.msg)||i.monitor?.work(),Ws(n.msg)){const a=e?i._checkHbOrderConsumer(n.msg):!0;return e||i.info.flow_control.heartbeat_count++,{ingest:a,protocol:!0}}else if(Vs(n.msg))return i.info.flow_control.fc_count++,{ingest:!0,protocol:!0};return{ingest:e?i._checkOrderedConsumer(t):!0,protocol:!1}}}}class pr{options;protocol;draining;listeners;_services;constructor(e){this.draining=!1,this.options=Oa(e),this.listeners=[]}static connect(e={}){return new Promise((t,s)=>{const i=new pr(e);ws.connect(i.options,i).then(n=>{i.protocol=n,(async function(){for await(const c of n.status())i.listeners.forEach(a=>{a.push(c)})})(),t(i)}).catch(n=>{s(n)})})}closed(){return this.protocol.closed}async close(){await this.protocol.close()}_check(e,t,s){if(this.isClosed())throw j.errorForCode(E.ConnectionClosed);if(t&&this.isDraining()||s&&this.protocol.noMorePublishing)throw j.errorForCode(E.ConnectionDraining);if(e=e||"",e.length===0)throw j.errorForCode(E.BadSubject)}publish(e,t,s){this._check(e,!1,!0),this.protocol.publish(e,t,s)}publishMessage(e){return this.publish(e.subject,e.data,{reply:e.reply,headers:e.headers})}respondMessage(e){return e.reply?(this.publish(e.reply,e.data,{reply:e.reply,headers:e.headers}),!0):!1}subscribe(e,t={}){this._check(e,!0,!1);const s=new _i(this.protocol,e,t);return this.protocol.subscribe(s),s}_resub(e,t,s){this._check(t,!0,!1);const i=e;i.max=s,s&&(i.max=s+i.received),this.protocol.resub(i,t)}requestMany(e,t=Ie,s={maxWait:1e3,maxMessages:-1}){const i=!this.protocol.options.noAsyncTraces;try{this._check(e,!0,!0)}catch(d){return Promise.reject(d)}if(s.strategy=s.strategy||Te.Timer,s.maxWait=s.maxWait||1e3,s.maxWait<1)return Promise.reject(new j("timeout",E.InvalidOption));const n=new oe;function c(d){n.push(()=>{n.stop(d)})}function a(d,m){d||m===null?c(d===null?void 0:d):n.push(m)}if(s.noMux){const d=i?new Error().stack:null;let m=typeof s.maxMessages=="number"&&s.maxMessages>0?s.maxMessages:-1;const x=this.subscribe(He(this.options.inboxPrefix),{callback:($,K)=>{if(K?.data?.length===0&&K?.headers?.status===E.NoResponders&&($=j.errorForCode(E.NoResponders)),$){d&&($.stack+=`

${d}`),v($);return}a(null,K),s.strategy===Te.Count&&(m--,m===0&&v()),s.strategy===Te.JitterTimer&&(O(),S=setTimeout(()=>{v()},300)),s.strategy===Te.SentinelMsg&&K&&K.data.length===0&&v()}});x.requestSubject=e,x.closed.then(()=>{c()}).catch($=>{n.stop($)});const v=$=>{$&&n.push(()=>{throw $}),O(),x.drain().then(()=>{c()}).catch(K=>{c()})};n.iterClosed.then(()=>{O(),x?.unsubscribe()}).catch($=>{O(),x?.unsubscribe()});try{this.publish(e,t,{reply:x.getSubject()})}catch($){v($)}let S=setTimeout(()=>{v()},s.maxWait);const O=()=>{S&&clearTimeout(S)}}else{const d=s;d.callback=a,n.iterClosed.then(()=>{m.cancel()}).catch(x=>{m.cancel(x)});const m=new qn(this.protocol.muxSubscriptions,e,d);this.protocol.request(m);try{this.publish(e,t,{reply:`${this.protocol.muxSubscriptions.baseInbox}${m.token}`,headers:s.headers})}catch(x){m.cancel(x)}}return Promise.resolve(n)}request(e,t,s={timeout:1e3,noMux:!1}){try{this._check(e,!0,!0)}catch(n){return Promise.reject(n)}const i=!this.protocol.options.noAsyncTraces;if(s.timeout=s.timeout||1e3,s.timeout<1)return Promise.reject(new j("timeout",E.InvalidOption));if(!s.noMux&&s.reply)return Promise.reject(new j("reply can only be used with noMux",E.InvalidOption));if(s.noMux){const n=s.reply?s.reply:He(this.options.inboxPrefix),c=W(),a=i?new Error:null,d=this.subscribe(n,{max:1,timeout:s.timeout,callback:(m,x)=>{m?(a&&m.code!==E.Timeout&&(m.stack+=`

${a.stack}`),d.unsubscribe(),c.reject(m)):(m=ci(x),m?(a&&(m.stack+=`

${a.stack}`),c.reject(m)):c.resolve(x))}});return d.requestSubject=e,this.protocol.publish(e,t,{reply:n,headers:s.headers}),c}else{const n=new di(this.protocol.muxSubscriptions,e,s,i);this.protocol.request(n);try{this.publish(e,t,{reply:`${this.protocol.muxSubscriptions.baseInbox}${n.token}`,headers:s.headers})}catch(a){n.cancel(a)}const c=Promise.race([n.timer,n.deferred]);return c.catch(()=>{n.cancel()}),c}}flush(){return this.isClosed()?Promise.reject(j.errorForCode(E.ConnectionClosed)):this.protocol.flush()}drain(){return this.isClosed()?Promise.reject(j.errorForCode(E.ConnectionClosed)):this.isDraining()?Promise.reject(j.errorForCode(E.ConnectionDraining)):(this.draining=!0,this.protocol.drain())}isClosed(){return this.protocol.isClosed()}isDraining(){return this.draining}getServer(){const e=this.protocol.getServer();return e?e.listen:""}status(){const e=new oe;return e.iterClosed.then(()=>{const t=this.listeners.indexOf(e);this.listeners.splice(t,1)}),this.listeners.push(e),e}get info(){return this.protocol.isClosed()?void 0:this.protocol.info}async context(){return(await this.request("$SYS.REQ.USER.INFO")).json((t,s)=>t==="time"?new Date(Date.parse(s)):s)}stats(){return{inBytes:this.protocol.inBytes,outBytes:this.protocol.outBytes,inMsgs:this.protocol.inMsgs,outMsgs:this.protocol.outMsgs}}async jetstreamManager(e={}){const t=new no(this,e);if(e.checkAPI!==!1)try{await t.getAccountInfo()}catch(s){const i=s;throw i.code===E.NoResponders&&(i.code=E.JetStreamNotEnabled),i}return t}jetstream(e={}){return new fr(this,e)}getServerVersion(){const e=this.info;return e?rt(e.version):void 0}async rtt(){if(!this.protocol._closed&&!this.protocol.connected)throw j.errorForCode(E.Disconnect);const e=Date.now();return await this.flush(),Date.now()-e}get features(){return this.protocol.features}get services(){return this._services||(this._services=new ho(this)),this._services}reconnect(){return this.isClosed()?Promise.reject(j.errorForCode(E.ConnectionClosed)):this.isDraining()?Promise.reject(j.errorForCode(E.ConnectionDraining)):this.protocol.reconnect()}}class ho{nc;constructor(e){this.nc=e}add(e){try{return new Vt(this.nc,e).start()}catch(t){return Promise.reject(t)}}client(e,t){return new Da(this.nc,e,t)}}class lo{bucket;sm;prefixLen;constructor(e,t,s){this.bucket=e,this.prefixLen=t,this.sm=s}get key(){return this.sm.subject.substring(this.prefixLen)}get value(){return this.sm.data}get delta(){return 0}get created(){return this.sm.time}get revision(){return this.sm.seq}get operation(){return this.sm.header.get(xs)||"PUT"}get length(){const e=this.sm.header.get(ge.MessageSizeHdr)||"";return e!==""?parseInt(e,10):this.sm.data.length}json(){return this.sm.json()}string(){return this.sm.string()}}class fo{bucket;key;sm;constructor(e,t,s){this.bucket=e,this.key=t,this.sm=s}get value(){return this.sm.data}get created(){return new Date(ur(this.sm.info.timestampNanos))}get revision(){return this.sm.seq}get operation(){return this.sm.headers?.get(xs)||"PUT"}get delta(){return this.sm.info.pending}get length(){const e=this.sm.headers?.get(ge.MessageSizeHdr)||"";return e!==""?parseInt(e,10):this.sm.data.length}json(){return this.sm.json()}string(){return this.sm.string()}}class ki extends Bn{js;monitor;constructor(e,t,s){super(e.nc,t,s),this.js=e,this.monitor=null,this.sub.closed.then(()=>{this.monitor&&this.monitor.cancel()})}set info(e){this.sub.info=e}get info(){return this.sub.info}_resetOrderedConsumer(e){if(this.info===null||this.sub.isClosed())return;const t=He(this.js.nc.options.inboxPrefix);this.js.nc._resub(this.sub,t);const i=this.info;i.config.name=Ze.next(),i.ordered_consumer_sequence.delivery_seq=0,i.flow_control.heartbeat_count=0,i.flow_control.fc_count=0,i.flow_control.consumer_restarts++,i.deliver=t,i.config.deliver_subject=t,i.config.deliver_policy=Q.StartSequence,i.config.opt_start_seq=e;const n={};n.stream_name=this.info.stream,n.config=i.config;const c=`${i.api.prefix}.CONSUMER.CREATE.${i.stream}`;this.js._request(c,n,{retries:-1}).then(a=>{const d=a,m=this.sub.info;m.last=d,this.info.config=d.config,this.info.name=d.name}).catch(a=>{const d=new j(`unable to recreate ordered consumer ${i.stream} at seq ${e}`,E.RequestError,a);this.sub.callback(d,{})})}_maybeSetupHbMonitoring(){const e=this.info?.config?.idle_heartbeat||0;e&&this._setupHbMonitoring(ur(e))}_setupHbMonitoring(e,t=0){const s={cancelAfter:0,maxOut:2};t&&(s.cancelAfter=t);const i=this.sub,n=c=>{const a=_n(409,`${Re.IdleHeartbeatMissed}: ${c}`,this.sub.subject);if(!this.info?.ordered)this.sub.callback(null,a);else{if(!this.js.nc.protocol.connected)return!1;const m=this.info?.ordered_consumer_sequence?.stream_seq||0;return this._resetOrderedConsumer(m+1),this.monitor?.restart(),!1}return!i.noIterator};this.monitor=new hr(e,n,s)}_checkHbOrderConsumer(e){const t=e.headers.get(ge.ConsumerStalledHdr);t!==""&&this.js.nc.publish(t);const s=parseInt(e.headers.get(ge.LastConsumerSeqHdr),10),i=this.info.ordered_consumer_sequence;return this.info.flow_control.heartbeat_count++,s!==i.delivery_seq&&this._resetOrderedConsumer(i.stream_seq+1),!1}_checkOrderedConsumer(e){const t=this.info.ordered_consumer_sequence,s=e.info.streamSequence,i=e.info.deliverySequence;return i!=t.delivery_seq+1?(this._resetOrderedConsumer(t.stream_seq+1),!1):(t.delivery_seq=i,t.stream_seq=s,!0)}async destroy(){this.isClosed()||await this.drain();const e=this.sub.info,t=e.config.durable_name||e.name,s=`${e.api.prefix}.CONSUMER.DELETE.${e.stream}.${t}`;await e.api._request(s)}async consumerInfo(){const e=this.sub.info,t=e.config.durable_name||e.name,s=`${e.api.prefix}.CONSUMER.INFO.${e.stream}.${t}`,i=await e.api._request(s);return e.last=i,i}}class po extends ki{constructor(e,t,s){super(e,t,s)}pull(e={batch:1}){const{stream:t,config:s,name:i}=this.sub.info,n=s.durable_name??i,c={};if(c.batch=e.batch||1,c.no_wait=e.no_wait||!1,(e.max_bytes??0)>0){const m=this.js.nc.features.get(U.JS_PULL_MAX_BYTES);if(!m.ok)throw new Error(`max_bytes is only supported on servers ${m.min} or better`);c.max_bytes=e.max_bytes}let a=0;e.expires&&e.expires>0&&(a=e.expires,c.expires=V(a));let d=0;if(e.idle_heartbeat&&e.idle_heartbeat>0&&(d=e.idle_heartbeat,c.idle_heartbeat=V(d)),d&&a===0)throw new Error("idle_heartbeat requires expires");if(d>a)throw new Error("expires must be greater than idle_heartbeat");if(this.info){this.monitor&&this.monitor.cancel(),a&&d&&(this.monitor?this.monitor._change(d,a):this._setupHbMonitoring(d,a));const m=this.info.api,x=`${m.prefix}.CONSUMER.MSG.NEXT.${t}.${n}`,v=this.sub.subject;m.nc.publish(x,m.jc.encode(c),{reply:v})}}}function mo(r,e){return r?go(e):bo(e)}function bo(r){return(e,t)=>e?[e,null]:(e=wt(t),e?[e,null]:[null,Jt(t,r)])}function go(r){return(e,t)=>{if(e)return[e,null];const s=wt(t);return s!==null?[Ei(s),null]:[null,Jt(t,r)]}}function Ei(r){if(r!==null)switch(r.code){case E.JetStream404NoMessages:case E.JetStream408RequestTimeout:return null;case E.JetStream409:return vn(r)?r:null;default:return r}return null}function yo(r){r&&r.ack()}function wo(r){const e=r.split(".");if(e.length===9&&e.splice(2,0,"_",""),e.length<11||e[0]!=="$JS"||e[1]!=="ACK")throw new Error("not js message");const t={};return t.domain=e[2]==="_"?"":e[2],t.account_hash=e[3],t.stream=e[4],t.consumer=e[5],t.deliveryCount=parseInt(e[6],10),t.redeliveryCount=t.deliveryCount,t.redelivered=t.deliveryCount>1,t.streamSequence=parseInt(e[7],10),t.deliverySequence=parseInt(e[8],10),t.timestampNanos=parseInt(e[9],10),t.pending=parseInt(e[10],10),t}class xo{msg;di;didAck;timeout;constructor(e,t){this.msg=e,this.didAck=!1,this.timeout=t}get subject(){return this.msg.subject}get sid(){return this.msg.sid}get data(){return this.msg.data}get headers(){return this.msg.headers}get info(){return this.di||(this.di=wo(this.reply)),this.di}get redelivered(){return this.info.deliveryCount>1}get reply(){return this.msg.reply||""}get seq(){return this.info.streamSequence}doAck(e){this.didAck||(this.didAck=!this.isWIP(e),this.msg.respond(e))}isWIP(e){return e.length===4&&e[0]===Ct[0]&&e[1]===Ct[1]&&e[2]===Ct[2]&&e[3]===Ct[3]}async ackAck(e){e=e||{},e.timeout=e.timeout||this.timeout;const t=W();if(this.didAck)t.resolve(!1);else if(this.didAck=!0,this.msg.reply){const i=this.msg.publisher,n=!i.options?.noAsyncTraces,c=new di(i.muxSubscriptions,this.msg.reply,{timeout:e.timeout},n);i.request(c);try{i.publish(this.msg.reply,Zr,{reply:`${i.muxSubscriptions.baseInbox}${c.token}`})}catch(a){c.cancel(a)}try{await Promise.race([c.timer,c.deferred]),t.resolve(!0)}catch(a){c.cancel(a),t.reject(a)}}else t.resolve(!1);return t}ack(){this.doAck(Zr)}nak(e){let t=Za;e&&(t=$r().encode(`-NAK ${JSON.stringify({delay:V(e)})}`)),this.doAck(t)}working(){this.doAck(Ct)}next(e,t={batch:1}){const s={};s.batch=t.batch||1,s.no_wait=t.no_wait||!1,t.expires&&t.expires>0&&(s.expires=V(t.expires));const i=$e().encode(s),n=kt.concat(Qa,to,i),c=e?{reply:e}:void 0;this.msg.respond(n,c)}term(e=""){let t=eo;e?.length>0&&(t=$r().encode(`+TERM ${e}`)),this.doAck(t)}json(){return this.msg.json()}string(){return this.msg.string()}}const _o="1.30.3",vo="nats.ws";class So{version;lang;closeError;connected;done;socket;options;socketClosed;encrypted;peeked;yields;signal;closedNotification;constructor(){this.version=_o,this.lang=vo,this.connected=!1,this.done=!1,this.socketClosed=!1,this.encrypted=!1,this.peeked=!1,this.yields=[],this.signal=W(),this.closedNotification=W()}async connect(e,t){const s=W();if(t.tls)return s.reject(new j("tls",E.InvalidOption)),s;this.options=t;const i=e.src;if(t.wsFactory){const{socket:n,encrypted:c}=await t.wsFactory(e.src,t);this.socket=n,this.encrypted=c}else this.encrypted=i.indexOf("wss://")===0,this.socket=new WebSocket(i);return this.socket.binaryType="arraybuffer",this.socket.onopen=()=>{this.isDiscarded()},this.socket.onmessage=n=>{if(this.isDiscarded())return;if(this.yields.push(new Uint8Array(n.data)),this.peeked){this.signal.resolve();return}const c=kt.concat(...this.yields),a=Gn(c);if(a!==""){const d=Ma.exec(a);if(!d){t.debug&&console.error("!!!",is(c)),s.reject(new Error("unexpected response from server"));return}try{const m=JSON.parse(d[1]);Ca(m,this.options),this.peeked=!0,this.connected=!0,this.signal.resolve(),s.resolve()}catch(m){s.reject(m);return}}},this.socket.onclose=n=>{if(this.isDiscarded())return;this.socketClosed=!0;let c;this.done||(n.wasClean||(c=new Error(n.reason)),this._closed(c))},this.socket.onerror=n=>{if(this.isDiscarded())return;const c=n,a=new j(c.message,E.Unknown,new Error(c.error));s.reject(a)},s}disconnect(){this._closed(void 0,!0)}async _closed(e,t=!0){if(!this.isDiscarded()&&this.connected&&!this.done){if(this.closeError=e,!e)for(;!this.socketClosed&&this.socket.bufferedAmount>0;)await Et(100);this.done=!0;try{this.socket.close(e?1002:1e3,e?e.message:void 0)}catch{}t&&this.closedNotification.resolve(e)}}get isClosed(){return this.done}[Symbol.asyncIterator](){return this.iterate()}async*iterate(){for(;;){if(this.isDiscarded())return;this.yields.length===0&&await this.signal;const e=this.yields;this.yields=[];for(let t=0;t<e.length;t++)this.options.debug&&console.info(`> ${is(e[t])}`),yield e[t];if(this.done)break;this.yields.length===0&&(e.length=0,this.yields=e,this.signal=W())}}isEncrypted(){return this.connected&&this.encrypted}send(e){if(!this.isDiscarded())try{this.socket.send(e.buffer),this.options.debug&&console.info(`< ${is(e)}`);return}catch(t){this.options.debug&&console.error(`!!! ${is(e)}: ${t}`)}}close(e){return this._closed(e,!1)}closed(){return this.closedNotification}isDiscarded(){return this.done?(this.discard(),!0):!1}discard(){this.done=!0;try{this.socket?.close()}catch{}}}function ko(r,e){/^(.*:\/\/)(.*)/.test(r)||(typeof e=="boolean"?r=`${e===!0?"https":"http"}://${r}`:r=`https://${r}`);let s=new URL(r);const i=s.protocol.toLowerCase();i==="ws:"&&(e=!1),i==="wss:"&&(e=!0),i!=="https:"&&i!=="http"&&(r=r.replace(/^(.*:\/\/)(.*)/gm,"$2"),s=new URL(`http://${r}`));let n,c;const a=s.hostname,d=s.pathname,m=s.search||"";switch(i){case"http:":case"ws:":case"nats:":c=s.port||"80",n="ws:";break;case"https:":case"wss:":case"tls:":c=s.port||"443",n="wss:";break;default:c=s.port||e===!0?"443":"80",n=e===!0?"wss:":"ws:";break}return`${n}//${a}:${c}${d}${m}`}function Ii(r={}){return Dn({defaultPort:443,urlParseFn:ko,factory:()=>new So}),pr.connect(r)}function Eo(r){try{const e=JSON.parse(r);if(e?.kind==="session.frame"&&e.frame==="token"&&typeof e.text=="string")return e.text}catch{}return null}async function Ai(r){const e=await fetch("/session/viewer",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({sessionId:r})});if(!e.ok)throw new Error(`viewer mint failed: ${e.status}`);return e.json()}async function Io(r,e){const t=await Ai(r),s=location.protocol==="https:"?"wss":"ws",i=await Ii({servers:`${s}://${location.host}/session/ws?t=${t.wsTicket}`,authenticator:yi(t.jwt)});return(async()=>{const n=new TextDecoder;for await(const c of i.subscribe(t.deliverSubject)){const a=Eo(n.decode(c.data));a!==null&&e(a)}})(),i}async function Ao(r){const e=await Ai(r),t=location.protocol==="https:"?"wss":"ws",s=await Ii({servers:`${t}://${location.host}/session/ws?t=${e.wsTicket}`,authenticator:yi(e.jwt)});return{async request(i){const n=await s.request("tb.app.browser.command",JSON.stringify(i),{timeout:5e3});return ar(n.data)},async watch(i,n){const c=Po(i,e.stateSubject),a=await ti(s,c),d=Ro(a,e.stateSubject),m=s.subscribe(d);let x=!0;return(async()=>{for await(const v of m)x&&n(ar(v.data))})(),await ti(s,{...c,commandId:`${c.commandId}-attach`}),()=>{x=!1,m.unsubscribe()}},close(){return s.close()}}}function Po(r,e){const t=Pi(r.payload)?{...r.payload,delivery:e}:{delivery:e};return{...r,payload:t}}async function ti(r,e){const t=await r.request("tb.app.browser.command",JSON.stringify(e),{timeout:5e3});return ar(t.data)}function Ro(r,e){if(!Pi(r)||r.status!=="accepted"||typeof r.deliverySubject!="string")throw new Error("state watch denied");if(!r.deliverySubject.startsWith(`${e}.`))throw new Error("state watch escaped viewer grant");return r.deliverySubject}function ar(r){const e=new TextDecoder().decode(r);return JSON.parse(e)}function Pi(r){return typeof r=="object"&&r!==null}const Ri=document.querySelector("#app");if(!Ri)throw new Error("missing app root");const Me=Ri,jo=new URLSearchParams(location.search),Ge=ye("tb_app"),Lt=ye("tb_participant"),Se=Ge!==""&&Lt!=="",ji=ye("tb_board")==="1",Xe=ye("tb_chess")==="1",Oi=ye("tb_board_no")||"board-001",Oo=ye("tb_name"),Is=ye("tb_type")==="1",Ci=ye("tb_race_no")||"race-001",Co=ye("tb_alias"),mr=ye("tb_visual"),Be=mr!=="",No=ye("tb_choice")||"diagram-a",cs=Bo(mr)||"artifact-001",or=ye("tb_session")||(Se?"demo-001":Be?"visual-001":"session-001"),Kt=ye("tb_state")||(Se?`apps.${Ge}.state.${Xe?`chess.${Oi}`:Is?`typerace.${Ci}`:ji?"board":"browser"}`:""),si=Number.parseInt(ye("tb_auto")||"0",10),ri=Number.parseInt(ye("tb_interval_ms")||"25",10),ft=en("generated artifact proof"),Mo=Xe||Is,he={sandbox:ft.sandbox,accepted:[],denied:[],dispatched:[],state:{delivery:"",events:0,lastRevision:0,errors:[]}};window.__tinkabotProof=he;Me.innerHTML=Mo?`
    <main class="${Xe?"chess-shell":"app-shell"}">
      <iframe
        data-proof="frame"
        title="${Xe?"Chess":"Typeracing"}"
        sandbox="${ft.sandbox}"
        referrerpolicy="${ft.referrerPolicy}"
      ></iframe>
    </main>
  `:`
    <main class="shell">
      <section>
        <p class="eyebrow">trusted shell</p>
        <h1>Tinkabot</h1>
        <p>Opaque generated content is isolated behind a leased message channel.</p>
        <dl>
          <div><dt>Sandbox</dt><dd data-proof="sandbox"></dd></div>
          <div><dt>Accepted</dt><dd data-proof="accepted">0</dd></div>
          <div><dt>Dispatched</dt><dd data-proof="dispatched">0</dd></div>
          <div><dt>Denied</dt><dd data-proof="denied">0</dd></div>
          <div><dt>Cookie Probe</dt><dd data-proof="cookie">pending</dd></div>
          <div><dt>Participant</dt><dd data-proof="participant">none</dd></div>
        </dl>
      </section>
      <section class="observe">
        <p class="eyebrow">session observation</p>
        <label>Session <input data-obs="sid" /></label>
        <button data-obs="go">Observe</button>
        <pre data-obs="log"></pre>
      </section>
      <iframe
        data-proof="frame"
        title="${ft.title}"
        sandbox="${ft.sandbox}"
        referrerpolicy="${ft.referrerPolicy}"
      ></iframe>
    </main>
  `;const Ni=Me.querySelector("iframe");if(!Ni)throw new Error("missing generated frame");const it=Ni,ve=sn({frameId:"frame-001",sessionId:or,capabilityId:Se?`cap-${Ge}-${Lt}`:Be?`cap-${cs}-visual`:"cap-001",artifactId:Se?`artifact-${Ge}-${Lt}`:Be?cs:"artifact-001",artifactRevision:"artifact.rev.7",schemaRevision:"schema.rev.1",appId:Se?Ge:void 0,participantId:Se?Lt:void 0,commands:Se?["participant_read","participant_action"]:Be?["item_submit"]:["select_artifact"],sessions:Se||Be?[or]:[],chain:{chainId:Se?`chain-${Ge}`:Be?`chain-${cs}`:"chain-001",rootId:Se?`root-${Ge}`:Be?`root-${cs}`:"root-001",hop:0,maxHops:5}});let Ut,bt,Bt;window.addEventListener("message",r=>{if(r.source!==it.contentWindow)return;he.ready={origin:r.origin,source:!0},nt();const e=r.data;if(e?.type==="content.ready"){it.contentWindow?.postMessage({type:"tinkabot.lease",lease:ve,demo:{stateKey:Kt,chess:Xe,boardNo:Oi,playerName:Oo,typeRace:Is,raceNo:Ci,alias:Co,board:ji,autoActions:Number.isFinite(si)?Math.max(0,si):0,intervalMs:Number.isFinite(ri)?Math.max(1,ri):25,visualKey:mr,choice:No}},"*"),$o();return}if(e?.type==="content.probe"){he.probe={cookie:e.cookie,storage:e.storage},nt();return}try{const t=nn(ve,r.source,it.contentWindow,e);he.accepted.push(t),(Se||Be)&&To(t)}catch(t){he.denied.push(t instanceof Error?t.message:String(t))}nt()});it.src=un();const dt=Me.querySelector('[data-obs="log"]'),Ks=Me.querySelector('[data-obs="sid"]'),ii=Me.querySelector('[data-obs="go"]');dt&&Ks&&ii&&(Ks.value=or,ii.addEventListener("click",()=>{dt.textContent="",bt&&bt.then(r=>r.close()).catch(()=>{}),bt=Io(Ks.value,r=>{dt.textContent+=r,dt.scrollTop=dt.scrollHeight}),bt.catch(r=>{dt.textContent=`observe failed: ${r instanceof Error?r.message:String(r)}`})}));window.addEventListener("beforeunload",()=>{Bt&&Bt.then(r=>r()).catch(()=>{}),Ut&&Ut.then(r=>r.close()).catch(()=>{}),bt&&bt.then(r=>r.close()).catch(()=>{})});nt();async function To(r){const e=performance.now();let t;try{Lo(r),t=await(await Mi()).request(r),Ke(t);const s=Uo(r,t,performance.now()-e);it.contentWindow?.postMessage({type:"tinkabot.command.result",commandId:r.commandId,response:s.response},"*")}catch(s){const i=s instanceof Error?s.message:String(s);he.dispatched.push({command:r.command,commandId:r.commandId,status:"failed",reason:i,latencyMs:Math.round(performance.now()-e)}),it.contentWindow?.postMessage({type:"tinkabot.command.result",commandId:r.commandId,error:i},"*")}nt()}function Mi(){return Ut??=Ao(ve.sessionId).catch(r=>{throw Ut=void 0,r}),Ut}function $o(){!Se||Kt===""||Bt||(Bt=Mi().then(r=>r.watch(qo(),Fo)).catch(r=>(Bt=void 0,he.state.errors.push(r instanceof Error?r.message:String(r)),nt(),()=>{})))}function qo(){return{kind:"browser.command_intent",type:"content.intent",command:"participant_watch",commandId:`watch-${ve.frameId}-${Date.now()}`,expectedRevision:ve.artifactRevision,payload:{key:Kt},context:{sessionId:ve.sessionId,capabilityId:ve.capabilityId,artifactId:ve.artifactId,artifactRevision:ve.artifactRevision,frameId:ve.frameId,appId:ve.appId,participantId:ve.participantId,chain:ve.chain}}}function Fo(r){try{Ke(r),he.state.delivery=r.source,he.state.events+=1,he.state.lastRevision=r.revision,it.contentWindow?.postMessage({type:"tinkabot.state",source:r.source,item:{key:r.key,status:r.status,value:r.value,revision:r.revision,observedAt:r.observedAt}},"*")}catch(e){he.state.errors.push(e instanceof Error?e.message:String(e))}nt()}function Lo(r){if(!Xe&&!Is)return;const e=ls(r.payload);if(r.command==="participant_read"&&e.key!==Kt)throw new Error(`${Xe?"chess board":"typerace"} read denied`);if(r.command==="participant_action"&&e.stateKey!==Kt)throw new Error(`${Xe?"chess board":"typerace"} action denied`)}function Uo(r,e,t){const s=ls(e),i=ls(s.item),n=ls(r.payload),c=typeof n.key=="string"?n.key:n.stateKey,a={command:r.command,commandId:r.commandId,status:typeof s.status=="string"?s.status:"unknown",reason:typeof s.reason=="string"?s.reason:void 0,latencyMs:Math.round(t),itemKey:typeof i.key=="string"?i.key:void 0,payloadKey:typeof c=="string"?c:void 0,response:e};return he.dispatched.push(a),a}function nt(){const r=Me.querySelector('[data-proof="sandbox"]');r&&(r.textContent=he.sandbox,Me.querySelector('[data-proof="accepted"]').textContent=String(he.accepted.length),Me.querySelector('[data-proof="dispatched"]').textContent=String(he.dispatched.length),Me.querySelector('[data-proof="denied"]').textContent=String(he.denied.length),Me.querySelector('[data-proof="cookie"]').textContent=he.probe?.cookie||"empty",Me.querySelector('[data-proof="participant"]').textContent=Se?`${Ge}/${Lt}`:Be?"visual":"none")}function ye(r){return jo.get(r)?.trim()??""}function Bo(r){return/^artifacts\.([A-Za-z0-9_-]+)\.results\./.exec(r)?.[1]??""}function ls(r){return typeof r=="object"&&r!==null?r:{}}});export default Do();
