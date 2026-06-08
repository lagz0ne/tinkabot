import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]
SCRIPT = PROJECT_ROOT / ".codex/skills/matched-abstraction-thinking/scripts/validate_layers.py"


def write_doc(root: Path, relative: str, body: str) -> None:
    path = root / relative
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(textwrap.dedent(body).strip() + "\n", encoding="utf-8")


def write_valid_docs(root: Path) -> None:
    write_doc(
        root,
        "approach/charter.md",
        """
        ---
        layer: approach
        topic: baseline
        references: []
        ---
        # Approach

        ## Scope
        Defines the top-level thinking boundary.

        ## Layer Contract
        Approach constrains the valid design space.

        ## Plan-Readiness Gate
        The Plan layer may proceed when invariants and non-goals are explicit.
        """,
    )
    write_doc(
        root,
        "plan/orchestration.md",
        """
        ---
        layer: plan
        topic: baseline
        references:
          - ../approach/charter.md
        ---
        # Plan

        ## Consumed Approach
        Uses the approved approach charter.

        ## Decomposition
        Splits work into protected layer agents.

        ## Verification Strategy
        Task agents must return concrete evidence.
        """,
    )
    write_doc(
        root,
        "task/baseline.md",
        """
        ---
        layer: task
        topic: baseline
        references:
          - ../plan/orchestration.md
          - ../approach/charter.md
        ---
        # Task

        ## Acceptance Contract
        The task has observable pass/fail criteria.

        ## RED Artifact
        The task captures failing or pre-change evidence first.

        ## Verification Evidence
        - `python3 -m unittest tests/test_validate_layers.py` -> `OK`.
        """,
    )


class ValidateLayersTest(unittest.TestCase):
    def run_validator(self, root: Path) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPT), str(root)],
            cwd=PROJECT_ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            check=False,
        )

    def test_accepts_valid_layer_docs(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)

            result = self.run_validator(root)

            self.assertEqual(result.returncode, 0, result.stdout)
            self.assertIn("Layer validation passed", result.stdout)

    def test_rejects_approach_task_checklists(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            approach = root / "approach/charter.md"
            approach.write_text(approach.read_text(encoding="utf-8") + "\n- [ ] implement concrete task\n", encoding="utf-8")

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("approach docs must not contain task checklists", result.stdout)

    def test_rejects_missing_topic_frontmatter(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            approach = root / "approach/charter.md"
            approach.write_text(approach.read_text(encoding="utf-8").replace("topic: baseline\n", ""), encoding="utf-8")

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("missing required frontmatter key: topic", result.stdout)

    def test_rejects_missing_references_frontmatter(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            approach = root / "approach/charter.md"
            approach.write_text(approach.read_text(encoding="utf-8").replace("references: []\n", ""), encoding="utf-8")

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("missing required frontmatter key: references", result.stdout)

    def test_rejects_plan_reference_to_task_authority(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            plan = root / "plan/orchestration.md"
            text = plan.read_text(encoding="utf-8").replace(
                "  - ../approach/charter.md",
                "  - ../approach/charter.md\n  - ../task/baseline.md",
            )
            plan.write_text(text, encoding="utf-8")

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("plan docs may not reference task docs as authority", result.stdout)

    def test_rejects_task_without_verification_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            task = root / "task/baseline.md"
            task.write_text(
                task.read_text(encoding="utf-8").replace(
                    "\n## Verification Evidence\n- `python3 -m unittest tests/test_validate_layers.py` -> `OK`.\n",
                    "\n",
                ),
                encoding="utf-8",
            )

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("task/baseline.md: missing required section: ## Verification Evidence", result.stdout)

    def test_rejects_second_task_without_verification_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            write_doc(
                root,
                "task/second.md",
                """
                ---
                layer: task
                topic: second
                references:
                  - ../plan/orchestration.md
                ---
                # Second Task

                ## Acceptance Contract
                It has an observable boundary.

                ## RED Artifact
                It has pre-change proof.
                """,
            )

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("task/second.md: missing required section: ## Verification Evidence", result.stdout)

    def test_rejects_promissory_task_verification_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            task = root / "task/baseline.md"
            task.write_text(
                task.read_text(encoding="utf-8").replace(
                    "- `python3 -m unittest tests/test_validate_layers.py` -> `OK`.",
                    "Evidence will be recorded later.",
                ),
                encoding="utf-8",
            )

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("task/baseline.md: verification evidence must contain concrete command/result evidence", result.stdout)

    def test_validates_nested_layer_docs(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            write_doc(
                root,
                "task/nested/missing.md",
                """
                ---
                layer: task
                topic: nested
                references:
                  - ../../plan/orchestration.md
                ---
                # Nested Task

                ## Acceptance Contract
                It has an observable boundary.

                ## RED Artifact
                It has pre-change proof.
                """,
            )

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("task/nested/missing.md: missing required section: ## Verification Evidence", result.stdout)

    def test_rejects_markdown_outside_layer_directories(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_valid_docs(root)
            write_doc(
                root,
                "overview.md",
                """
                ---
                layer: approach
                topic: misplaced
                references: []
                ---
                # Misplaced

                ## Scope
                Misplaced docs are invalid.
                """,
            )

            result = self.run_validator(root)

            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("overview.md: markdown docs must live under approach/, plan/, or task/", result.stdout)


if __name__ == "__main__":
    unittest.main()
