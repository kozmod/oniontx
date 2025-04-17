# Contributing

### With issues:

+ Use the search tool before opening a new issue.
+ Please provide source code and commit sha (optional) if you found a bug.
+ Review existing issues and provide feedback or react to them.

###  With pull requests:

+ Open your pull request against `dev` branch.

  **a)** The pull request name format should correspond to format
    ```
    [#<issue reference>] <pull request name>
    ```
  Example  
    ```
    [#61] add `CONTRIBUTING.md`
    ```
  **b)** The pull request branch should contains tag (`bugfix`, `feature`, etc.) and correspond to format
    ```
    <tag>/<issue reference>_<additional description if required>
    ```
  Example
    ```
    feature/61_contributing_info
    ```
  **c)** The commit message should correspond to format
    ```
    [#<issue reference>] <commit message>
    ```
  Example
    ```
    [#61] add contributing info
    ```
  **Note**:

  `[#0]` uses for `technical` commit messages


+ Pull request should have no more than **one** commits, if not you should squash them.
+ It should pass all tests in the available continuous integration systems such as GitHub Actions.
+ Pull request should contain tests (added/modified) to cover your proposed code changes.
+ If pull request contains a new feature, all information about feature must be described in `README.md`.