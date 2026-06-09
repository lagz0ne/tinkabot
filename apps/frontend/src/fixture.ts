export function generatedUrl() {
  return URL.createObjectURL(new Blob([generatedHtml()], { type: "text/html" }));
}

function generatedHtml() {
  return `<!doctype html>
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
    </script>
  </body>
</html>`;
}
