---
title: "Glossary of Terms and Definitions"
type: "reference"
category: "glossary"
tags: ["glossary", "definitions", "terminology", "compliance", "reference"]
related: ["[[naming-conventions]]", "[[data-formats]]", "[[soc2]]", "[[iso27001]]"]
created: 2025-09-10
modified: 2025-09-10
status: "active"
---

# Glossary of Terms and Definitions

## Overview

This glossary defines technical terms, compliance terminology, and domain-specific concepts used throughout GRCTool and its documentation. It serves as a reference for developers, compliance professionals, and auditors working with the system.

## General Terms

### **Architecture Decision Record (ADR)**
A document that captures important architectural decisions made during the project, including context, decision, and consequences. Used for maintaining decision history and rationale. See [[adr-template]] for the standard format.

### **Audit Period** 
The timeframe for which evidence is collected and evaluated during a compliance audit. Typically spans 12 months for SOC 2 Type II audits and varies for other frameworks.

### **Audit Readiness**
The state of having complete, validated evidence packages ready for external auditor review. Includes automated evidence collection, manual documentation, and quality validation.

### **Automation Level**
Classification of how much of an evidence collection process can be automated:
- **Full**: 95%+ automated (minimal human intervention)
- **High**: 75-95% automated (some manual review required)  
- **Medium**: 50-75% automated (significant manual work needed)
- **Low**: 25-50% automated (mostly manual with tool assistance)
- **Manual**: <25% automated (primarily manual processes)

### **Batch Processing**
Executing multiple operations together as a group, typically for performance optimization. Example: `grctool evidence generate --all --parallel` processes multiple evidence tasks concurrently.

## Compliance and GRC Terms

### **Compliance Framework**
A structured set of guidelines, standards, and requirements for organizational governance, risk management, and compliance. Examples: SOC 2, ISO 27001, NIST Cybersecurity Framework, PCI DSS.

### **Control**
A safeguard or countermeasure designed to preserve the confidentiality, integrity, and availability of information. Controls can be preventive, detective, or corrective in nature.

### **Control Objective** 
High-level statement of desired outcome that controls are designed to achieve. Example: "Ensure logical access to information and system resources is restricted to authorized users."

### **Control Testing**
The process of evaluating whether controls are operating effectively. Can be performed through inquiry, observation, inspection, or re-performance of control activities.

### **Evidence**
Documentation or artifacts that demonstrate the implementation and effectiveness of controls. Can be automated (tool-generated) or manual (human-created).

### **Evidence Task**
A specific requirement to collect documentation or artifacts that support control implementation. Referenced by standardized IDs like `ET-0001`.

### **GRC (Governance, Risk, and Compliance)**
An integrated approach to managing governance, risk management, and compliance activities across an organization.

### **Risk Assessment**
Systematic evaluation of potential risks to organizational assets, operations, and objectives. Includes risk identification, analysis, and evaluation phases.

### **Statement of Applicability (SoA)**
Document required for ISO 27001 certification that identifies which controls from Annex A are applicable, implemented, or excluded, with justification.

## SOC 2 Terminology

### **Common Criteria**
The foundational security criteria that apply to all SOC 2 audits, focusing on the design and implementation of controls relevant to security.

### **Service Organization**
An entity that provides services to user entities that are relevant to those user entities' internal control over financial reporting.

### **SOC 2 Type I**
Report on controls at a service organization relevant to security, availability, processing integrity, confidentiality, or privacy as of a specific date.

### **SOC 2 Type II**
Report on controls at a service organization relevant to security, availability, processing integrity, confidentiality, or privacy throughout a specified period.

### **Trust Service Criteria (TSC)**
The criteria established by the AICPA for evaluating controls relevant to security, availability, processing integrity, confidentiality, and privacy.

**TSC Categories:**
- **Security (Common Criteria)**: Foundation for all SOC 2 audits
- **Availability**: System availability for operation as committed
- **Processing Integrity**: Complete, valid, accurate, timely, and authorized processing
- **Confidentiality**: Information designated as confidential is protected
- **Privacy**: Personal information collected, used, retained, disclosed, and disposed of properly

