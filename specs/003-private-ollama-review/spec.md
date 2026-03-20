# Feature Specification: Private local model review with repository context

**Feature Branch**: `003-private-ollama-review`  
**Created**: 2026-03-20  
**Status**: Draft  
**Input**: User description: "자신의 데이터를 네트워크 혹은 제 3자 API에 제공하는 것을 원하지 않는 개발자 고객들을 위해, 내부망에 설치된 Ollama 서버를 이용하면서 퀄리티를 유지시킬 수 있는 방법을 고안해야 한다. Ollama는 다른 provider와 다르게 스스로 코드베이스를 탐색하며 코드 컨벤션 등을 직접 확인할 수 있는 능력이 없기 때문에 스스로 코드 베이스를 탐색할 수 있게 만드는 tools 등을 아주 상세하게 제공해주어야 한다"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run pre-push review without sending repository data to third parties (Priority: P1)

A privacy-conscious developer wants to run the same kind of pre-push review they get from a managed assistant, but only using inference that runs on infrastructure their organization controls (for example, a server on the internal network). They configure the product so that staged changes and any supporting repository context are only sent to that trusted endpoint, not to external inference providers.

**Why this priority**: Without this outcome, the feature does not address the primary constraint (data residency and third-party avoidance).

**Independent Test**: Can be validated by configuring private mode, running a review, and confirming (via agreed observation methods such as network policy logs or controlled capture) that repository content does not leave the trusted boundary defined for this mode.

**Acceptance Scenarios**:

1. **Given** private mode is enabled and pointed at an organization-controlled endpoint, **When** the developer runs a pre-push review, **Then** the review completes using only that trust boundary for inference-related traffic for repository content.
2. **Given** private mode is enabled, **When** the developer inspects product documentation or in-product disclosure, **Then** they can understand what is allowed to leave their machine and what stays internal.

---

### User Story 2 - Keep review usefulness high without built-in codebase exploration (Priority: P2)

A developer uses a local model runtime that does not automatically browse the repository the way some managed assistants do. They still need review comments that reflect project conventions and surrounding code. The product supplies a rich, well-defined set of ways for the review workflow to pull in just enough repository context—such as locating files, reading contents, and finding patterns—so the model can ground its feedback in the actual codebase.

**Why this priority**: Privacy alone is not enough if reviews become generic or wrong; structured access to the repository closes the quality gap.

**Independent Test**: Can be validated by running reviews on repositories with known conventions and measuring whether feedback references concrete locations and convention-aligned suggestions compared to a baseline of “diff-only” review.

**Acceptance Scenarios**:

1. **Given** a repository with identifiable style or structure rules, **When** the developer runs a review in private mode, **Then** at least a minimum share of findings reference specific files or regions of the repository (not only the staged patch in isolation).
2. **Given** the developer needs to confirm a convention, **When** the review runs, **Then** the workflow can incorporate additional file reads or searches within configured limits without the developer manually pasting large chunks of code.

---

### User Story 3 - Operate safely when the private endpoint is missing or unhealthy (Priority: P3)

When the organization-hosted inference service is unreachable, misconfigured, or times out, the developer must not be silently blocked from pushing solely because of a tool failure, consistent with the product’s existing fail-open expectations for optional tooling.

**Why this priority**: Reliability of internal services varies; predictable behavior avoids surprise workflow breaks.

**Independent Test**: Simulate or observe endpoint failure and confirm the push path remains allowed with a clear warning rather than a hard failure attributable to the review tool.

**Acceptance Scenarios**:

1. **Given** the organization-hosted endpoint is unavailable, **When** the developer attempts a push that triggers review, **Then** the process exits in the non-blocking way defined for tool unavailability and surfaces a warning the developer can act on.
2. **Given** partial responses or timeouts from the endpoint, **When** the review runs, **Then** the developer receives a clear outcome (success with limitation, or fail-open) rather than an ambiguous hang.

---

### Edge Cases

- Very large repositories: limits on how much context can be pulled per step must be predictable so users understand why some files were not consulted.
- Sensitive paths: organization policy may exclude certain directories from automated reads; the product must respect configurable boundaries.
- Mismatch between model capability and task: reviews may be shallow or overconfident; users need transparency when context was truncated or tools were not invoked.
- Mixed trust environments: developer accidentally points private mode at an external endpoint; documentation and validation reduce misconfiguration risk.

## Assumptions

- “Third parties” means commercial or external inference services outside the organization’s agreed trust zone; internal network endpoints administered by the customer are acceptable.
- Organization-hosted inference may not browse the repository by default; bridging that gap is done by explicit, user-visible capabilities in the review workflow rather than by expecting the model to “just know” the tree layout.
- Quality is judged by grounded, actionable feedback and convention alignment—not by matching any single proprietary assistant verbatim.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST offer a configuration path where pre-push review of staged changes uses only organization-controlled inference endpoints chosen by the user, with no requirement to use external inference providers for that path.
- **FR-002**: When this private path is active, the product MUST make clear—before or during use—which categories of data may be sent where, so a privacy-conscious developer can confirm alignment with their policy.
- **FR-003**: The product MUST expose a documented set of capabilities that let the review workflow obtain structured repository context (for example, listing paths, reading file contents, and searching within the repository) within administrator- or user-defined boundaries.
- **FR-004**: Each capability in FR-003 MUST have documented limits (such as maximum size, depth, or number of invocations per review) that are understandable without reading implementation code.
- **FR-005**: The product MUST ensure that reviews using these capabilities still complete within a predictable upper time bound or degrade gracefully with a clear message when limits are hit.
- **FR-006**: If organization-hosted inference is unavailable, misconfigured, or times out, the product MUST follow the existing fail-open rule for tooling failures: the developer’s push MUST NOT be blocked solely for that reason, and a warning MUST explain what happened.
- **FR-007**: The product MUST allow operators to restrict which parts of the repository automated context gathering may touch, to support least-privilege and sensitive-tree exclusions.

### Key Entities *(include if feature involves data)*

- **Trust boundary configuration**: The user’s choice of which inference endpoints and network paths are considered organization-controlled for review traffic.
- **Staged change package**: The material under review for a push (and metadata needed to interpret it).
- **Repository context request**: A single bounded request for additional repository information made during a review (what to fetch, what limits apply, and what was returned or skipped).
- **Review finding**: A discrete item of feedback that may cite locations in the repository or the staged change, with a severity or category suitable for developer action.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In validation runs with private mode enabled, 100% of cases show no transmission of repository content to domains or endpoints outside the configured trust boundary (verified by an agreed observation method such as network capture or organizational proxy logs).
- **SC-002**: In a representative sample of repositories with documented conventions, at least 85% of generated findings include at least one concrete file path or scoped location reference (not only generic advice).
- **SC-003**: First-time setup of private mode (endpoint, trust confirmation, and a successful trial review) is completable in 15 minutes or less for a prepared administrator in a lab checklist.
- **SC-004**: When the organization-hosted endpoint is unavailable, 100% of simulated runs result in a non-blocking exit for the push path with a visible warning, matching the product’s fail-open policy for tool failures.
- **SC-005**: At least 90% of participants in a small internal usability review report that they understand what data can leave their environment when private mode is on (measured with a short post-task questionnaire).
