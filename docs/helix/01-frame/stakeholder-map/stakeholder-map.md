---
title: "Stakeholder Map"
phase: "01-frame"
category: "planning"
tags: ["stakeholders", "communication", "engagement", "raci"]
related: ["product-requirements", "user-stories", "compliance-requirements"]
created: 2025-01-10
updated: 2026-03-17
helix_mapping: "Frame phase stakeholder identification and engagement planning"
---

# Stakeholder Map

**Document Type**: Stakeholder Map
**Status**: Approved
**Last Updated**: 2026-03-17
**Author**: Engineering Team

## Executive Summary

GRCTool serves a diverse set of stakeholders spanning internal compliance and engineering teams through to external auditors. The primary value proposition -- automating SOC 2 and ISO 27001 evidence collection -- means that compliance managers and security engineers are the most deeply engaged stakeholders, while CISOs and external auditors depend on the quality and traceability of outputs. This stakeholder map identifies key parties, their influence and interest levels, and defines communication and engagement strategies to ensure project success.

## Stakeholder Identification

### Primary Stakeholders (High Influence, High Interest)

#### Compliance Manager
- **Role**: Oversees organizational compliance programs and audit preparation
- **Organization**: Compliance / GRC Team
- **Interest**: Reduce audit preparation time from weeks to days; ensure evidence quality meets auditor requirements; maintain continuous compliance posture
- **Influence**: High -- primary budget justifier and adoption champion; determines whether GRCTool becomes the standard compliance workflow
- **Concerns**: Evidence accuracy and completeness; auditor acceptance of generated evidence; learning curve for CLI-based tooling; data privacy in evidence collection
- **Success Criteria**: 60%+ reduction in manual evidence preparation time; 90%+ evidence acceptance rate on first review; audit readiness achieved in hours rather than days
- **Communication Preference**: Weekly status updates; quarterly roadmap reviews; direct Slack channel for issues

#### Security Engineer
- **Role**: Implements and validates security controls across infrastructure
- **Organization**: Security Engineering Team
- **Interest**: Prove control effectiveness with technical evidence from actual infrastructure; automate repetitive evidence gathering from Terraform and GitHub
- **Influence**: High -- primary technical user; shapes tool requirements through daily usage; validates that evidence accurately reflects infrastructure state
- **Concerns**: Evidence accuracy against real infrastructure; integration reliability with GitHub and Terraform; tool performance on large codebases; false positives in security analysis
- **Success Criteria**: Automated evidence generation for 85%+ of technical controls; infrastructure analysis completes within minutes; evidence maps correctly to SOC 2 and ISO 27001 controls
- **Communication Preference**: GitHub issues and pull requests; technical documentation; sprint planning participation

#### DevOps / Site Reliability Engineer
- **Role**: Manages infrastructure, deployment pipelines, and operational health
- **Organization**: Platform / Infrastructure Team
- **Interest**: Validate infrastructure compliance without disrupting deployment workflows; integrate compliance checks into CI/CD pipelines
- **Influence**: High -- controls the infrastructure that GRCTool analyzes; can block adoption if tool impacts deployment velocity
- **Concerns**: Performance impact on CI/CD pipelines; accuracy of Terraform and Atmos stack analysis; credential management for automation; support for multi-environment scanning
- **Success Criteria**: Evidence generation integrates into existing pipelines without workflow disruption; infrastructure compliance verification in under 5 minutes; zero false negatives on critical controls
- **Communication Preference**: CI/CD integration documentation; infrastructure-focused changelogs; async Slack communication

### Secondary Stakeholders (Variable Influence/Interest)

#### External Auditor
- **Role**: Conducts SOC 2 Type II and ISO 27001 audits
- **Organization**: External audit firm
- **Interest**: Structured, traceable evidence with clear reasoning and source attribution; efficient audit process
- **Influence**: High influence (determines audit outcome) but low direct project involvement
- **Engagement Level**: Inform -- provide evidence format specifications; gather feedback on output quality post-audit
- **Concerns**: Evidence integrity and authenticity; completeness of audit trails; ability to trace evidence back to source systems; compliance with professional auditing standards

#### CISO / Executive Leadership
- **Role**: Chief Information Security Officer; executive oversight of security and compliance programs
- **Organization**: Executive Team
- **Interest**: Compliance posture visibility; risk reduction; cost efficiency of compliance operations; board-level reporting capability
- **Influence**: High -- budget authority and strategic direction; can accelerate or halt adoption
- **Engagement Level**: Consult -- quarterly briefings on compliance posture improvements and ROI metrics
- **Concerns**: Return on investment; risk exposure from tool failures; reputational impact of audit findings; competitive positioning

