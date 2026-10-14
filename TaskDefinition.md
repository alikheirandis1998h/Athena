# Senior DevOps/SRE Technical Assignment

## Automated Local Kubernetes Platform

Design and implement a reproducible, **Infrastructure-as-Code-managed Kubernetes platform** on local virtual machines.

The assessment focuses on outcomes and engineering judgment. Technology choices, architecture, and repository structure are intentionally left to the candidate and must be justified.

---

## 1. Scope

### Must be implemented

- Multi-node local virtual infrastructure
- Automated and idempotent Kubernetes provisioning
- Production-oriented node and container-runtime tuning
- Automatic deployment of a versioned HTTP application
- Basic operational visibility
- One meaningful failure exercise
- A single command that validates the implementation

### Must be designed and documented

Implementation is optional for:

- GitLab CI/CD
- Production deployment workflow
- Rollback workflow
- Production monitoring and alerting

### Optional bonuses

- HTTP/application-level autoscaling
- Monitoring and alerting implementation
- Production-oriented database backend
- Candidate-developed application

---

## 2. Platform Requirements

The environment must be recoverable from version control without manually preparing its virtual machines.

The candidate must select and justify the virtualization, configuration-management, Kubernetes bootstrap, container runtime, networking, ingress, storage, and access-control approaches.

The solution must:

- Use multiple Kubernetes nodes and support replicated workloads.
- Provide functional pod networking, cluster DNS, and ingress.
- Provide persistent storage where required by the workload.
- Control important software and dependency versions.
- Support repeatable creation, destruction, and recreation.
- Use modular and maintainable automation.
- Clearly define ownership when multiple automation systems are used.
- Avoid continuous management of the same resource by conflicting systems.

Automation must be idempotent. Repeated execution must not unexpectedly reset the cluster, interrupt healthy workloads, or recreate unchanged resources without justification.

Local simplifications are acceptable, including a non-HA control plane, but they must be identified along with the corresponding production design.

---

## 3. Production-Oriented Tuning

Apply and justify operating-system, container-runtime, and Kubernetes-node tuning appropriate to the available resources.

Relevant areas may include:

- Networking and packet forwarding
- Kernel modules and limits
- File descriptors
- Connection tracking
- Memory and disk pressure
- Image and log accumulation
- Time synchronization
- Kubelet resource reservations
- Eviction thresholds
- Graceful shutdown and recovery

For each important setting, explain:

1. The problem it addresses
2. Why the selected value is appropriate
3. Potential negative effects
4. How it would be selected or calculated in production
5. How its application is verified

> Generic tuning values copied without reasoning are insufficient.

---

## 4. Application

Automatically deploy an immutable, versioned HTTP application as part of the platform bootstrap.

The application must allow verification of:

- Liveness
- Readiness
- Current version
- Successful HTTP traffic

It must also:

- Run multiple replicas.
- Define resource requests and limits.
- Run without unnecessary privileges.
- Handle startup and termination gracefully.
- Route traffic only to healthy instances.
- Remain available during a normal rolling update.
- Be reachable through the selected ingress approach.
- Be traceable to an immutable artifact and source revision.

Mutable-only tags such as `latest` are not acceptable.

Writing the application and exposing application metrics are optional.

---

## 5. CI/CD, Deployment, and Rollback Design

> A working GitLab pipeline, production deployment workflow, and rollback implementation are **not required**.

Document a design specific to the submitted platform and detailed enough to implement without making new major architectural decisions.

### CI/CD

Describe:

- Pipeline triggers and blocking checks
- Application, infrastructure, and Kubernetes validation
- Secret, dependency, and image scanning
- Immutable image publication
- Release creation and environment promotion
- Approval and verification controls
- Credential protection
- Artifact traceability
- Failure handling

Explain how artifacts would be promoted without rebuilding and how build permissions would be separated from deployment permissions.

### Deployment

Describe:

- The source of desired state
- Resource ownership and drift handling
- Environment-specific configuration
- Secret delivery
- Deployment strategy
- Availability and health verification
- Promotion and approval controls
- Failed-deployment behavior

The same resource must not be continuously managed by conflicting systems.

### Rollback

Describe:

- How the last healthy release is identified
- How previous artifacts and configuration remain available
- What initiates rollback
- Which system performs it
- Required permissions and approvals
- How recovery is observed and verified
- What happens if rollback fails

Also address incompatible configuration, Kubernetes resource changes, and database migrations where applicable.

Rollback must not depend on rebuilding an old artifact, mutable tags, or undocumented manual reconstruction.

---

## 6. Operational Visibility

The implementation must make it possible to inspect:

- Kubernetes node, pod, and workload status
- Liveness and readiness state
- Application and container logs
- Kubernetes events
- Application version
- Relevant conditions during the failure exercise

