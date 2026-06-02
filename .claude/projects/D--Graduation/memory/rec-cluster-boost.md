---
name: rec-cluster-boost
description: How clusterContribution works in the recommendation service and why it reads 0.0 in the dev environment
metadata:
  type: project
---

In `recommendation-service`, `clusterContribution` is a collaborative-filtering boost: `_hybrid_score` weights it at 0.10. The signal is read from Postgres `cluster_metadata.top_courses` (NOT from the Qdrant `clusters` collection, which only stores a vestigial truncated centroid). The flow: KMeans clusters users → `_aggregate_cluster_preferences` ranks courses each cluster's members engaged with → `ClusterService.get_course_affinity_for_user` returns `{courseId: score}` → `search_relevant_courses` applies it per candidate.

**Why it reads 0.0 in dev:** the dev environment has exactly 1 user (`ab5d0830-...`) whose analytics profile has `AllAnalytics_count: 0` (no enrollments/engagement). With one user and no behavioral records there is nothing to aggregate, so `top_courses=[]` and every boost is 0.0. This is a DATA limitation, not a code bug — verified working end-to-end with synthetic multi-user data (3 similar users → same cluster → a non-enrolled course got `clusterContribution=0.0577`).

**How to apply:** to see a non-zero contribution you need ≥2 users sharing a cluster where at least one enrolled in a course the target user hasn't. Clustering runs on startup, on `POST /api/v1/recommendations/refresh`, and via `POST /api/v1/recommendations/clusters/rebuild`. `gather_known_user_ids` unions `RecommendationHistory.user_id` with `DISTINCT user_id FROM public.user_course_analytics` (same `graduation` DB) so any user with learning analytics is clusterable. Cluster count targets ~4 users/cluster (`round(n/4)`); features are standardized before KMeans; cluster-level metadata is upserted once per distinct cluster (not per user — doing it per user caused a UniqueViolation).

**Seeder:** `scripts/seed_recommendation_cohort.py` creates a 12-user cohort in 3 subject personas (mobile_ux / data_science / cloud) with `enrollments` + `user_course_analytics` rows and deliberate gaps; it reuses the existing catalog and places the demo student `student@example.com` (`ab5d0830-...`) in mobile_ux. Run it inside the recommendation-service container (has psycopg2+bcrypt): `docker compose cp` it in, then `MSYS_NO_PATHCONV=1 docker compose exec -e SEED_DB_URL=postgresql://graduation:graduation_secret@postgres:5432/graduation recommendation-service python /tmp/seed_cohort.py`. After seeding + restart, Adham gets `clusterContribution` ~0.06–0.08 on courses his cluster-mates took. The old `scripts/seed_data.py` and `reseed_all.py` never seed `user_course_analytics`, which is why analytics were empty.
