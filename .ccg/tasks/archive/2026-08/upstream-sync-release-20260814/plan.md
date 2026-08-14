# Plan

1. Inspect repository state, project specs, deployment script, and remote reachability.
2. Fetch `upstream` and collect commit/file/schema/version differences.
3. Run Codex and Claude analysis in parallel and consolidate merge risks.
4. Merge `upstream/main`; resolve conflicts while preserving downstream behavior.
5. Regenerate generated code when required and run focused plus full verification.
6. Run Codex and Claude review in parallel; fix critical and relevant warning findings, then re-verify.
7. Commit and push `main` to `origin`.
8. Build and push DockerHub release tags and verify registry digests.
9. Deploy the release to `192.168.48.12` and `192.168.10.151`, then verify health, image, compose state, and logs.
10. Record the review, archive the task, commit, and push the archive.
