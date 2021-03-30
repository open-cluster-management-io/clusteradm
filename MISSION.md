[comment]: # ( Copyright Contributors to the Open Cluster Management project )**Table of Contents**

## Mission Statement

We seek to form an open community around multicluster and multicloud scenarios for containerized applications. We propose to anchor the initial community around  github.com/open-cluster-management and open-cluster-management.io.

We seek to add value to the community by a focused effort around many aspects of how users are deploying and managing Kubernetes clusters today. We seek to engage other parts of the community and both contribute to pre-existing efforts and invite contributors in those communities to cross-collaborate as part of this project.

We are initially interested in the following lifecycles associated with expanding adoption of Kubernetes:

1. Cluster Lifecycle. How are clusters provisioned, upgraded, registered, scaled out or in and decommissioned?
2. Policy & Configuration Lifecycle. How are clusters configured, audited, secured, access controlled, managed for quota or cost?
3. Application Lifecycle. How are containerized or hybrid applications delivered across one or more clusters? How are those applications kept current with ongoing changes?
4. Observability. How does a user understand the health of their cluster fleet? How does a user understand the health of distributed applications? How does a user search available clusters or applications and diagnose problems when they occur?

Our initial goals for the project are to define API and reference implementations for common use cases that we have observed as users grow their adoption of Kubernetes:

- Define API for cluster registration independent of cluster CRUD lifecycle.
- Define API for work distribution across multiple clusters.
- Define API for dynamic placement of content and behavior across multiple clusters.
- Define API for policy definition to ensure desired configuration and security settings are auditable or enforceable.
- Define API for distributed application delivery across many clusters and the ability to deliver ongoing updates.
- Define API to collect cluster and application health metrics and alerts across multiple clusters.

We expect that over time, the project will make sense to contribute to an appropriate foundation for stewardship. In the meantime, we intend to engage and contribute where similar use cases are under active discussion in the community including the Kubernetes SIG-Multicluster and SIG-Policy workgroups, among others.

## Contributor Code of Conduct
The Open Cluster Management project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). The English text of the CNCF Code of Conduct is made available here for reference. Additional [language translations](https://github.com/cncf/foundation/blob/master/code-of-conduct.md) are available.

“As contributors and maintainers of this project, and in the interest of fostering an open and welcoming community, we pledge to respect all people who contribute through reporting issues, posting feature requests, updating documentation, submitting pull requests or patches, and other activities.

We are committed to making participation in this project a harassment-free experience for everyone, regardless of level of experience, gender, gender identity and expression, sexual orientation, disability, personal appearance, body size, race, ethnicity, age, religion, or nationality.
Examples of unacceptable behavior by participants include:
The use of sexualized language or imagery
Personal attacks
Trolling or insulting/derogatory comments
Public or private harassment
Publishing others' private information, such as physical or electronic addresses, without explicit permission
Other unethical or unprofessional conduct.
Project maintainers have the right and responsibility to remove, edit, or reject comments, commits, code, wiki edits, issues, and other contributions that are not aligned to this Code of Conduct. By adopting this Code of Conduct, project maintainers commit themselves to fairly and consistently applying these principles to every aspect of managing this project. Project maintainers who do not follow or enforce the Code of Conduct may be permanently removed from the project team.
This code of conduct applies both within project spaces and in public spaces when an individual is representing the project or its community.”
Instances of abusive, harassing, or otherwise unacceptable behavior in Open Cluster Management may be reported by contacting [NEED POINT OF CONTACT].” [[Reference](https://github.com/cncf/foundation/blob/master/code-of-conduct.md)]

## Getting Involved

Anyone who is interested in getting involved is welcome to contribute in a number of ways:

Join the recurring meeting forums (see below) to provide input as a stakeholder and help validate proposed use cases.
Suggest enhancements via github.com/open-cluster-management/enhancements for consideration to the community.
Contribute to development via Pull Request for new enhancements or defect fixes.

Suggested API and implementations will be accepted in accordance with the broad use cases outlined above. Our goal is to reserve the Kubernetes API Group open-cluster-management.io for well-reviewed and widely supported features.

## Community Meeting Forum

To ensure opportunities for broad user contributions, a public forum will be hosted to demonstrate new capabilities, solicit feedback and offer a forum for real time Q&A.
Meeting recordings will be posted to a YouTube channel for offline viewing.

- Public Agenda (Link TBD)
- First Monday of the month
  - 5-6 PM US Pacific / 8-9 PM US Eastern / 1-2 AM GMT / 8-9 AM (Tuesday) China
- Third Monday of the month
  - 8-9 AM US Pacific / 11 AM - 12 PM US Eastern / 4-5 PM GMT / 8-9 PM (Monday) China

## Community Architecture Forum

Technical discussions will also be held in an open forum. The topics for the Community Architecture Forum will cover technical review of APIs and relevant implementation details.
Meeting recordings will be posted to a YouTube channel for offline viewing.


- Public Agenda (Link TBD)
- Second Monday of the month
  - 5-6 PM US Pacific / 8-9 PM US Eastern / 1-2 AM GMT / 8-9 AM (Tuesday) China
- Fourth Monday of the month
  - 8-9 AM US Pacific / 11 AM -12 PM US Eastern / 4-5 PM GMT / 8-9 PM (Monday) China

## Communication

A public [Google Group](https://groups.google.com/g/open-cluster-management) has been created to facilitate offline discussion outside of Github issues or Pull Requests.

The project team has chosen to defer a new Slack team until we get enough critical mass in the community to warrant joining another Slack team or workspace.

## Governance

* **Committees** The project will initially have a 3-person Bootstrap Steering Committee. The present steering
  committee is a bootstrap committee and we want to work towards a future state where there is community representation and community determination of the steering committee members. In that future state, the steering committee size may be expanded to meet the needs of the community.

* **Special Interest Group (SIG)** are persistent open groups that focus on a part of the project.
  SIGs must have open and transparent proceedings.
  Anyone is welcome to participate and contribute provided they follow the Code of Conduct.

  The project has a bootstrap [sig-architecture](sig-architecture) to provide oversight and guidance on API and architectural aspects of the project to ensure a consistent and robust technical foundation for the project. More SIGs are expected to be established with the evolution of the project.

## Public Roadmap




## Security Response

Please see https://github.com/open-cluster-management/community/blob/master/SECURITY.md.