# Agent-Ops Standards Map
## Informative alignment v0.4.0

**Source revision aom-03-r12 | English precedence**

> This document is an informative map; the named external sources remain authoritative for their own requirements.

## 1. Interpretation

Mapping means that an Agent-Ops control may provide evidence relevant to an external concern. It MUST NOT be read as certification, legal advice, or proof of complete conformity. Every claim remains subject to verification on the specific deployment.

The mapping to ISO/IEC 42001 was produced by Git in Sky from the risk-to-control map published in the OWASP Agentic Skills Top 10. OWASP has neither produced nor endorsed such a mapping for Agent-Ops.

## 2. Detailed control map

| Agent-Ops section | ISO/IEC 42001, Annex A | External risk catalog |
| --- | --- | --- |
| 7 Evidence, 16.5 agent-artifact coverage record | A.6.2.4 Verification and validation | AST08 Poor Scanning; AISVS C11 (adversarial testing) |
| 15 declared and observed state, agent artifact metadata | A.6.2.7 Technical documentation | AST04 Insecure Metadata |
| 16 Project Operations Harness, 16.2 machine-readable permissions | A.6.2.5 Deployment | AST03 Over-Privileged Skills; AISVS C5, C9 |
| 16.1 trust hierarchy, 20.1 security, 20.3 context lifecycle | A.6.2.6 Operation and monitoring | AST05 Untrusted External Instructions; AST07 Update Drift |
| 16.5 agent artifact inventory | A.10.3 Suppliers | AST02 Supply Chain Compromise |
| 18 Governance Mesh, 18.1 admission and outcome receipts | A.2.2 AI policy | AST09 No Governance; Regulation (EU) 2024/1689, Article 12 |
| 9 Plan, 11 Controlled Change and gated Executor, 20.1 credential separation | A.4.5 System and computing resources | AST06 Weak Isolation |
| 20.1 applied security controls and no-mutation invariant | A.6.2.4 Verification and validation | AST01 Malicious Skills; AST10 Cross-Platform Reuse |

## 3. Standards and regulations

- ISO/IEC 42001:2023, Information technology - Artificial intelligence - Management system: <https://www.iso.org/standard/42001>
- Regulation (EU) 2024/1689 (EU AI Act), Article 12 on record-keeping and Article 113 on application dates: <https://eur-lex.europa.eu/eli/reg/2024/1689/oj/eng>
- NIST AI Risk Management Framework, GOVERN function: <https://www.nist.gov/itl/ai-risk-management-framework>
- SLSA, provenance and build-integrity framework: <https://slsa.dev/>

## 4. Risk catalogs and verification standards

- OWASP Agentic Skills Top 10 (AST01-AST10), version 1.0, 2026: <https://owasp.org/www-project-agentic-skills-top-10/>; repository: <https://github.com/OWASP/www-project-agentic-skills-top-10>
- OWASP AI Security Verification Standard (AISVS), which verifies whether a control is implemented: <https://github.com/OWASP/AISVS>
- OWASP Top 10 for Agentic Applications (Agentic Security Initiative, ASI): <https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/>
- OWASP MCP Top 10, covering Model Context Protocol servers—an adjacent surface not covered by AST: <https://owasp.org/www-project-mcp-top-10/>
- OWASP Gen AI Security Project, including the Top 10 for LLM Applications: <https://genai.owasp.org/>
- CSA MAESTRO, a seven-layer threat model for agentic systems: <https://cloudsecurityalliance.org/blog/2025/02/06/agentic-ai-threat-modeling-framework-maestro>; repository: <https://github.com/CloudSecurityAlliance/MAESTRO>

## 5. Analyzed systems and security case sources

The following systems and publications informed comparative architecture and threat-model analysis. Their inclusion is informative: it is not an endorsement, a normative dependency, or evidence that a named system conforms to Agent-Ops.

- HolmesGPT, an open-source SRE agent for production-incident investigation: repository <https://github.com/HolmesGPT/holmesgpt>; documentation <https://holmesgpt.dev/latest/>.
- Warden, an identity-aware gateway that mediates agent access to external systems: repository <https://github.com/stephnangue/warden>; documentation <https://wardengateway.com/>.
- Hugging Face, "Anatomy of a Frontier Lab Agent Intrusion: A Technical Timeline of the July 2026 Incident": <https://huggingface.co/blog/agent-intrusion-technical-timeline>.
- Docker, "A new security baseline for enterprise agentic adoption": <https://www.docker.com/blog/a-new-security-baseline-for-enterprise-agentic-adoption/>.
- Cloudflare, "The Agent Access Model": <https://blog.cloudflare.com/the-agent-access-model/>.
- Ganesh Gurudu, "Building an AI Agent That Runs Your SRE Operations - What I Learned, What Works, and How You Can Do It Too," 8 April 2026: <https://blog.stackademic.com/building-an-ai-agent-that-runs-your-sre-operations-what-i-learned-what-works-and-how-you-can-do-8a3801124bdc>. The article is an informative implementation-experience source; Agent-Ops does not adopt its product stack, risk scale, deployment topology, or accuracy claims.
- Ali Shazal and Matthew Koen, Anthropic, "A guide to the anatomy of effective commerce agents," 2 September 2026: <https://claude.com/blog/the-anatomy-of-effective-commerce-agents>. The article is an informative source outside the operations domain; Agent-Ops does not adopt its commerce architecture, model selection, latency techniques, or deployment stack.

## 6. Caveats

- OWASP AIVSS (AI Vulnerability Scoring System, <https://aivss.owasp.org/>) is not applied in Agent-Ops: version 1.0 is unreleased as of this edition, and the OWASP Agentic Skills Top 10 itself refrains from assigning severity ratings until it ships.
- The incident observations, campaigns, and scanner-bypass results underpinning the OWASP Agentic Skills Top 10 are not reproduced here and have not been independently verified. Primary-source references are given in the OWASP AST itself and in the separate applicability analysis published alongside this methodology.
- Conformance with an external risk catalog means the methodology addresses the named risk, not that the risk is eliminated in a given deployment. Elimination is proven by run evidence, not by text.

## 7. Evidence use

An implementation SHOULD record the exact external edition, applicable clause, local control, and evidence identity. A generic standards badge is insufficient.

## 8. Change control

External sources MAY change independently. A mapping update creates a new source revision and does not rewrite prior evidence.

## 9. Legal boundary

The accountable organization MUST determine legal applicability and retain its decision. Agent-Ops supplies lifecycle evidence, not legal authority.