A dedicated metrics, monitoring, dashboarding, or alerting platform is optional.

Document how production observability would address:

- Application and platform signals
- Metrics and log collection
- Dashboards and alerting
- Deployment health
- SLOs and error budgets
- Retention and access control
- Failure of the observability platform itself

---

## 7. Security

The solution must demonstrate:

- No plaintext credentials, tokens, private keys, or sensitive kubeconfigs in Git
- Least-privilege access
- Workloads without unnecessary privileges
- Traceable images and dependencies
- Protection of sensitive values from logs
- Documented trust and access boundaries

Clearly distinguish between:

- Controls implemented locally
- Controls proposed for production
- Risks accepted because of local constraints

---

## 8. Failure Exercise

Demonstrate at least one meaningful failure, such as:

- Application process or readiness failure
- Worker-node failure
- Resource exhaustion
- Ingress failure
- Dependency or storage failure

Document:

- Expected and observed behavior
- Detection method
- User impact
- Recovery mechanism
- Remaining risks
- How production behavior would differ

Kubernetes status, events, probes, and logs are sufficient. A monitoring platform is not mandatory.

---

## 9. Single-Command Acceptance Test

Provide one documented command that runs the complete automated acceptance test, for example:

```bash
make test
```

The command name and implementation are the candidate's choice.

It must return a clear pass/fail result, identify failed checks, and verify at least:

- Virtual machines are reachable.
- Expected Kubernetes nodes are healthy.
- Required cluster services are operational.
- Pod networking and cluster DNS work.
- The application is reachable through ingress.
- Expected application replicas are ready.
- Health checks work.
- The application reports an immutable version.
- Required node tuning is present.
- Workload resource and security controls are applied.
- No known plaintext secrets exist in the repository.

The evaluator must not need undocumented commands or manual inspection.

CI/CD, rollback, HPA, and monitoring checks are outside the required acceptance test.

---

## 10. Documentation

Documentation must include:

- Build, test, troubleshooting, destruction, and recreation instructions
- At least one architecture and traffic-flow diagram
- Important decisions, alternatives, and trade-offs
- Security and operational consequences
- Resource-ownership boundaries
- Local limitations and production evolution
- CI/CD, deployment, rollback, and production-observability designs

Clearly label capabilities as:

- **Implemented**
- **Designed but not implemented**
- **Out of scope**

The production discussion should address relevant topics such as high availability, failure domains, storage, capacity, secrets, backup and restore, upgrades, network security, observability, disaster recovery, and environment isolation.

---

## 11. Bonus Objectives

### Application-level autoscaling

Use an HTTP or application-level metric and demonstrate:

- Metric collection and autoscaler integration
- Scale-out under load
- Controlled scale-in
- Stable scaling boundaries
- Behavior when the metrics source is unavailable

CPU-only HPA receives partial credit.

### Monitoring and alerting

Implement monitoring or alerting that detects or explains a meaningful failure. Document ownership, retention, access, resource cost, and failure behavior.

### Production-oriented database

Implement a stateful database with:

- Persistent storage
- Secure credentials
- Resource and connection controls
- Schema-migration strategy
- Failure and disruption handling
- Backup and demonstrated restoration
- Defined recovery objectives

A database without a tested restore path does not qualify.

### Candidate-developed application

Additional credit is available for a small application with tests, structured logging, graceful shutdown, version information, secure container construction, and dependency timeout handling.

---

## 12. Constraints

The solution must not rely on:

- Manually prepared virtual machines
- Repeated manual bootstrap steps
- Mutable-only image tags
- Plaintext secrets
- Undocumented operator knowledge
- Conflicting resource ownership
- A single-node development cluster represented as multi-node
- Unsupported claims of production readiness

Third-party components are permitted when their versions, role, and operational implications are controlled and documented.

---

## 13. Evaluation

| Area | Weight |
|---|---:|
| Automation, reproducibility, and idempotency | 25% |
| Kubernetes and infrastructure architecture | 20% |
| Production-oriented tuning | 15% |
| Security and secrets management | 10% |
| Acceptance testing and failure handling | 10% |
| CI/CD, deployment, rollback, and observability design | 10% |
| Documentation and technical decisions | 10% |
| **Total** | **100%** |

Bonus work does not compensate for an incomplete core solution.

---

## 14. Demonstration

Be prepared to demonstrate:

1. Automated environment creation
2. Idempotent automation re-execution
3. The single-command acceptance test
4. Application access and version identification
5. Applied production tuning
6. The selected failure exercise
7. Major architecture decisions and trade-offs
8. Production evolution
9. Proposed CI/CD, deployment, rollback, and observability designs

A live GitLab pipeline, rollback, HPA, or monitoring platform is not required.
