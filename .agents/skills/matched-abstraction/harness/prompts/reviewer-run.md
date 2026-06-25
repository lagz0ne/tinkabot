# Reviewer Run Prompt

Evaluate one generated matched-abstraction output. Use the supplied rubric, topic prompt, topic notes, and review lenses. Do not reward information that appears only as implied intent if it is absent from the generated Plan.

Return:

1. Score table with each rubric category, score, and one-sentence evidence.
2. Any score caps that apply.
3. Missing Plan commitments that would likely become missing Task delivery.
4. Layer-boundary violations.
5. Artifact set gaps, including missing depth, timing, derivation hints, matching hints, use-now artifacts that are named but not rendered, overlap, or rabbit-hole artifacts.
6. One concise recommendation for improving the skill or rerunning the topic.

Judge the output as a Plan-ending design artifact. Do not expect implementation, code, command output, or completed Task evidence.
