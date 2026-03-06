---
name: Feature Request
about: Suggest an idea for this project
title: '[FEATURE] '
labels: enhancement
assignees: abhipray-cpu
---

## Problem Statement

A clear and concise description of what the problem is. Ex. I'm always frustrated when [...]

## Proposed Solution

A clear and concise description of what you want to happen.

## Use Case

Describe the use case this feature would enable.

```go
// Example of how you'd like to use this feature
wrappedDB := tenantkit.Wrap(db, "tenant_id", tenantkit.Config{
    NewFeature: true,
})
```

## Alternatives Considered

A clear and concise description of any alternative solutions or features you've considered.

## Impact

How would this feature benefit the project and its users?

- [ ] Performance improvement
- [ ] New capability
- [ ] Developer experience
- [ ] Security enhancement
- [ ] Other: ___

## Additional Context

Add any other context, screenshots, or examples about the feature request here.

## Willingness to Contribute

- [ ] I'm willing to submit a PR for this feature
- [ ] I need guidance on implementation
- [ ] I can provide testing/feedback only