## ISO 27001 Terminology

### **Annex A**
Reference control objectives and controls from ISO/IEC 27002, providing 114 security controls organized into 14 categories.

### **Corrective Action**
Action taken to eliminate the cause of detected nonconformity or other undesirable situation to prevent recurrence.

### **Information Security Management System (ISMS)**
Framework of policies and procedures that includes all legal, physical and technical controls involved in an organization's information risk management processes.

### **Internal Audit**
Systematic, independent examination to determine whether activities and related results conform to planned arrangements.

### **Management Review**
Formal evaluation by top management of the status and adequacy of the ISMS in relation to information security policy and objectives.

### **Nonconformity**
Non-fulfillment of a requirement, whether specified in the standard or determined by the organization.

### **Plan-Do-Check-Act (PDCA)**
Management methodology for the continual improvement of processes and systems, fundamental to ISO 27001.

### **Preventive Action**
Action taken to eliminate the cause of potential nonconformity or other potential undesirable situation to prevent occurrence.

### **Risk Treatment**
Process of selecting and implementing appropriate measures to modify risk through risk avoidance, risk optimization, risk transfer, or risk retention.

## Technical Terms

### **API (Application Programming Interface)**
Set of protocols, routines, and tools for building software applications. GRCTool uses APIs to integrate with external systems like GitHub, AWS, and Tugboat Logic.

### **Authentication**
Process of verifying the identity of a user, device, or system. GRCTool uses browser-based authentication with Tugboat Logic instead of API keys.

### **Checksum**
A value calculated from data to detect errors or changes. GRCTool uses SHA-256 checksums to verify evidence integrity.

### **CI/CD (Continuous Integration/Continuous Deployment)**
Development practice where code changes are automatically built, tested, and deployed to production environments.

### **CLI (Command Line Interface)**
Text-based user interface for interacting with computer programs. GRCTool provides a comprehensive CLI built with Cobra framework.

### **Correlation ID**
Unique identifier used to track requests across multiple systems or components for debugging and audit purposes.

### **JSON (JavaScript Object Notation)**
Lightweight data interchange format used for configuration files, API responses, and evidence storage in GRCTool.

### **OAuth**
Open standard for access delegation, commonly used for token-based authentication without exposing passwords.

### **REST API (Representational State Transfer)**
Architectural style for designing networked applications using HTTP methods for communication between systems.

### **VCR (Video Cassette Recorder)**
Testing technique that records HTTP interactions for playback during testing, ensuring consistent and offline-capable tests.

### **Webhook**
HTTP callback that occurs when specific events happen, enabling real-time notifications between systems.

### **YAML (YAML Ain't Markup Language)**
Human-readable data serialization standard used for configuration files in GRCTool.

## GRCTool-Specific Terms

### **Evidence Assembly**
The process of collecting, validating, and formatting evidence from multiple sources into audit-ready packages.

### **Evidence Envelope**
Standardized JSON structure containing evidence content plus metadata (correlation ID, timestamps, tool information, quality metrics).

### **Evidence Generator**
Tool or service that automatically creates evidence documentation from various data sources (infrastructure, repositories, policies).

### **Evidence Quality Score**
Numerical rating (0-100) indicating the completeness, accuracy, and reliability of collected evidence.

### **Evidence Source**
System or location from which evidence is collected. Examples: GitHub repositories, Terraform configurations, policy documents, monitoring logs.

### **Evidence Validator**
Component that checks evidence for completeness, accuracy, and compliance with quality standards.

### **Path Safety**
Security mechanism that prevents directory traversal attacks by validating and restricting file system access to approved paths.

### **Reference ID Normalization**
Process of converting various input formats (ET1, ET-1, 328001) to standardized reference IDs (ET-0001).

### **Tool Registry**
System for discovering, registering, and executing evidence collection tools with consistent interfaces and metadata.

