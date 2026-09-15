"""Offline migration invariants; no production data or database access."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('report_unify', Path(__file__).with_name('report-unify.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class MigrationTests(unittest.TestCase):
    def source(self):
        p = Path(__file__).parents[1] / 'api/data/v1/report/testdata/signal-publication-request.json'
        return json.loads(p.read_text())['report']

    def test_preserves_source_and_every_original_assessment(self):
        source = self.source()
        snapshot = copy.deepcopy(source)
        result, bindings = module.convert(source)
        self.assertEqual(source, snapshot)
        self.assertTrue(bindings)
        for kind in ('geopolitical_stories', 'macroeconomic_stories', 'concept_analyses', 'industry_chain_analyses'):
            for before, after in zip(source[kind], result[kind]):
                old = before['detail']; new = after['detail']['reasonings']
                self.assertEqual(len(new), len(old['macro_impacts']) + len(old['industry_chains']))
                for original, converted in zip(old['macro_impacts'] + old['industry_chains'], new):
                    self.assertEqual(original['assessment'], converted['assessment'])
                    if 'affected_nodes' in original:
                        for node in original['affected_nodes']:
                            self.assertIn(node, converted['affected_assets'])
                        self.assertEqual(original['graph'], converted['graph'])
                        self.assertEqual(original['reasoning_summary'], converted['reasoning_summary'])
                for ref in after['summary']['affected_refs']:
                    reasoning = next(r for r in new if r['local_key'] == ref['reasoning_local_key'])
                    self.assertTrue(any(a['local_key'] == ref['local_key'] for a in reasoning['affected_assets']))
                self.assertEqual(old.get('companies'), after['detail'].get('companies'))
        self.assertEqual(source['company_analyses'], result['company_analyses'])

    def test_rejects_dangling_reference_instead_of_guessing(self):
        source = self.source()
        source['geopolitical_stories'][0]['summary']['affected_refs'][0]['local_key'] = 'missing'
        with self.assertRaisesRegex(ValueError, 'Dangling'):
            module.convert(source)

    def test_rejects_a_second_conversion(self):
        result, _ = module.convert(self.source())
        with self.assertRaises(ValueError):
            module.convert(result)


if __name__ == '__main__':
    unittest.main()
