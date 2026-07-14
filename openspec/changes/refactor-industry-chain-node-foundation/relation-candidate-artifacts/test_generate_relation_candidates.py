import importlib.util, json, pathlib, tempfile

HERE=pathlib.Path(__file__).parent
spec=importlib.util.spec_from_file_location("generator",HERE/"generate_relation_candidates.py")
g=importlib.util.module_from_spec(spec); spec.loader.exec_module(g)

def test_candidate_contract_and_determinism():
    g.main(); first=g.OUTPUT.read_bytes(); g.main(); assert first==g.OUTPUT.read_bytes()
    data=json.loads(first)
    assert data["input"]["node_count"]==842
    assert data["counts"]["reviewable_semantic"]>0
    assert data["counts"]["physical_constraint_blocked"]>0
    assert data["counts"]["reviewable_by_relation_type"]=={"is_component_of":1,"is_subcategory_of":95}
    assert data["counts"]["blocked_by_relation_type"]=={"depends_on":1,"input_to":5,"is_subcategory_of":47}
    assert data["qa"]["self_loops"]==0
    assert data["qa"]["write_ready_input_or_dependency"]==0
    assert not ({"contains","supplies_to","substitutes_for","transmits_to"} & {x["relation_type"] for x in data["reviewable_semantic"]})
    assert all(x["mechanism"] and x["derivation_evidence"] and x["counterexample"] for x in data["reviewable_semantic"])
    approved={e["key"] for e in json.loads(g.INPUT.read_text())["entities"]}
    review_tuples=[]
    for item in data["reviewable_semantic"]:
        assert item["from"]["entity_key"] in approved and item["to"]["entity_key"] in approved
        assert item["from"]["entity_key"] != item["to"]["entity_key"]
        review_tuples.append((item["from"]["entity_key"],item["relation_type"],item["to"]["entity_key"]))
    assert len(review_tuples)==len(set(review_tuples))
    assert all(x["missing_evidence"] and x["counterexample"] and x["uncertainty"] for x in data["blocked_needs_evidence"])
    assert all(x["subject"]["entity_key"] in approved and x["missing_evidence"] and x["recommendation"] for x in data["physical_constraints"])
