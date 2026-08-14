# Requirements

1. Fetch the latest `upstream/main` and merge it into local `main` while preserving downstream changes.
2. Regenerate Ent code when Ent schema or generated Ent files are affected.
3. Run repository-appropriate build and test verification before release.
4. Review the merged result with both Codex and Claude and resolve critical findings.
5. Commit the merge and push `main` to `origin`.
6. Build `zhangyc/sub2api` and push `latest`, the version from `backend/cmd/server/VERSION`, and the same version prefixed with `v` to DockerHub; verify a shared digest.
7. Run `deploy/deploy-to-server.sh` for both default targets, including `192.168.10.151`.
8. Verify each remote container is running and healthy, uses `zhangyc/sub2api:latest`, and has startup logs without panic.
9. Archive this CCG task, commit the archive, and push the archive commit.