### **Tugboat Integration**
Connection to Tugboat Logic GRC platform for synchronizing organizational data (policies, controls, evidence tasks).

## File and Data Terms

### **Cache Directory**
Local storage location for temporary data that can be regenerated, used to improve performance and reduce API calls.

### **Data Directory**
Primary storage location for synchronized organizational data (policies, controls, evidence tasks).

### **Evidence Record**
Complete documentation package for a specific evidence task, including content, sources, metadata, and validation information.

### **File Sanitization**
Process of removing or replacing characters in filenames that are not safe for file systems (spaces, special characters).

### **JSON Schema**
Vocabulary for validating the structure of JSON documents, used to ensure data consistency across GRCTool.

### **Tugboat ID**
Unique numeric identifier assigned by Tugboat Logic to organizational entities (policies, controls, evidence tasks).

## Tool Categories

### **Access Control Tools**
Evidence collection tools focused on user access management, permissions, and identity systems:
- `github-permissions`: Repository access analysis
- `github-deployment-access`: Deployment environment controls
- `identity-evidence-collector`: Identity management systems (planned)

### **Infrastructure Analysis Tools**
Tools for analyzing infrastructure configuration and security:
- `terraform-scanner`: Infrastructure security analysis
- `terraform-hcl-parser`: Detailed configuration parsing

### **Security Analysis Tools**
Tools focused on security controls and monitoring:
- `github-security-features`: Repository security configuration
- `github-workflow-analyzer`: CI/CD security analysis  
- `security-ops-evidence-collector`: Security operations evidence (planned)

### **Document Management Tools**
Tools for collecting and analyzing documentation:
- `google-workspace`: Google Workspace document analysis
- `policy-summary-generator`: Policy documentation
- `evidence-generator`: Multi-source evidence generation

### **Utility Tools**
Supporting tools for file operations and data management:
- `storage-read`: Safe file reading operations
- `storage-write`: Safe file writing operations
- `name-generator`: Filesystem-safe name generation

## Compliance Control Categories

### **Access Control Categories**
- **Logical Access Controls**: Software-based access restrictions
- **Physical Access Controls**: Physical security measures
- **Privileged Access Management**: Administrative access controls
- **User Access Reviews**: Periodic access validation

### **Change Management Categories**
- **Development Controls**: Software development lifecycle security
- **Deployment Controls**: Production deployment safeguards
- **Emergency Changes**: Expedited change procedures
- **Change Documentation**: Change tracking and approval records

### **Monitoring Categories**
- **Security Event Monitoring**: Automated security event detection
- **Performance Monitoring**: System performance and availability tracking
- **Compliance Monitoring**: Continuous compliance status monitoring
- **Audit Logging**: Comprehensive activity logging for audit trails

### **Risk Management Categories**
- **Risk Assessment**: Systematic risk identification and analysis
- **Risk Treatment**: Risk mitigation and control implementation
- **Risk Monitoring**: Ongoing risk level assessment
- **Incident Response**: Security incident handling procedures

## Quality and Validation Terms

### **Accuracy Score**
Measure of how correct and precise the evidence is, typically expressed as a percentage or ratio.

### **Completeness Score**
Measure of how much of the required evidence has been collected, expressed as a percentage.

### **Cross-Reference Validation**
Process of verifying that evidence references align correctly across different systems and documents.

### **Evidence Gap**
Missing or insufficient evidence required for compliance demonstration.

### **Quality Threshold**
Minimum acceptable score for evidence quality metrics (typically 85% for automated evidence).

### **Validation Rules**
Automated checks applied to evidence to ensure it meets quality and completeness requirements.

## Process and Workflow Terms

### **Audit Trail**
Chronological record of system activities that provides documentary evidence of operations, procedures, or events.

### **Batch Operation**
Execution of multiple related operations as a group, often for performance optimization.

### **Evidence Collection Workflow**
Standardized process for gathering, validating, and packaging evidence for audit purposes.

### **Incremental Sync**
Data synchronization that only updates changed or new items since the last synchronization.