#### Engineering Manager
- **Role**: Manages development teams that contribute to compliance-relevant systems
- **Organization**: Engineering Department
- **Interest**: Minimize compliance burden on development teams; understand evidence requirements for systems their teams own
- **Influence**: Medium -- controls engineering team priorities and resource allocation for compliance tasks
- **Engagement Level**: Consult -- involve in evidence task ownership and team capacity planning
- **Concerns**: Time impact on development velocity; clarity of evidence task assignments; quality of AI-generated evidence requiring review

### Supporting Stakeholders (Low Influence, Variable Interest)

#### Tool Developers / Contributors
- **Description**: Engineers who maintain and extend GRCTool itself
- **Impact**: Directly affected by architecture decisions, security requirements, and development practices
- **Engagement**: Active participation in design reviews, code reviews, and sprint planning; access to comprehensive developer documentation
- **Interest**: Clean codebase with good test coverage; extensible tool architecture; clear contribution guidelines

#### Infrastructure / Platform Team
- **Description**: Team managing cloud infrastructure (AWS/GCP), Terraform configurations, and Atmos stacks
- **Impact**: GRCTool analyzes their infrastructure configurations; evidence accuracy depends on their cooperation
- **Engagement**: Inform about scanning schedules; consult on Terraform path configurations; collaborate on multi-environment support
- **Interest**: Minimal disruption to infrastructure operations; accurate representation of security controls

#### AI/LLM Integration Partners (Anthropic/Claude AI)
- **Description**: AI service provider powering evidence generation, analysis, and compliance document summarization
- **Role**: AI service provider for evidence generation and analysis
- **Impact**: Core evidence generation pipeline depends on Claude AI capabilities; changes to model behavior, API availability, or pricing directly affect GRCTool functionality
- **Interest**: Data handling practices (what compliance data is sent to the API); API usage patterns and rate limits; model capability alignment with evidence generation requirements; responsible AI usage
- **Influence**: High -- core evidence generation depends on AI service availability and capability; model changes can affect evidence quality
- **Engagement**: Monitor API documentation and changelog for breaking changes; maintain service agreements; track capability updates and model version changes; validate evidence quality after model updates
- **Concerns**: Data privacy and retention policies for compliance data sent via API; service reliability and uptime; cost predictability; model output consistency across versions

#### HR and People Operations
- **Description**: Human resources team responsible for personnel security processes
- **Impact**: Some evidence tasks require HR data (background checks, terminations, training compliance)
- **Engagement**: Inform about evidence requirements; coordinate manual evidence collection for HR-dependent tasks
- **Interest**: Clear data handling expectations; minimal additional workload

#### Integration Target Vendors
- **Description**: External platforms that GRCTool integrates with bidirectionally -- Tugboat Logic as the primary integration target, with future targets including identity providers (Okta, Azure AD), SIEM systems (Splunk, Datadog), and other GRC platforms (Vanta, Drata, ServiceNow GRC)
- **Role**: External platforms whose APIs, data formats, and sync protocols GRCTool depends on for bidirectional compliance data exchange
- **Impact**: API changes, deprecations, or outages can break inbound/outbound integration adapters; data format changes affect master index reconciliation
- **Interest**: API compatibility and versioning standards; data format consistency (JSON schema contracts); sync reliability and conflict resolution behavior; integration certification and partnership programs
- **Influence**: Medium-High -- API breaking changes can disrupt sync operations and evidence submission workflows; vendor platform decisions constrain adapter design
- **Engagement**: Track API versioning and changelogs; maintain integration test suites against vendor sandboxes; coordinate on data format standards; explore partnership and certification programs for preferred integration status
- **Concerns**: Breaking API changes without adequate deprecation notice; data format drift between platform versions; rate limiting and throttling during bulk sync operations; authentication protocol changes

## RACI Matrix

| Activity/Decision | Compliance Manager | Security Engineer | DevOps Engineer | CISO | External Auditor | Tool Developer | Integration Target Vendors |
|---|---|---|---|---|---|---|---|
| Product Vision and Roadmap | C | C | C | A | I | R | I |
| Security Requirements | C | R | C | A | I | R | I |
| Evidence Task Definition | A | R | C | I | C | I | I |
| Tool Development and Testing | I | C | C | I | I | R/A | I |
| Evidence Generation | A | R | R | I | I | I | I |
| Evidence Review and Approval | A | R | C | I | C | I | I |
| Audit Preparation | R/A | R | C | I | C | I | I |
| Auditor Handoff | A | C | I | I | R | I | I |
| Infrastructure Integration | C | C | R/A | I | I | C | I |
| Authentication and Security | C | R/A | C | A | I | R | C |
| Release Decisions | C | C | C | A | I | R | I |
| Sync Operations | A | R | C | I | I | R | C |
| Data Governance | A | C | I | A | C | R | C |
| Integration Management | C | R | C | I | I | R/A | R |

