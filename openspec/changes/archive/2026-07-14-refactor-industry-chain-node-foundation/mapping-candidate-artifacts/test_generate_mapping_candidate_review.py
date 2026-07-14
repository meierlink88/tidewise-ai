import json
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
ARTIFACTS = ROOT / "mapping-candidate-artifacts"
WORKBOOK = Path("/Users/meierlink/.codex/visualizations/2026/07/12/019f5477-445d-75d3-acf2-61a4fdd5b1d4/outputs/产业链节点候选-稳定节点宽口径筛选与合并.xlsx")
MANIFEST = ROOT / "final-seed-candidate-artifacts" / "final-seed-candidate-manifest.json"
GENERATOR = ARTIFACTS / "generate_mapping_candidate_review.py"


class MappingCandidateReviewTest(unittest.TestCase):
    def generate(self, output_dir):
        subprocess.run(
            ["python3", str(GENERATOR), "--workbook", str(WORKBOOK), "--manifest", str(MANIFEST), "--output-dir", str(output_dir)],
            check=True,
        )
        return json.loads((output_dir / "external-identifier-mapping-candidates.json").read_text()), json.loads((output_dir / "external-identifier-mapping-validation.json").read_text())

    def test_expands_user_verified_multi_taxonomy_codes_without_raw_identity_conflicts(self):
        with tempfile.TemporaryDirectory() as first, tempfile.TemporaryDirectory() as second:
            candidates, report = self.generate(Path(first))
            repeated, repeated_report = self.generate(Path(second))

        self.assertEqual(1169, report["counts"]["candidates"])
        self.assertEqual(818, report["counts"]["eastmoney"])
        self.assertEqual(351, report["counts"]["ths"])
        self.assertEqual(241, report["counts"]["dual_source_nodes"])
        self.assertEqual(13, report["counts"]["multi_taxonomy_source_codes"])
        self.assertEqual(1143, report["counts"]["single_taxonomy_source_codes"])
        self.assertTrue(report["validation"]["ready_for_write"])
        self.assertEqual([], report["validation"]["external_identity_duplicates"])
        self.assertEqual([], report["validation"]["deterministic_id_duplicates"])
        self.assertEqual(candidates, repeated)
        self.assertEqual(report, repeated_report)

        pairs = {(item["source_system"], item["external_code"], item["source_taxonomy_type"]) for item in candidates["mappings"]}
        self.assertEqual(1169, len(pairs))
        self.assertEqual(1169, len({item["id"] for item in candidates["mappings"]}))
        self.assertTrue(all(item["entity_id"] for item in candidates["mappings"]))
        self.assertEqual(0, report["validation"]["orphan_expected"])

        double_taxonomy_codes = {
            (item["source_system"], item["external_code"])
            for item in candidates["mappings"]
            if item["source_taxonomy_type"] in {"industry_sector", "concept_sector"}
        }
        self.assertEqual(13, len(report["review_lists"]["multi_taxonomy_source_codes"]))
        self.assertTrue(all(len(entry["taxonomy_types"]) == 2 for entry in report["review_lists"]["multi_taxonomy_source_codes"]))
        self.assertGreaterEqual(len(double_taxonomy_codes), 13)


if __name__ == "__main__":
    unittest.main()
