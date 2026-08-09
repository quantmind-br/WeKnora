# Product Meeting Minutes · 2024 Q1 Planning

**Meeting Topic**: Smart Home Product Line Q1 Priority Review
**Time**: 2024-01-15 14:00–16:30
**Location**: HQ Building A, 3F Wutong Conference Room
**Recorder**: Li Wei (Product Operations)

## Attendees

| Name | Department | Role |
| --- | --- | --- |
| Chen Hao | Product Department | Product Director |
| Zhang Ming | R&D Department | Head of Knowledge Base & AI Module |
| Wang Xue | Design Department | Head of Interaction Design |
| Zhao Lei | QA Department | Test Manager |
| Liu Yang | Sales Department | East China Sales Director |

## I. Q1 Core Objectives

1. **Central Control Pro Firmware 3.5**: Complete Matter 1.2 certification, release to canary before end of March.
2. **NovaHome App 4.0**: Refactor device list and scene editor, enter closed beta by mid-February.
3. **Enterprise Knowledge Assistant**: Based on in-house RAG solution, provide installation/troubleshooting Q&A for the after-sales team (POC owned by Zhang Ming's team).

## II. Feature Priorities (MoSCoW)

| Priority | Feature | Owner | Target Date |
| --- | --- | --- | --- |
| Must | Matter bridge stability fix | Hardware Team | 2024-02-01 |
| Must | Scene editor drag-and-drop redesign | Wang Xue | 2024-02-15 |
| Must | After-sales knowledge base POC launched on internal network | Zhang Ming | 2024-03-01 |
| Should | Local voice pack offline recognition | Voice Team | 2024-03-15 |
| Could | Energy consumption weekly report | Data Team | Re-evaluate in Q2 |

## III. Key Decisions

### 3.1 After-Sales Knowledge Base POC Scope

- **Initial corpus**: Product manuals, installation guides, Top 50 ticket FAQs, approximately 200 documents expected.
- **Acceptance criteria**: For 100 randomly sampled after-sales questions, Top-3 hit rate ≥ 85%, and answers must include source links.
- **Technology selection**: Use a privately deployed WeKnora instance, local model for embedding, dialogue model connected to the company's unified API gateway.
- **Owner**: Zhang Ming; Product liaison: Li Wei.

### 3.2 App 4.0 Experience Principles

- Device list grouped by "room" by default, with one-click switch to "by type" view.
- Scene editing changed from "form-based" to "card + timeline", reducing the learning curve for new users.

### 3.3 Sales-Side Feedback

Liu Yang's feedback: East China customers care most about **offline availability** and **multi-hub cascading** (villa scenarios). The cascading solution is slated for Q2 preliminary research; this quarter will only deliver a technical feasibility report.

## IV. Risks and Dependencies

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Tight Matter certification schedule | Firmware release delayed | Submit pre-certification materials 2 weeks early |
| Inconsistent quality of after-sales corpus | Q&A accuracy falls short | Zhao Lei's team performs sample annotation, weekly review |
| Unified API gateway rate limiting | POC demo lag | Request dedicated quota, separate tenant for demo environment |

## V. Action Items

- [ ] Zhang Ming: Complete WeKnora test environment deployment and initial corpus import by January 20 (resources approved by Chen Hao)
- [ ] Wang Xue: Deliver high-fidelity scene editor prototype by January 25
- [ ] Li Wei: Send weekly POC progress email to attendees every Friday
- [ ] Zhao Lei: Submit cleaned Top 50 FAQ Excel by February 1

## VI. Next Meeting

- **Time**: 2024-02-05 14:00
- **Topics**: App 4.0 closed beta feedback, mid-term review of knowledge base POC
