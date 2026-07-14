#!/usr/bin/env python3
import hashlib, json, pathlib

ROOT = pathlib.Path(__file__).resolve().parents[1]
INPUT = ROOT / "final-seed-candidate-artifacts/node-profile-seed-manifest.json"
OUTPUT = pathlib.Path(__file__).with_name("relation-candidate-review.json")
RULE_VERSION = "phase-b-semantic-v1"
SUBCATEGORY_ROOTS = {"芯片","电池","小家电","物流","电商","零售","游戏","保险","疫苗","塑料","储能","半导体","乘用车","机器人","饲料","印刷","出版","银行","医院","物联网","仪器仪表","自动化设备","通信设备","计算机设备","通用设备","专用设备","金属新材料","化学制品","化学原料","化学纤维","农产品加工","医疗服务","专业工程","专业服务","家居用品","数字媒体","橡胶制品","生物制品","种植业","饰品","黑色家电","化妆品","航母","军工","教育","大数据","电网","租赁","免疫治疗"}
CONNECTORS = ("及", "与", "和", "/")
REVIEW_BLOCK_NAMES = {"造纸印刷", "短剧互动游戏", "国防军工"}

def sha(path): return hashlib.sha256(path.read_bytes()).hexdigest()
def load():
    doc=json.loads(INPUT.read_text()); return doc, {e["name"]:e for e in doc["entities"]}
def endpoint(e):
    p=e["profile"]; return {"name":e["name"],"entity_key":e["key"],"definition":p["definition"],"boundary_note":p.get("boundary_note")}
def candidate(kind,a,b,mechanism,rule,confidence="high"):
    return {"disposition":"reviewable_semantic","relation_type":kind,"from":endpoint(a),"to":endpoint(b),"direction":f'{a["name"]} -> {b["name"]}',"mechanism":mechanism,"condition_note":"仅限已批准 definition/boundary 所表达的稳定语义范围","derivation_evidence":{"source":"approved 842-node manifest","input_sha256":sha(INPUT),"rule":rule},"provenance":f"{RULE_VERSION}:{rule}","confidence":confidence,"counterexample":"若子项名称只是市场标签、并列复合词或边界未覆盖全部实例，则关系不成立","uncertainty":"尚未经过第二遍独立 Serenity Reviewer"}
def blocked(kind,a,b,mechanism,missing,rule):
    return {"disposition":"blocked_needs_evidence","relation_type":kind,"from":endpoint(a),"to":endpoint(b),"direction":f'{a["name"]} -> {b["name"]}',"mechanism":mechanism,"current_evidence":"仅有节点名称、definition 与 boundary，不能证明真实投入或功能约束","missing_evidence":missing,"provenance":f"{RULE_VERSION}:{rule}","counterexample":"可能只是相邻概念、可替代路径或共同受第三因素影响","uncertainty":"不得进入可写 manifest"}