### **Parallel Processing**
Simultaneous execution of multiple operations to improve performance and reduce overall processing time.

## Error and Status Terms

### **Auth Status**
Current authentication state with external services:
- `authenticated`: Valid credentials available
- `unauthenticated`: No credentials available  
- `expired`: Credentials have expired
- `error`: Authentication error occurred

### **Data Source Status**
Indicates the primary source of data for evidence collection:
- `file_system`: Local files and directories
- `github_api`: GitHub REST API
- `tugboat_api`: Tugboat Logic API
- `google_api`: Google Workspace API
- `cached`: Previously cached data

### **Validation Status**
Result of evidence validation process:
- `validated`: Passed all validation checks
- `pending`: Validation in progress
- `failed`: Failed validation checks
- `warning`: Passed with warnings

## Integration and External Service Terms

### **GitHub Integration**
Connection to GitHub services for repository analysis, including permissions, security features, workflows, and deployment configurations.

### **Google Workspace Integration**
Integration with Google Workspace (formerly G Suite) for document analysis, including Drive, Docs, Sheets, and organizational data.

### **Tugboat Logic**
GRC platform that provides the master data source for policies, controls, and evidence tasks in organizational compliance programs.

### **VCR Cassette**
Recorded HTTP interactions stored for test playback, ensuring consistent and reliable testing without live API calls.

## Acronyms and Abbreviations

### **AICPA**
American Institute of Certified Public Accountants - organization that maintains SOC audit standards.

### **API**
Application Programming Interface

### **CISO** 
Chief Information Security Officer

### **CLI**
Command Line Interface

### **GDPR**
General Data Protection Regulation - European privacy regulation

### **GRC**
Governance, Risk, and Compliance

### **HIPAA**
Health Insurance Portability and Accountability Act

### **HTTP/HTTPS**
HyperText Transfer Protocol (Secure)

### **ISMS**
Information Security Management System

### **JSON**
JavaScript Object Notation

### **NIST**
National Institute of Standards and Technology

### **PCI DSS**
Payment Card Industry Data Security Standard

### **RBAC**
Role-Based Access Control

### **REST**
Representational State Transfer

### **SLA**
Service Level Agreement

### **SOC**
Service Organization Control

### **TSC**
Trust Service Criteria

### **UUID**
Universally Unique Identifier

### **VCR**
Video Cassette Recorder (testing pattern)

### **YAML**
YAML Ain't Markup Language

## Usage Context

### **CLI Context Terms**
- **Flag**: Command-line option (e.g., `--task-ref`)
- **Subcommand**: Secondary command (e.g., `evidence generate`)
- **Global Flag**: Option available to all commands (e.g., `--config`)

### **Audit Context Terms**
- **Audit Package**: Complete set of evidence prepared for auditor review
- **Audit Sample**: Subset of evidence selected by auditors for detailed testing
- **Management Letter**: Communication from auditors about findings and recommendations

### **Development Context Terms**
- **Hot Path**: Frequently executed code paths requiring optimization
- **Tech Debt**: Code or architecture compromises that may need future remediation
- **Integration Test**: Testing that verifies interaction between components

## References

- [[soc2]] - SOC 2 compliance framework guide for Trust Service Criteria definitions
- [[iso27001]] - ISO 27001 compliance framework guide for ISMS terminology  
- [[controls-mapping]] - Control-to-evidence mapping for compliance terms
- [[naming-conventions]] - Naming standards and reference ID formats
- [[data-formats]] - JSON schemas and data structure definitions
- [AICPA Trust Service Criteria](https://www.aicpa.org/interestareas/frc/assuranceadvisoryservices/trustdataintegritytaskforce.html)
- [ISO/IEC 27001:2022](https://www.iso.org/standard/27001) - Information Security Management Systems
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework) - Framework for cybersecurity risk management

---

*This glossary is maintained collaboratively and updated regularly. For term additions or corrections, see the [[helix/06-iterate/roadmap-feedback|Documentation Working Group]] process.*