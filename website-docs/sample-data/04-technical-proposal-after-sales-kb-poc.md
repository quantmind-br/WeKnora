# Technical Proposal · After-Sales Knowledge Base POC

**Project codename**: Nova-KB-POC
**Version**: 0.3 Draft
**Author**: Zhang Ming (R&D Department)
**Reviewers**: Chen Hao (Product Department), Zhao Lei (QA Department)
**Last updated**: 2024-01-18

## 1. Background and Goals

Nebula Technology's after-sales team handles about 400 installation and troubleshooting tickets daily. The current solution relies on Confluence full-text search, and frontline engineers report "I can find the paragraph but don't know which sentence to send directly to the customer."

Goals of this POC:

1. Import internal documents such as product manuals, meeting minutes, and FAQs;
2. Support natural language questions, returning answers with **traceable sources**;
3. Complete an on-premise demo within 4 weeks, for mid-term review at the Q1 planning meeting.

## 2. System Architecture

```text
[Corpus Sources]  Confluence export / Markdown / PDF manuals
      ↓
[WeKnora]   docreader parsing → chunking → vector + BM25 hybrid retrieval
      ↓
[Model Layer]    Embedding: local bge-m3
            LLM: company API gateway → DeepSeek-V3
      ↓
[Integration]      Embedded via iframe in the after-sales ticketing system (Phase 2)
```

### 2.1 Key Roles

| Role | Name | Responsibility |
| --- | --- | --- |
| Technical Lead | Zhang Ming | Deployment, corpus pipeline, retrieval tuning |
| Product Liaison | Li Wei | Requirement prioritization, acceptance question bank |
| QA Lead | Zhao Lei | Annotate 100 acceptance questions, accuracy statistics |
| Approver | Chen Hao | Resource requests, scope trimming |

### 2.2 Relationship with the Central Control Product

The initial corpus for the POC primarily covers the **Smart Home Central Control Pro** (see the *Product Manual*), supplemented with milestone and owner information from the *Q1 Product Meeting Minutes*, to facilitate answering "who is responsible" and "when will it be delivered" type questions.

## 3. Corpus List (First Batch)

| No. | Document | Format | Estimated Chunks |
| --- | --- | --- | --- |
| 1 | Smart Home Central Control Pro Product Manual | Markdown / PDF | 40 |
| 2 | Q1 Product Meeting Minutes | Markdown | 25 |
| 3 | Employee Handbook · Reimbursement and Leave (After-Sales Travel Scenarios) | Markdown | 30 |
| 4 | Top 50 Ticket FAQ | Excel Import | 50 |

**Approximately 145 knowledge units in total**, meeting the minimum requirement for the March 1 on-premise demo.

## 4. Retrieval Strategy

- **Default**: Vector + BM25 hybrid retrieval, RRF fusion, Top-5 sent to the LLM.
- **Optional**: Enable Rerank (`bge-reranker-v2-m3`) to improve accuracy for polysemous scenarios.
- **Not yet enabled**: Knowledge graph (Neo4j has not yet been incorporated into the POC environment), automatic Wiki generation.

### 4.1 Chunking Parameters (Recommended)

| Parameter | Value | Notes |
| --- | --- | --- |
| chunk_size | 512 | Matches the paragraph length of the manual |
| chunk_overlap | 64 | Preserves table context |
| Parent-child chunking | Enabled | Retrieval hits child chunks, generation uses parent chunks |

## 5. Acceptance Test Cases (Examples)

| Question | Expected Answer Key Points | Expected Source |
| --- | --- | --- |
| What is the maximum number of devices the Central Control Pro can connect? | 256, recommended ≤ 120 | Product Manual · Technical Specifications |
| When will the Q1 knowledge base POC be accepted? | On-premise launch on 2024-03-01 | Q1 Meeting Minutes |
| What is the accommodation reimbursement cap for tier-1 cities? | 600 RMB / night | Employee Handbook · Travel Reimbursement |
| What is the target version for Matter certification? | Firmware 3.5, gray release by end of March | Q1 Meeting Minutes |
| Who is responsible for the after-sales knowledge base POC? | Zhang Ming | Meeting Minutes / This Proposal |

Zhao Lei will expand this into 100 formal acceptance questions based on this.

## 6. Risks

1. **Corpus obsolescence**: Product Manual v2.1 conflicts with the soon-to-be-released v2.2 → Establish a "corpus owner" mechanism, with Li Wei verifying monthly.
2. **Hallucination**: Enforce a prompt requiring "answer only based on citations; reply 'I don't know' if there is no basis."
3. **Permissions**: Single-tenant during the POC phase; production will require splitting the knowledge base ACL by regional after-sales group.

## 7. Milestones

| Date | Deliverable |
| --- | --- |
| 2024-01-20 | WeKnora test environment available, first batch of 3 Markdown documents imported |
| 2024-02-01 | FAQ Excel import completed, hybrid retrieval + Rerank tuning |
| 2024-02-05 | Mid-term review demo |
| 2024-03-01 | 100-question acceptance report |