def main():
    doc,nodes=load(); reviewable=[]; blocked_items=[]; rejected=[]; seen=set()
    names=sorted(nodes)
    aliases={alias:e["name"] for e in doc["entities"] for alias in e.get("aliases",[])}
    for child in names:
        # Enumerate only suffixes of this name; never construct the 842 x 842 pair space.
        for offset in range(1, len(child)-1):
            parent=child[offset:]
            if parent not in nodes or len(parent)<2: continue
            key=(child,parent,"is_subcategory_of")
            if child in aliases and aliases[child]==parent:
                rejected.append({"candidate":key,"reason":"alias/synonym is identity normalization, not relation"}); continue
            if any(x in child[:-len(parent)] for x in CONNECTORS):
                rejected.append({"candidate":key,"reason":"compound/union name cannot prove all instances belong to parent"}); continue
            if parent in SUBCATEGORY_ROOTS and child not in REVIEW_BLOCK_NAMES and (child.startswith("其他") or len(child)-len(parent)<=6):
                reviewable.append(candidate("is_subcategory_of",nodes[child],nodes[parent],f"{child} 的全部稳定语义实例属于 {parent} 的范围", "suffix-subtype-allowlist")); seen.add(key)
            else:
                blocked_items.append(blocked("is_subcategory_of",nodes[child],nodes[parent],"名称后缀提示可能的分类范围从属","需要权威分类定义或技术标准证明全部实例从属","suffix-needs-boundary-proof"))
    if "汽车零部件" in nodes and "汽车" in nodes:
        reviewable.append(candidate("is_component_of",nodes["汽车零部件"],nodes["汽车"],"汽车零部件是汽车可识别的物理/系统组成","explicit-component-name"))
    leads=[
      ("input_to","锂","锂电池","锂作为可识别材料投入锂电池生产","需要工艺/BOM/技术标准证明直接消耗"),
      ("input_to","半导体材料","半导体","半导体材料作为制造过程可识别输入","需要工艺文件或行业标准证明材料与过程边界"),
      ("depends_on","半导体","半导体设备","半导体产出在关键设备缺失或受限时受约束","需要设备不可替代性、产线配置或资格认证证据"),
      ("input_to","光伏主材","光伏电池组件","光伏主材可能被组件制造直接消耗","需要BOM、工艺路线或技术标准"),
      ("input_to","稀土产业","稀土永磁","稀土原料可能被永磁材料生产直接消耗","需要材料配方、工艺或行业标准"),
      ("input_to","铜产业","铜缆高速连接","铜材可能被高速铜缆直接消耗","需要产品BOM或材料规格")]
    for kind,a,b,m,missing in leads:
        if a in nodes and b in nodes: blocked_items.append(blocked(kind,nodes[a],nodes[b],m,missing,"serenity-hard-evidence-lead"))
    physical=[]
    for name,ctype,desc,missing in [
      ("半导体","process_yield","制造良率可能构成可量化硬工艺约束","需要工艺节点良率、设备/材料敏感性与产能损失来源"),
      ("半导体设备","equipment_capacity","关键设备供给与安装调试能力可能约束扩产","需要交付周期、装机、认证与扩产数据"),
      ("锂矿","resource_availability","可采资源与开发周期可能约束物理供给","需要储量、品位、许可、建设周期与产量来源"),
      ("电池","material_purity","关键材料纯度可能约束产品性能与良率","需要材料规格、失效机理与资格认证来源")]:
        if name in nodes: physical.append({"disposition":"blocked_needs_evidence","subject":endpoint(nodes[name]),"constraint_type":ctype,"description":desc,"why_physical":"直接限制可生产数量、良率、合格率或扩产速度，而非价格、政策、情绪或行情","current_evidence":"节点语义仅支持研究方向，不足以确认约束事实","missing_evidence":missing,"recommendation":"补足强技术/产能/资源证据后由第二遍 Reviewer 决定 approve/reject"})
    reviewable=sorted(reviewable,key=lambda x:(x["relation_type"],x["direction"])); blocked_items=sorted(blocked_items,key=lambda x:(x["relation_type"],x["direction"])); rejected=sorted(rejected,key=lambda x:str(x["candidate"]))
    out={"artifact_type":"phase_b_relation_candidate_review","rule_version":RULE_VERSION,"input":{"path":str(INPUT.relative_to(ROOT)),"sha256":sha(INPUT),"node_count":len(nodes)},"counts":{"reviewable_semantic":len(reviewable),"blocked_needs_evidence":len(blocked_items),"physical_constraint_blocked":len(physical),"rejected":len(rejected),"reviewable_by_relation_type":{},"blocked_by_relation_type":{}},"reviewable_semantic":reviewable,"blocked_needs_evidence":blocked_items,"physical_constraints":physical,"rejected":rejected,"rejected_rules":[{"rule":"self_loop","disposition":"rejected"},{"rule":"alias_or_synonym","disposition":"rejected"},{"rule":"legacy_relation_type","values":["contains","supplies_to","substitutes_for","transmits_to"],"disposition":"rejected"},{"rule":"dynamic_event_transmission","disposition":"rejected"},{"rule":"same_mechanism_input_and_dependency","disposition":"rejected"}],"qa":{"algorithm":"suffix-index O(sum(name_length)); no Cartesian product","no_full_pair_scan":True,"self_loops":0,"forbidden_relation_types":0,"write_ready_input_or_dependency":0,"write_ready_physical_constraints":0}}
    for item in reviewable: out["counts"]["reviewable_by_relation_type"][item["relation_type"]]=out["counts"]["reviewable_by_relation_type"].get(item["relation_type"],0)+1
    for item in blocked_items: out["counts"]["blocked_by_relation_type"][item["relation_type"]]=out["counts"]["blocked_by_relation_type"].get(item["relation_type"],0)+1
    out["deterministic_qa_sample"]=(reviewable[:5]+reviewable[-5:]) if len(reviewable)>10 else reviewable
    payload=json.dumps(out,ensure_ascii=False,indent=2)+"\n"; OUTPUT.write_text(payload); print(json.dumps({"output":str(OUTPUT),"sha256":hashlib.sha256(payload.encode()).hexdigest(),"counts":out["counts"]},ensure_ascii=False))
if __name__=="__main__": main()