**Legend:**
- **R** = Responsible (does the work)
- **A** = Accountable (approves, has final authority)
- **C** = Consulted (provides input)
- **I** = Informed (kept in the loop)

## Stakeholder Analysis

### Power/Interest Grid

```
High Power  | Keep Satisfied          | Manage Closely
            | - CISO/Executive        | - Compliance Manager
            | - External Auditor      | - Security Engineer
            |                         | - DevOps Engineer
            |                         | - AI/LLM Partners (Anthropic)
            |                         | - Integration Target Vendors
            |-------------------------|-------------------------
Low Power   | Monitor                 | Keep Informed
            | - HR/People Ops         | - Engineering Manager
            |                         | - Tool Developers
            |                         | - Infrastructure Team
            Low Interest              High Interest
```

### Influence/Impact Matrix

| Stakeholder | Influence Level | Impact of Project on Them | Current Attitude | Engagement Strategy |
|---|---|---|---|---|
| Compliance Manager | High | High -- transforms daily workflow | Champion | Co-design evidence workflows; weekly syncs |
| Security Engineer | High | High -- automates manual evidence tasks | Supportive | Involve in technical design; rapid feedback loops |
| DevOps Engineer | High | Medium -- adds compliance to CI/CD | Cautious | Demonstrate minimal pipeline impact; early pilots |
| CISO | High | Medium -- improves risk visibility | Supportive | Quarterly ROI briefings; compliance dashboards |
| External Auditor | High | Low -- receives better-formatted evidence | Neutral | Share evidence format samples; collect feedback |
| Engineering Manager | Medium | Medium -- evidence tasks affect team capacity | Cautious | Clear task ownership; minimize disruption |
| Tool Developers | Low | High -- daily development work | Champion | Comprehensive docs; clean architecture |
| Infrastructure Team | Low | Medium -- systems are analyzed | Neutral | Transparent scanning; config collaboration |
| AI/LLM Partners (Anthropic) | High | Medium -- API and model changes affect evidence generation | Partner | API monitoring; service agreement reviews; capability validation |
| Integration Target Vendors | Medium-High | Medium -- API and platform changes affect adapter reliability | Partner | API version tracking; integration testing; partnership coordination |
| HR/People Ops | Low | Low -- occasional data requests | Neutral | Clear, infrequent communication |

## Communication Plan

### Regular Communications

| Stakeholder Group | Channel | Frequency | Content | Owner |
|---|---|---|---|---|
| Compliance Manager | 1:1 Meeting / Slack | Weekly | Evidence generation status, task progress, quality metrics | Product Lead |
| Security Engineer | Sprint Planning / GitHub | Bi-weekly | Technical roadmap, tool capabilities, integration updates | Tech Lead |
| DevOps Engineer | Slack / Documentation | Bi-weekly | Pipeline integration guides, performance metrics, breaking changes | Tech Lead |
| CISO | Executive Briefing | Quarterly | Compliance posture, ROI metrics, risk reduction, roadmap | Product Lead |
| External Auditor | Email / Formal Report | Per audit cycle | Evidence format specifications, output samples, process documentation | Compliance Manager |
| Engineering Manager | Team Meeting | Monthly | Evidence task assignments, capacity impact, upcoming requirements | Compliance Manager |
| Tool Developers | GitHub / Sprint Ceremonies | Daily/Weekly | Technical decisions, code reviews, architecture changes | Tech Lead |
| Infrastructure Team | Slack / Documentation | As needed | Terraform scanning configuration, multi-environment support | DevOps Lead |
| AI/LLM Partners (Anthropic) | API docs / Service agreements | Monthly | API changes, model updates, capability alignment, data handling policies | Tech Lead |
| Integration Target Vendors | API changelogs / Partnership channels | Monthly | API version changes, deprecation notices, data format updates, integration health metrics | Tech Lead |

### Escalation Path

1. **Level 1**: Tech Lead -- day-to-day technical decisions, bug prioritization, sprint scope
2. **Level 2**: Product Lead -- feature prioritization, resource conflicts, stakeholder concerns
3. **Level 3**: CISO -- security architecture decisions, compliance risk acceptance, budget changes
4. **Level 4**: Executive Committee -- strategic direction changes, major investment decisions

## Stakeholder Engagement Strategy

### Engagement Principles
1. Transparency in compliance posture and evidence quality metrics
2. Regular, predictable communication cadence tailored to each stakeholder's needs
3. Two-way feedback channels that influence product direction
4. Minimize disruption to existing workflows during adoption

### Engagement Tactics by Stakeholder

