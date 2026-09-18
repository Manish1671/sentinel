# infrastructure/docker

Application Dockerfiles live next to each service (`apps/api`, `apps/web`, `services/*`). Root `docker-compose.yml` remains the local inner loop.

Kubernetes and AWS do **not** replace Compose. Image build notes: [kubernetes/README.md](../kubernetes/README.md).
