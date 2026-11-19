# Working on Promptherder

This repository is retired and archived. The README points to its plugin
successor; archived planning documents remain available at tag `v1.0.2`.

- Merging is a human task. Agents may prepare feature branches and pull requests;
  never merge or push to the default branch.
- Versions live in `VERSION`. The existing auto-tag workflow runs tests, tags and
  publishes after a version-bumped PR merges. Patch is for fixes and improvements,
  minor for features. Never bump major without the user's explicit approval.
  Do not publish from a dirty working tree.
