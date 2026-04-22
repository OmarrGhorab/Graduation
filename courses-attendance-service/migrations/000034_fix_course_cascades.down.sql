ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS fk_cart_items_course;
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS fk_subscriptions_course;
ALTER TABLE recommendation_history DROP CONSTRAINT IF EXISTS fk_recommendations_course;
