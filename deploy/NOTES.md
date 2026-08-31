# Implementation Notes and Decision Record

> Record only assumptions, decisions, and evidence from this submission. Reference specific files, jobs, commands, or runtime results. Keep the document concise and aim for no more than 1,000 words.


## 1. Key Assumptions

1. I treated the GitHub-hosted runner as the deployment boundary for this exercise rather than provisioning a persistent environment. This kept the solution demonstrable without requiring paid cloud infrastructure. If a persistent environment were required, I would deploy the same SHA-tagged image to the target platform and add environment-specific deployment and rollback steps.

2. I assumed `main` is the trusted integration branch. Pull requests only run validation, while publishing and deployment happen after a successful push to `main`. If the team uses release branches, tags, or environment approvals instead, I would move the publish/deploy conditions to that release boundary.

3. I assumed a single local application instance is sufficient for validating the required observability signals. Prometheus therefore scrapes `app:8080` through the Docker Compose network. In a multi-instance environment, I would replace the static target with the target discovery mechanism provided by the deployment platform and aggregate metrics across instances.


## 2. Delivery Path and Rollback

A pull request to `main` runs the `validate` job in `.github/workflows/ci.yml`. The job runs `go vet`, `go test -race -count=1 ./...`, compilation, and a Docker image build. Publishing and deployment are intentionally skipped for pull requests.

On a push to `main`, `validate` runs again. If it succeeds, the `publish` job authenticates to GHCR using `GITHUB_TOKEN`, builds the image, tags it as `ghcr.io/<repository>:<commit-sha>`, and pushes it to the registry.

The `deploy` job then pulls that exact SHA-tagged image rather than rebuilding it. It runs the container on the GitHub-hosted runner, waits for Docker health to report `healthy`, and verifies `/healthz` with `curl`. The deploy job cleans up the temporary container afterward, including when its runtime validation fails.

During validation, GHCR initially rejected the image reference because the GitHub owner contained uppercase characters. I normalized the repository component to lowercase in both `publish` and `deploy` and reran the pipeline successfully.

The deployable and rollback unit is the SHA-tagged container image. For a persistent target, rollback would mean redeploying the previous validated image SHA rather than rebuilding an earlier source revision.


## 3. Actual Validation / Investigation

I used `docker compose` to run the API, Prometheus, and Grafana together. Prometheus reported the `task-api` target as `UP`, and Grafana successfully queried the provisioned Prometheus datasource.

For the dashboard test, I expected successful API requests to produce a non-zero request rate, intentional 404s to produce failures, the request-duration histogram to produce P50/P95/P99 values, and created tasks to appear in the task-state metrics.

My initial short burst of traffic changed the task gauges and produced HTTP counters, but the request-rate and latency panels showed little useful activity. I checked the raw Prometheus metrics and PromQL and confirmed that requests and histogram observations were being recorded.

The application was therefore left unchanged. I repeated the experiment with traffic spread over approximately 60 seconds, continuously calling `GET /tasks` and generating an intentional `GET /tasks/999` 404 every fifth iteration. This crossed multiple 15-second Prometheus scrape intervals.

After subsequent scrapes, Grafana showed non-zero request and failure rates, P50/P95/P99 latency values, and the expected task state. The raw Prometheus queries matched the dashboard. I concluded that the initial result was caused by the traffic experiment relative to the scrape/rate windows, not broken instrumentation.


## 4. Engineering Trade-offs

### Temporary deployment vs. persistent infrastructure

I used the GitHub-hosted runner as the deployment target instead of provisioning cloud infrastructure. The deploy job still crosses a real artifact boundary: it pulls the SHA-tagged image from GHCR, starts it, waits for Docker health, and verifies `/healthz`.

The trade-off is that the environment disappears after the job and cannot demonstrate long-running availability or environment promotion. I would choose a persistent target if those behaviors were part of the requirements.

### Standard Prometheus client vs. custom metrics output

The starter application had a small custom Prometheus-compatible `/metrics` response. I replaced that path with the standard Prometheus Go client so HTTP counters, normalized route labels, histogram buckets, and task gauges use standard Prometheus metric types.

This introduced `github.com/prometheus/client_golang` and increased the final image size, but the measured image remained approximately 3.9 MB, well below the 15 MiB requirement. I validated the instrumentation with race-enabled tests, raw `/metrics` output, Prometheus queries, and Grafana. If dependency/image-size constraints were substantially tighter and only a few fixed metrics were needed, I would reconsider maintaining the smaller custom implementation.


## 5. Time, Omitted Work, and Next Steps

- **Actual time spent:** 4.5 hours
- **Deliberately omitted:** I did not provision a persistent cloud environment because the assignment allows a temporary demonstrable deployment target. I also did not add alerting, tracing, or additional production infrastructure beyond the required delivery and observability path.
- **With another 60 minutes:** I would pin the Prometheus and Grafana container versions for more reproducible local runs, add a small repeatable smoke/traffic test for the observability path, and improve rollback automation if deploying to a persistent target.


## 6. Use of AI

- **Transcript:** `deploy/ai-transcripts/chatgpt-transcript.md`
- **Tool/model:** ChatGPT (GPT-5.6 Sol)

I used AI as an engineering review, implementation, and debugging aid while working through the container, CI/CD, and observability tasks. I reviewed and validated generated suggestions against the repository requirements and runtime behavior before keeping them.

One AI-assisted output I corrected was the initial CI/CD workflow. The generated workflow used ${{ github.repository }} directly when constructing the GHCR image reference. During an actual pipeline run, authentication succeeded but Docker rejected the image tag because the repository component contained uppercase characters. I investigated the failure, confirmed the GHCR/OCI naming constraint, updated both the publish and deploy jobs to normalize the repository name to lowercase, and reran the pipeline successfully. This validation caught an issue in the AI-assisted implementation before I treated the workflow as complete.