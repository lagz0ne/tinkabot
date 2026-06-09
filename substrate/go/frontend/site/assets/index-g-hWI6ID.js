(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const t of document.querySelectorAll('link[rel="modulepreload"]'))o(t);new MutationObserver(t=>{for(const i of t)if(i.type==="childList")for(const c of i.addedNodes)c.tagName==="LINK"&&c.rel==="modulepreload"&&o(c)}).observe(document,{childList:!0,subtree:!0});function r(t){const i={};return t.integrity&&(i.integrity=t.integrity),t.referrerPolicy&&(i.referrerPolicy=t.referrerPolicy),t.crossOrigin==="use-credentials"?i.credentials="include":t.crossOrigin==="anonymous"?i.credentials="omit":i.credentials="same-origin",i}function o(t){if(t.ep)return;t.ep=!0;const i=r(t);fetch(t.href,i)}})();class b extends Error{constructor(n,r,o={}){super(r),this.kind=n,this.details=o}kind;details;layer="FrontendIsolation"}const w="allow-scripts",v=new Set(["allow","allowresponses","credential","credentials","deny","headers","nats","permission","permissions","publish","reply","replysubject","subject","subjects","subscribe","token","tokens"]);function I(e="generated artifact"){return{title:e,sandbox:R(w),referrerPolicy:"no-referrer"}}function R(e){const n=new Set(e.split(/\s+/).filter(Boolean));if(n.size!==1||!n.has("allow-scripts"))throw a("FrameSandboxDenied","Generated content requires script-only sandbox",{value:e});return[...n].sort().join(" ")}function x(e){return{...e,nonce:e.nonce??crypto.randomUUID()}}function S(e,n,r,o){if(n!==r)throw a("FrameLeaseDenied","Message source does not match leased frame",{frameId:e.frameId});d(o);const t=k(o);if(t.nonce!==e.nonce)throw a("FrameLeaseDenied","Message nonce does not match frame lease",{frameId:e.frameId});if(t.frameId!==e.frameId)throw a("FrameLeaseDenied","Message frame id does not match lease",{frameId:e.frameId,actual:t.frameId});if(t.artifactRevision!==e.artifactRevision)throw a("FrameLeaseDenied","Message artifact revision is stale",{expected:e.artifactRevision,actual:t.artifactRevision});if(t.expectedRevision!==e.artifactRevision)throw a("FrameLeaseDenied","Message expected revision is stale",{expected:e.artifactRevision,actual:t.expectedRevision});if(t.schemaRevision!==e.schemaRevision)throw a("FrameLeaseDenied","Message schema revision is stale",{expected:e.schemaRevision,actual:t.schemaRevision});if(!e.commands.includes(t.command))throw a("FrameCapabilityDenied","Command is not allowed for frame lease",{command:t.command});return{kind:"browser.command_intent",type:"content.intent",command:t.command,commandId:t.commandId,expectedRevision:t.expectedRevision,payload:t.payload,context:{sessionId:e.sessionId,capabilityId:e.capabilityId,artifactId:e.artifactId,artifactRevision:e.artifactRevision,frameId:e.frameId,chain:e.chain}}}function d(e,n=[],r=new WeakSet){if(h(e)&&!r.has(e)){if(r.add(e),Array.isArray(e)){e.forEach((o,t)=>d(o,[...n,String(t)],r));return}if(e instanceof Map){let o=0;for(const[t,i]of e){const c=typeof t=="string"?t:String(o);u(c,[...n,c]),typeof t!="string"&&d(t,[...n,`$key${o}`],r),d(i,[...n,c],r),o+=1}return}if(e instanceof Set){let o=0;for(const t of e)d(t,[...n,String(o)],r),o+=1;return}for(const[o,t]of Object.entries(e))u(o,[...n,o]),d(t,[...n,o],r)}}function k(e){if(!M(e))throw a("FrameMessageInvalid","Message must be an object");if(e.type!=="content.intent")throw a("FrameMessageInvalid","Message type is not supported",{type:e.type});for(const n of["command","commandId","expectedRevision","nonce","frameId","artifactRevision","schemaRevision"])if(typeof e[n]!="string"||e[n].length===0)throw a("FrameMessageInvalid","Message field is required",{field:n});return e}function a(e,n,r={}){return new b(e,n,r)}function u(e,n){const r=e.toLowerCase().replace(/[-_]/g,"");if(v.has(r))throw a("FrameCapabilityDenied","Generated content cannot send raw authority",{path:n.join(".")})}function h(e){return typeof e=="object"&&e!==null}function M(e){return h(e)}function L(){return URL.createObjectURL(new Blob([F()],{type:"text/html"}))}function F(){return`<!doctype html>
<html>
  <body>
    <main id="generated">Generated content</main>
    <script>
      const read = (fn) => {
        try { return String(fn()); } catch (err) { return "denied:" + err.name; }
      };

      window.addEventListener("message", (event) => {
        if (event.data?.type !== "tinkabot.lease") return;
        const lease = event.data.lease;

        parent.postMessage({
          type: "content.probe",
          cookie: read(() => document.cookie),
          storage: read(() => localStorage.length),
        }, "*");

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
      });

      parent.postMessage({ type: "content.ready" }, "*");
    <\/script>
  </body>
</html>`}const g=document.querySelector("#app");if(!g)throw new Error("missing app root");const f=g,p=I("generated artifact proof"),s={sandbox:p.sandbox,accepted:[],denied:[]};window.__tinkabotProof=s;f.innerHTML=`
  <main class="shell">
    <section>
      <p class="eyebrow">trusted shell</p>
      <h1>Tinkabot</h1>
      <p>Opaque generated content is isolated behind a leased message channel.</p>
      <dl>
        <div><dt>Sandbox</dt><dd data-proof="sandbox"></dd></div>
        <div><dt>Accepted</dt><dd data-proof="accepted">0</dd></div>
        <div><dt>Denied</dt><dd data-proof="denied">0</dd></div>
        <div><dt>Cookie Probe</dt><dd data-proof="cookie">pending</dd></div>
      </dl>
    </section>
    <iframe
      data-proof="frame"
      title="${p.title}"
      sandbox="${p.sandbox}"
      referrerpolicy="${p.referrerPolicy}"
    ></iframe>
  </main>
`;const m=f.querySelector("iframe");if(!m)throw new Error("missing generated frame");const y=x({frameId:"frame-001",sessionId:"session-001",capabilityId:"cap-001",artifactId:"artifact-001",artifactRevision:"artifact.rev.7",schemaRevision:"schema.rev.1",commands:["select_artifact"],chain:{chainId:"chain-001",rootId:"root-001",hop:0,maxHops:5}});window.addEventListener("message",e=>{if(e.source!==m.contentWindow)return;s.ready={origin:e.origin,source:!0},l();const n=e.data;if(n?.type==="content.ready"){m.contentWindow?.postMessage({type:"tinkabot.lease",lease:y},"*");return}if(n?.type==="content.probe"){s.probe={cookie:n.cookie,storage:n.storage},l();return}try{const r=S(y,e.source,m.contentWindow,n);s.accepted.push(r)}catch(r){s.denied.push(r instanceof Error?r.message:String(r))}l()});m.src=L();l();function l(){f.querySelector('[data-proof="sandbox"]').textContent=s.sandbox,f.querySelector('[data-proof="accepted"]').textContent=String(s.accepted.length),f.querySelector('[data-proof="denied"]').textContent=String(s.denied.length),f.querySelector('[data-proof="cookie"]').textContent=s.probe?.cookie||"empty"}
