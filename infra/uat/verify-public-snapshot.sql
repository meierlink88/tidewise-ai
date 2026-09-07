-- Issue #422: exact row-content and sequence checks for the approved v84 snapshot.
-- Values are fingerprints, never business records or credentials.
SET timezone = 'UTC';
SET datestyle = 'ISO, YMD';
DO $verify$
DECLARE
  actual_count bigint;
  actual_digest text;
  seq_value bigint;
  seq_called boolean;
BEGIN
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."chain_node" t;
  IF actual_count <> 3049 OR actual_digest <> 'abef799206f991bb190c2ab9cc3d95ef' THEN
    RAISE EXCEPTION 'snapshot mismatch: chain_node';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."commodity_profiles" t;
  IF actual_count <> 45 OR actual_digest <> 'bc304ca60e638a071c5c1b66b26dd180' THEN
    RAISE EXCEPTION 'snapshot mismatch: commodity_profiles';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."company" t;
  IF actual_count <> 13264 OR actual_digest <> 'e9cb3b42df86cce72876a5ba40f31442' THEN
    RAISE EXCEPTION 'snapshot mismatch: company';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."company_industry_links" t;
  IF actual_count <> 0 OR actual_digest <> 'd41d8cd98f00b204e9800998ecf8427e' THEN
    RAISE EXCEPTION 'snapshot mismatch: company_industry_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."concept" t;
  IF actual_count <> 212 OR actual_digest <> '02d49051d60e846e6a20d71f4850ec76' THEN
    RAISE EXCEPTION 'snapshot mismatch: concept';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."countries" t;
  IF actual_count <> 201 OR actual_digest <> '00d46995b6e18d50784b288ead53c06a' THEN
    RAISE EXCEPTION 'snapshot mismatch: countries';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."country_region_links" t;
  IF actual_count <> 201 OR actual_digest <> '0544d66e7d3544e3d29e6ca5cf9e5d82' THEN
    RAISE EXCEPTION 'snapshot mismatch: country_region_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."entity_edges" t;
  IF actual_count <> 44 OR actual_digest <> '756788f12c2f5042607f4a9924d2a239' THEN
    RAISE EXCEPTION 'snapshot mismatch: entity_edges';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."entity_nodes" t;
  IF actual_count <> 355 OR actual_digest <> '36914f32ecfa85e01d44baeb5a4ad83e' THEN
    RAISE EXCEPTION 'snapshot mismatch: entity_nodes';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."event_actor_links" t;
  IF actual_count <> 0 OR actual_digest <> 'd41d8cd98f00b204e9800998ecf8427e' THEN
    RAISE EXCEPTION 'snapshot mismatch: event_actor_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."event_asset_links" t;
  IF actual_count <> 0 OR actual_digest <> 'd41d8cd98f00b204e9800998ecf8427e' THEN
    RAISE EXCEPTION 'snapshot mismatch: event_asset_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."event_evidence_links" t;
  IF actual_count <> 173 OR actual_digest <> '9c497274a5fe28c70f804a698d1b8c51' THEN
    RAISE EXCEPTION 'snapshot mismatch: event_evidence_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."event_publication_receipts" t;
  IF actual_count <> 168 OR actual_digest <> '412584e43b026d5adc0707ddc70f8708' THEN
    RAISE EXCEPTION 'snapshot mismatch: event_publication_receipts';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."events" t;
  IF actual_count <> 168 OR actual_digest <> '7f350514ca2351be9cd65a7b8532ae02' THEN
    RAISE EXCEPTION 'snapshot mismatch: events';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."evidence_categories" t;
  IF actual_count <> 11 OR actual_digest <> '84842edbb3bacfaff012ea868f26e7b1' THEN
    RAISE EXCEPTION 'snapshot mismatch: evidence_categories';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."evidences" t;
  IF actual_count <> 802 OR actual_digest <> '4e7a5ec380fb6a58e1e1d4183d7e09ce' THEN
    RAISE EXCEPTION 'snapshot mismatch: evidences';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."geopolitic_domains" t;
  IF actual_count <> 14 OR actual_digest <> '72c0ed9c1268126733487c76ba998989' THEN
    RAISE EXCEPTION 'snapshot mismatch: geopolitic_domains';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."geopolitic_rivalries" t;
  IF actual_count <> 44 OR actual_digest <> 'c1856bf992430582c15ae1266fc1a1d1' THEN
    RAISE EXCEPTION 'snapshot mismatch: geopolitic_rivalries';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."goose_db_version" t;
  IF actual_count <> 85 OR actual_digest <> '2a2fffbdbc5bf751c0dbcba0f1170ff3' THEN
    RAISE EXCEPTION 'snapshot mismatch: goose_db_version';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."index_profiles" t;
  IF actual_count <> 43 OR actual_digest <> 'c667be420c5de5c5bd4e6316b0c241da' THEN
    RAISE EXCEPTION 'snapshot mismatch: index_profiles';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."industry" t;
  IF actual_count <> 512 OR actual_digest <> 'c8bf4396b1dbb5b317079700226a82c1' THEN
    RAISE EXCEPTION 'snapshot mismatch: industry';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."industry_chain" t;
  IF actual_count <> 708 OR actual_digest <> '345c8c60aba7bb0b25ca05e73c33814e' THEN
    RAISE EXCEPTION 'snapshot mismatch: industry_chain';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."industry_chain_concept_links" t;
  IF actual_count <> 619 OR actual_digest <> '7fffb0ea97a16fe7b3d3265a0322eb04' THEN
    RAISE EXCEPTION 'snapshot mismatch: industry_chain_concept_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."industry_chain_graph_edges" t;
  IF actual_count <> 2645 OR actual_digest <> '78101d606156a35f04be88a75d398b87' THEN
    RAISE EXCEPTION 'snapshot mismatch: industry_chain_graph_edges';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."industry_chain_industry_links" t;
  IF actual_count <> 716 OR actual_digest <> '9bdad0e7b9c0c8383aa0694b75fb3574' THEN
    RAISE EXCEPTION 'snapshot mismatch: industry_chain_industry_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."industry_chain_node_memberships" t;
  IF actual_count <> 3350 OR actual_digest <> '29efec8596bad5958f92b66ae02385fd' THEN
    RAISE EXCEPTION 'snapshot mismatch: industry_chain_node_memberships';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."institutions" t;
  IF actual_count <> 0 OR actual_digest <> 'd41d8cd98f00b204e9800998ecf8427e' THEN
    RAISE EXCEPTION 'snapshot mismatch: institutions';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."instrument_profiles" t;
  IF actual_count <> 4 OR actual_digest <> 'dc95af4f5e29e6db50761f39dfe7eb00' THEN
    RAISE EXCEPTION 'snapshot mismatch: instrument_profiles';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."macro_economics" t;
  IF actual_count <> 34 OR actual_digest <> 'd4d86d4a9a36eed7b794f592fdf0c1db' THEN
    RAISE EXCEPTION 'snapshot mismatch: macro_economics';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."macro_economics_domain" t;
  IF actual_count <> 10 OR actual_digest <> '0b6689c8710251dc98f221c59f513238' THEN
    RAISE EXCEPTION 'snapshot mismatch: macro_economics_domain';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."market_profiles" t;
  IF actual_count <> 47 OR actual_digest <> '2ce1a94016119a1284ef68cf6c1e904c' THEN
    RAISE EXCEPTION 'snapshot mismatch: market_profiles';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."ministries" t;
  IF actual_count <> 0 OR actual_digest <> 'd41d8cd98f00b204e9800998ecf8427e' THEN
    RAISE EXCEPTION 'snapshot mismatch: ministries';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."organization_categories" t;
  IF actual_count <> 4 OR actual_digest <> 'd1e7553b03b2d2c029b08e266df663f1' THEN
    RAISE EXCEPTION 'snapshot mismatch: organization_categories';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."organization_domain_tag_links" t;
  IF actual_count <> 78 OR actual_digest <> '4dff7c26913326bf0343ae2b4e380b6f' THEN
    RAISE EXCEPTION 'snapshot mismatch: organization_domain_tag_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."organization_domain_tags" t;
  IF actual_count <> 21 OR actual_digest <> '4ca6b5fd48bbcf41037cfd60ed457748' THEN
    RAISE EXCEPTION 'snapshot mismatch: organization_domain_tags';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."organization_functions" t;
  IF actual_count <> 7 OR actual_digest <> '26626932429e875538cb9c9927f904c7' THEN
    RAISE EXCEPTION 'snapshot mismatch: organization_functions';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."organization_members" t;
  IF actual_count <> 1543 OR actual_digest <> '10c70527810c877229553c5e79ea16b0' THEN
    RAISE EXCEPTION 'snapshot mismatch: organization_members';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."organizations" t;
  IF actual_count <> 78 OR actual_digest <> 'be0eb5cf2f48b2bbebd3ff971e2381f6' THEN
    RAISE EXCEPTION 'snapshot mismatch: organizations';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."person_profiles" t;
  IF actual_count <> 30 OR actual_digest <> 'b5ffe00cc6e446e7ea7c237f4ef98922' THEN
    RAISE EXCEPTION 'snapshot mismatch: person_profiles';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."policy_body_profiles" t;
  IF actual_count <> 30 OR actual_digest <> 'e9113348f58bff22ff6097f03f603148' THEN
    RAISE EXCEPTION 'snapshot mismatch: policy_body_profiles';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."raw_evidence_category_links" t;
  IF actual_count <> 334 OR actual_digest <> '27a6f8608aff31705a5890429344d46e' THEN
    RAISE EXCEPTION 'snapshot mismatch: raw_evidence_category_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."raw_evidences" t;
  IF actual_count <> 334 OR actual_digest <> 'a6254c72312cabce7e142d5d90e91227' THEN
    RAISE EXCEPTION 'snapshot mismatch: raw_evidences';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."regions" t;
  IF actual_count <> 22 OR actual_digest <> '177100a7c4a464a986350c8ee937d9a5' THEN
    RAISE EXCEPTION 'snapshot mismatch: regions';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."report_evidence_links" t;
  IF actual_count <> 256 OR actual_digest <> 'bf885b228e290db1f316862a90ecc021' THEN
    RAISE EXCEPTION 'snapshot mismatch: report_evidence_links';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."reports" t;
  IF actual_count <> 2 OR actual_digest <> '2cb73392df1bd81796bd9c8d9ec1d6a1' THEN
    RAISE EXCEPTION 'snapshot mismatch: reports';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."security_profiles" t;
  IF actual_count <> 77 OR actual_digest <> 'a8d6a1d5cc0c74db259f78afcb7d47b5' THEN
    RAISE EXCEPTION 'snapshot mismatch: security_profiles';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."sources" t;
  IF actual_count <> 27 OR actual_digest <> '72e74e2e5199cdd38c893a7c4f0edf3c' THEN
    RAISE EXCEPTION 'snapshot mismatch: sources';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."subdivisions" t;
  IF actual_count <> 0 OR actual_digest <> 'd41d8cd98f00b204e9800998ecf8427e' THEN
    RAISE EXCEPTION 'snapshot mismatch: subdivisions';
  END IF;
  SELECT count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '' ORDER BY md5(to_jsonb(t)::text)), ''))
    INTO actual_count, actual_digest FROM public."theme_profiles" t;
  IF actual_count <> 0 OR actual_digest <> 'd41d8cd98f00b204e9800998ecf8427e' THEN
    RAISE EXCEPTION 'snapshot mismatch: theme_profiles';
  END IF;
  SELECT last_value, is_called INTO seq_value, seq_called FROM public."goose_db_version_id_seq";
  IF seq_value <> 85 OR seq_called <> true THEN
    RAISE EXCEPTION 'snapshot sequence mismatch: goose_db_version_id_seq';
  END IF;
END;
$verify$;
