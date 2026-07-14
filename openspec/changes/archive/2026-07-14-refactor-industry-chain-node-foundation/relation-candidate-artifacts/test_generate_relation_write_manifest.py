import importlib.util
import json
import pathlib
import unittest


HERE = pathlib.Path(__file__).resolve().parent
SCRIPT = HERE / "generate_relation_write_manifest.py"


class RelationWriteManifestTest(unittest.TestCase):
    def test_builds_only_the_96_approved_semantic_relations(self):
        spec = importlib.util.spec_from_file_location("relation_write_manifest", SCRIPT)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)

        manifest, validation = module.build()

        self.assertEqual(len(manifest["relations"]), 96)
        self.assertEqual(manifest["physical_constraints"], [])
        self.assertEqual(validation["by_relation_type"], {
            "is_subcategory_of": 95,
            "is_component_of": 1,
            "input_to": 0,
            "depends_on": 0,
        })
        self.assertEqual(validation["blocked_included"], 0)
        self.assertEqual(validation["rejected_included"], 0)
        self.assertEqual(validation["self_loops"], 0)
        self.assertEqual(validation["tuple_duplicates"], 0)
        self.assertEqual(validation["id_duplicates"], 0)
        for relation in manifest["relations"]:
            self.assertTrue(relation["verified_at"])
            self.assertIn("final-seed-candidate-artifacts/node-profile-seed-manifest.json", relation["provenance"])
            self.assertIn("9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e", relation["provenance"])
            self.assertIn("ffb243e", relation["provenance"])
            self.assertIn("main-serenity", relation["provenance"])

        first_manifest = json.dumps(manifest, ensure_ascii=False, sort_keys=True)
        second_manifest = json.dumps(module.build()[0], ensure_ascii=False, sort_keys=True)
        self.assertEqual(first_manifest, second_manifest)


if __name__ == "__main__":
    unittest.main()
