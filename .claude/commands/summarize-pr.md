---
name: summarize-pr
description: Summarize a CVO pull request with a brief description of changes and their impact on CVO and OCP.
parameters:
  - name: pr
    description: "PR number or URL (e.g., 1348 or https://github.com/openshift/cluster-version-operator/pull/1348)"
    required: true
---

You are helping a former CVO maintainer stay aware of changes to the Cluster Version Operator repository.

## Your Task

Given PR: "{{pr}}"

1. If the input is a full GitHub URL, extract the PR number. If it's just a number, use it directly against the `openshift/cluster-version-operator` repository.
2. Use `gh pr view <number> --repo openshift/cluster-version-operator` to fetch the PR title, body, and state.
3. Use `gh pr diff <number> --repo openshift/cluster-version-operator` to fetch the code diff.
4. If the diff is large or touches areas that need more context to understand, read the relevant source files in this repository to understand what the changed code does.
5. Write exactly two brief paragraphs (2-4 sentences each):
   - **First paragraph**: What is being changed and why. Be specific about the code areas affected (e.g., sync worker, Cincinnati client, resource application, preconditions, manifest ordering).
   - **Second paragraph**: What impact this change has on CVO behavior and, where relevant, on the broader OpenShift platform (upgrade reliability, update availability, cluster stability, operator reconciliation, etc.).

## Output Guidelines

- Keep it concise. Two short paragraphs, not an essay.
- Use plain language accessible to someone who knows CVO architecture but hasn't been reading the code recently.
- Don't list individual files or quote code. Summarize at the conceptual level.
- If the PR is trivial (typo fix, dependency bump, test-only change), say so briefly without forcing two full paragraphs.
