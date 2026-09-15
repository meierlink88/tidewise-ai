#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Offline, explicit v4/v5 -> v6 publication conversion. Never connects to a database."""
import argparse
import copy
import hashlib
import json
from pathlib import Path


def convert(report):
    if report.get('schema_version') not in ('report-publication/v4', 'report-publication/v5'):
        raise ValueError('Expected an explicit v4/v5 source report')
    result = copy.deepcopy(report)
    result['schema_version'] = 'report-publication/v6'
    bindings = []
    for kind in ('geopolitical_stories', 'macroeconomic_stories', 'concept_analyses', 'industry_chain_analyses'):
        for unit in result.get(kind, []):
            detail = unit['detail']
            reasonings, refmap = [], {}
            for macro in detail.pop('macro_impacts', []):
                reasoning = copy.deepcopy(macro)
                reasoning['local_key'] = macro['local_key'] + '-reasoning'
                reasoning['title'] = reasoning.pop('name')
                reasoning['reasoning_summary'] = {
                    'logic': macro['assessment']['transmission_logic'],
                    'objections': reasoning.pop('objections')
                }
                asset = copy.deepcopy(macro)
                asset['node_local_key'] = ''
                reasoning['affected_assets'] = [asset]
                reasoning['reasoning_blocks'] = []
                reasonings.append(reasoning)
                refmap[('macroeconomic_story', None, macro['local_key'])] = {
                    'reasoning_local_key': reasoning['local_key'], 'local_key': asset['local_key']}
            for chain in detail.pop('industry_chains', []):
                reasoning = copy.deepcopy(chain)
                reasoning['title'] = reasoning.pop('name')
                reasoning['affected_assets'] = reasoning.pop('affected_nodes')
                reasoning['reasoning_blocks'] = []
                for node in reasoning['affected_assets']:
                    refmap[('industry_chain_node', chain['local_key'], node['local_key'])] = {
                        'reasoning_local_key': chain['local_key'], 'local_key': node['local_key']}
                # Preserve a distinct whole-chain homepage judgment, not an arbitrary child node.
                if any(r['target_type'] == 'industry_chain' and r['local_key'] == chain['local_key']
                       for r in unit['summary']['affected_refs']):
                    asset = {k: copy.deepcopy(v) for k, v in chain.items() if k in
                             ('source_id', 'name', 'assessment', 'judgment_origin', 'reasoning_sources', 'variable_signals')}
                    asset['local_key'] = chain['local_key'] + '-asset'
                    asset['node_local_key'] = ''
                    asset['objections'] = copy.deepcopy(chain['reasoning_summary']['objections'])
                    if any(x['local_key'] == asset['local_key'] for x in reasoning['affected_assets']):
                        raise ValueError('Generated whole-chain asset key collision')
                    reasoning['affected_assets'].insert(0, asset)
                    refmap[('industry_chain', None, chain['local_key'])] = {
                        'reasoning_local_key': chain['local_key'], 'local_key': asset['local_key']}
                reasonings.append(reasoning)
            if not reasonings:
                raise ValueError(f'No reasoning content in {kind}/{unit["local_key"]}')
            detail['reasonings'] = reasonings
            refs = []
            for ref in unit['summary']['affected_refs']:
                old = (ref['target_type'], ref.get('chain_local_key'), ref['local_key'])
                if old not in refmap:
                    raise ValueError(f'Dangling source reference: {old}')
                refs.append(copy.deepcopy(refmap[old]))
                bindings.append({'kind': kind, 'story': unit['local_key'], 'old': ref, 'new': refs[-1]})
            unit['summary']['affected_refs'] = refs
    return result, bindings


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', required=True, type=Path)
    parser.add_argument('--units', type=Path, help='Explicit split-storage export [{kind, detail: full unit}]')
    parser.add_argument('--source-report-id', required=True)
    parser.add_argument('--publisher-report-id', required=True)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    raw = args.source.read_bytes()
    report = json.loads(raw)
    unit_bytes = args.units.read_bytes() if args.units else b''
    if args.units:
        for row in json.loads(unit_bytes):
            report.setdefault(row['kind'], []).append(row['detail'])
    converted, bindings = convert(report)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    payload = {'publisher_report_id': args.publisher_report_id, 'report': converted}
    args.output.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n')
    manifest = {'source_report_id': args.source_report_id,
                'source_sha256': hashlib.sha256(raw).hexdigest(),
                'units_sha256': hashlib.sha256(unit_bytes).hexdigest() if args.units else None,
                'output_sha256': hashlib.sha256(args.output.read_bytes()).hexdigest(),
                'publisher_report_id': args.publisher_report_id, 'references': bindings,
                'note': 'Offline only. Original snapshots retained. Missing metrics and allocations not invented.'}
    args.output.with_suffix('.manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n')
    print(f'Converted {len(bindings)} explicit homepage references; no database writes')


if __name__ == '__main__':
    main()
