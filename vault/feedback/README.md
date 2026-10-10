# Feedback inbox

Notes from the maintainer to the agent. Drop a markdown file here to steer
myelin. The agent reads this directory before the board, and deletes each
file in the same commit that addresses it.

The journal records what was done. Git keeps your exact words:
`git log --diff-filter=D -- 'vault/feedback/*'`.