#### Compliance Manager (Champion)
- **Current State**: Actively using GRCTool for quarterly audit preparation
- **Desired State**: GRCTool is the standard compliance workflow; evidence generation is routine and trusted
- **Actions**:
  - Co-design evidence review and approval workflows
  - Provide early access to new evidence tools for validation
  - Gather feedback on evidence quality after each audit cycle
  - Include in roadmap prioritization discussions
- **Success Metrics**: Evidence acceptance rate; audit preparation time reduction; voluntary tool advocacy

#### Security Engineer (Supportive)
- **Current State**: Uses GRCTool for Terraform and GitHub evidence generation
- **Desired State**: GRCTool covers 90%+ of technical evidence needs; deeply integrated into security workflows
- **Actions**:
  - Involve in technical architecture decisions
  - Rapid iteration on tool accuracy based on infrastructure changes
  - Provide API and extension documentation for custom tools
  - Pair on complex evidence generation scenarios
- **Success Metrics**: Tool coverage of assigned evidence tasks; time saved per evidence collection cycle

#### DevOps Engineer (Cautious)
- **Current State**: Aware of GRCTool; concerned about pipeline impact
- **Desired State**: GRCTool compliance checks integrated into CI/CD with minimal overhead
- **Actions**:
  - Run performance benchmarks demonstrating minimal pipeline impact
  - Provide CI/CD integration examples (GitHub Actions)
  - Pilot with non-blocking compliance checks before making them gates
  - Collaborate on multi-environment scanning support
- **Success Metrics**: Pipeline integration adoption; zero deployment delays from compliance checks

#### CISO (Keep Satisfied)
- **Current State**: Supportive of automation initiative; wants ROI evidence
- **Desired State**: Confident in compliance posture; uses GRCTool metrics for board reporting
- **Actions**:
  - Deliver quarterly compliance posture summaries with trend analysis
  - Quantify cost and time savings vs. manual processes
  - Report on audit outcomes and evidence quality improvements
  - Highlight risk reduction from continuous evidence collection
- **Success Metrics**: Continued budget approval; executive advocacy for broader adoption

## Risk Analysis

### Stakeholder-Related Risks

| Risk | Stakeholder | Probability | Impact | Mitigation |
|---|---|---|---|---|
| Auditor rejects AI-generated evidence format | External Auditor | Medium | High | Share sample outputs early; include source attribution and reasoning transparency |
| DevOps team blocks CI/CD integration | DevOps Engineer | Low | High | Demonstrate minimal performance impact; start with non-blocking checks |
| CISO withdraws budget support | CISO | Low | Critical | Regular ROI reporting; quantify audit cost and time savings |
| Compliance manager overwhelmed by tool complexity | Compliance Manager | Medium | Medium | Guided workflows; clear documentation; training sessions |
| Engineering teams resist evidence task assignments | Engineering Manager | Medium | Medium | Automate evidence pre-population; minimize manual effort per task |
| Tool developers leave project | Tool Developers | Low | Medium | Comprehensive documentation; clean architecture; knowledge sharing |

## Change Management

### Stakeholder Readiness Assessment

| Stakeholder | Current Readiness | Required Readiness | Gap | Actions |
|---|---|---|---|---|
| Compliance Manager | High | High | Minimal -- already the primary user | Continue co-design; formalize training materials |
| Security Engineer | Medium | High | Needs deeper integration with workflow | Pair programming sessions; custom tool development support |
| DevOps Engineer | Low | Medium | Needs CI/CD integration examples | Provide pipeline templates; pilot program |
| CISO | Medium | Medium | Needs dashboard/reporting capabilities | Quarterly briefing format; compliance posture metrics |
| External Auditor | Low | Low | Needs evidence format familiarity | Pre-audit evidence sample sharing |

### Adoption Strategy

1. **Champions**: Compliance Manager and Security Engineer -- leverage as internal advocates; involve in design decisions; showcase their success to other teams
2. **Early Adopters**: DevOps team pilot with non-critical pipelines; Infrastructure team for Terraform scanning validation
3. **Training Plan**: CLI quickstart guide for new users; advanced tool development documentation for power users; evidence quality review training for compliance staff
4. **Support Structure**: Dedicated Slack channel; GitHub Issues for bug reports; weekly office hours during initial rollout

## Monitoring and Review

### Engagement Metrics

- Stakeholder meeting attendance: target 80%+
- Feedback response rate: target 70%+
- Evidence quality satisfaction scores: target 4/5+
- Issue resolution time: target 3 business days

### Review Schedule

- Weekly: Team-level stakeholder check-ins (Compliance Manager, Security Engineer)
- Monthly: Broader stakeholder engagement review (Engineering Manager, DevOps)
- Quarterly: Full stakeholder analysis update (CISO briefing, auditor feedback)
- Per audit cycle: External auditor feedback collection

---

*This is a living document. Update regularly as the stakeholder landscape changes, particularly after audit cycles and major feature releases.*
